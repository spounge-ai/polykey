package secrets

import "context"

type BootstrapSecretProvider interface {
	GetSecret(ctx context.Context, name string) (string, error)
}
