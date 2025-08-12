package kms

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/spounge-ai/polykey/internal/domain"
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
	input := &kms.EncryptInput{
		KeyId:     &p.kmsKeyARN,
		Plaintext: plaintextDEK,
	}

	result, err := p.client.Encrypt(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.CiphertextBlob, nil
}

func (p *AWSKMSProvider) DecryptDEK(ctx context.Context, key *domain.Key) ([]byte, error) {
	input := &kms.DecryptInput{
		CiphertextBlob: key.EncryptedDEK,
		KeyId:          &p.kmsKeyARN,
	}

	result, err := p.client.Decrypt(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.Plaintext, nil
}
