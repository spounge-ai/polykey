package core

import (
	"context"
	"log/slog"

	"github.com/spounge-ai/polykey/pkg/testutil"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// TestClient defines an interface for the test client, exposing methods needed by test suites.
// It is implemented by testutil.Client.
type TestClient interface {
	Authenticate() (string, error)
	CreateAuthenticatedContext(token string) context.Context
	Client() pk.PolykeyServiceClient
	Logger() *slog.Logger
	Creds() *testutil.ClientSecretConfig
	Ctx() context.Context
}
