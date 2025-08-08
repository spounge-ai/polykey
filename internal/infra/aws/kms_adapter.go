package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

type KMSAdapter struct {
	client    *kms.Client
	kmsKeyARN string
}

func NewKMSAdapter(cfg aws.Config, kmsKeyARN string) *KMSAdapter {
	return &KMSAdapter{
		client:    kms.NewFromConfig(cfg),
		kmsKeyARN: kmsKeyARN,
	}
}

// EncryptDEK encrypts a Data Encryption Key (DEK) using the specified master key in AWS KMS.
func (a *KMSAdapter) EncryptDEK(ctx context.Context, plaintextDEK []byte, isPremium bool) ([]byte, error) {
	if !isPremium {
		return nil, fmt.Errorf("cannot use aws kms for non-premium keys")
	}
	input := &kms.EncryptInput{
		KeyId:     &a.kmsKeyARN,
		Plaintext: plaintextDEK,
	}

	result, err := a.client.Encrypt(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.CiphertextBlob, nil
}

// DecryptDEK decrypts a Data Encryption Key (DEK) using AWS KMS.
func (a *KMSAdapter) DecryptDEK(ctx context.Context, encryptedDEK []byte, isPremium bool) ([]byte, error) {
	if !isPremium {
		return nil, fmt.Errorf("cannot use aws kms for non-premium keys")
	}
	input := &kms.DecryptInput{
		CiphertextBlob: encryptedDEK,
		KeyId:          &a.kmsKeyARN,
	}

	result, err := a.client.Decrypt(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.Plaintext, nil
}
