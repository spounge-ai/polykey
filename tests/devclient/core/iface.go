package core

import (
	"context"
	"log/slog"

	app_errors "github.com/spounge-ai/polykey/internal/errors"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// TestClient defines an interface for the test client, exposing methods needed by test suites.
type TestClient interface {
	Client() pk.PolykeyServiceClient
	Logger() *slog.Logger
	Creds() *ClientSecretConfig
	ErrorClassifier() *app_errors.ErrorClassifier
	Authenticate() string
	CreateAuthenticatedContext(token string) context.Context
	Ctx() context.Context
}
