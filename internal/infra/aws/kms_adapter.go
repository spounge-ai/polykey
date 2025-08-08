package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	
)

type KMSAdapter struct {
	client *kms.Client
}

func NewKMSAdapter(cfg aws.Config) *KMSAdapter {
	return &KMSAdapter{
		client: kms.NewFromConfig(cfg),
	}
}

// EncryptDEK encrypts a Data Encryption Key (DEK) using the specified master key in AWS KMS.
func (a *KMSAdapter) EncryptDEK(ctx context.Context, plaintextDEK []byte, masterKeyID string) ([]byte, error) {
	input := &kms.EncryptInput{
		KeyId:     &masterKeyID,
		Plaintext: plaintextDEK,
	}

	result, err := a.client.Encrypt(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.CiphertextBlob, nil
}

// DecryptDEK decrypts a Data Encryption Key (DEK) using AWS KMS.
func (a *KMSAdapter) DecryptDEK(ctx context.Context, encryptedDEK []byte, masterKeyID string) ([]byte, error) {
	input := &kms.DecryptInput{
		CiphertextBlob: encryptedDEK,
		KeyId:          &masterKeyID,
	}

	result, err := a.client.Decrypt(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.Plaintext, nil
}
