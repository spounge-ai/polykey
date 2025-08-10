package secrets

import "context"

// BootstrapSecretProvider is an interface for retrieving bootstrap secrets.
type BootstrapSecretProvider interface {
	GetSecret(ctx context.Context, name string) (string, error)
}
