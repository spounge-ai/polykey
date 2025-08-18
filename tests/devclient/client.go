package devclient

import (
	"context"
	"log/slog"
	"os"

	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/wiring"
	"github.com/spounge-ai/polykey/tests/devclient/core"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"gopkg.in/yaml.v3"
)

type PolykeyTestClient struct {
	client          pk.PolykeyServiceClient
	logger          *slog.Logger
	creds           *core.ClientSecretConfig
	ctx             context.Context
	cancel          context.CancelFunc
	errorClassifier *app_errors.ErrorClassifier
}

func NewPolykeyTestClient(logger *slog.Logger) (*PolykeyTestClient, error) {
	port := getPort()
	logger.Info("Configuration loaded", "server", "localhost:"+port)

	creds, err := loadCredentials(SecretConfigPath, logger)
	if err != nil {
		return nil, err
	}

	conn, err := establishConnection(port, logger)
	if err != nil {
		return nil, err
	}

	client := pk.NewPolykeyServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)

	return &PolykeyTestClient{
		client:          client,
		logger:          logger,
		creds:           creds,
		ctx:             ctx,
		cancel:          cancel,
		errorClassifier: app_errors.NewErrorClassifier(logger),
	}, nil
}

func (tc *PolykeyTestClient) Close() {
	tc.cancel()
}

func (tc *PolykeyTestClient) Client() pk.PolykeyServiceClient {
	return tc.client
}

func (tc *PolykeyTestClient) Logger() *slog.Logger {
	return tc.logger
}

func (tc *PolykeyTestClient) Creds() *core.ClientSecretConfig {
	return tc.creds
}

func (tc *PolykeyTestClient) ErrorClassifier() *app_errors.ErrorClassifier {
	return tc.errorClassifier
}

func (tc *PolykeyTestClient) Ctx() context.Context {
	return tc.ctx
}

func (tc *PolykeyTestClient) Authenticate() string {
	authResp, err := tc.client.Authenticate(tc.ctx, &pk.AuthenticateRequest{
		ClientId: tc.creds.ID,
		ApiKey:   tc.creds.Secret,
	})

	if err != nil {
		tc.logger.Error("Authentication failed", "error", err)
		return ""
	}

	tc.logger.Info("Authentication successful", "expires_in", authResp.GetExpiresIn())
	return authResp.GetAccessToken()
}

func (tc *PolykeyTestClient) CreateAuthenticatedContext(token string) context.Context {
	return metadata.AppendToOutgoingContext(tc.ctx, AuthHeader, BearerPrefix+token)
}

func getPort() string {
	if port := os.Getenv("POLYKEY_GRPC_PORT"); port != "" {
		return port
	}
	return DefaultPort
}

func loadCredentials(path string, logger *slog.Logger) (*core.ClientSecretConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to read client secret file", "path", path, "error", err)
		return nil, err
	}
	var config core.ClientSecretConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		logger.Error("failed to unmarshal client secret file", "path", path, "error", err)
		return nil, err
	}
	return &config, nil
}

func establishConnection(port string, logger *slog.Logger) (*grpc.ClientConn, error) {
	tlsConfig, err := wiring.ConfigureClientTLS(TLSConfigPath)
	if err != nil {
		logger.Error("failed to configure client TLS", "error", err)
		return nil, err
	}

	creds := credentials.NewTLS(tlsConfig)

	conn, err := grpc.NewClient("localhost:"+port, grpc.WithTransportCredentials(creds))
	if err != nil {
		logger.Error("gRPC connection failed", "error", err)
		return nil, err
	}

	logger.Info("gRPC connection established successfully")
	return conn, nil
}
