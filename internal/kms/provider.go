package kms

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
)

type KMSProvider interface {
	EncryptDEK(ctx context.Context, plaintextDEK []byte, key *domain.Key) ([]byte, error)
	DecryptDEK(ctx context.Context, key *domain.Key) ([]byte, error)
}
