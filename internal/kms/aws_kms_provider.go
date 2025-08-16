package kms

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/pkg/execution"
)

const (
	awsKmsTimeout    = 5 * time.Second
	maxRetries       = 3
	initialBackoff   = 100 * time.Millisecond
	maxBackoff       = 1 * time.Second
)

type AWSKMSProvider struct {
	client    *kms.Client
	kmsKeyARN string
}

func NewAWSKMSProvider(cfg aws.Config, kmsKeyARN string) *AWSKMSProvider {
	return &AWSKMSProvider{
		client:    kms.NewFromConfig(cfg),
		kmsKeyARN: kmsKeyARN,
	}
}

func (p *AWSKMSProvider) EncryptDEK(ctx context.Context, plaintextDEK []byte, key *domain.Key) ([]byte, error) {
	return execution.WithRetry(ctx, maxRetries, initialBackoff, maxBackoff, func(ctx context.Context) ([]byte, error) {
		return execution.WithTimeout(ctx, awsKmsTimeout, func(ctx context.Context) ([]byte, error) {
			input := &kms.EncryptInput{
				KeyId:     &p.kmsKeyARN,
				Plaintext: plaintextDEK,
			}

			result, err := p.client.Encrypt(ctx, input)
			if err != nil {
				return nil, err
			}

			return result.CiphertextBlob, nil
		})
	})
}

func (p *AWSKMSProvider) DecryptDEK(ctx context.Context, key *domain.Key) ([]byte, error) {
	return execution.WithRetry(ctx, maxRetries, initialBackoff, maxBackoff, func(ctx context.Context) ([]byte, error) {
		return execution.WithTimeout(ctx, awsKmsTimeout, func(ctx context.Context) ([]byte, error) {
			input := &kms.DecryptInput{
				CiphertextBlob: key.EncryptedDEK,
				KeyId:          &p.kmsKeyARN,
			}

			result, err := p.client.Decrypt(ctx, input)
			if err != nil {
				return nil, err
			}

			return result.Plaintext, nil
		})
	})
}

func (p *AWSKMSProvider) HealthCheck(ctx context.Context) error {
	_, err := execution.WithRetry(ctx, maxRetries, initialBackoff, maxBackoff, func(ctx context.Context) (any, error) {
		return execution.WithTimeout(ctx, awsKmsTimeout, func(ctx context.Context) (*kms.ListKeysOutput, error) {
			return p.client.ListKeys(ctx, &kms.ListKeysInput{Limit: aws.Int32(1)})
		})
	})
	return err
}
