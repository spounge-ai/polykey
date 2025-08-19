package testutil

import (
	"context"
	"log/slog"
	"os"

	"github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/wiring"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"gopkg.in/yaml.v3"
)

const (
	AuthHeader   = "authorization"
	BearerPrefix = "Bearer "
)

// Client is a generic, reusable test client for the Polykey service.
// It is configured via the Config struct.
type Client struct {
	service         pk.PolykeyServiceClient
	logger          *slog.Logger
	creds           *ClientSecretConfig
	ctx             context.Context
	cancel          context.CancelFunc
	errorClassifier *errors.ErrorClassifier
	Conn            *grpc.ClientConn
}

// New creates a new test client from the given configuration.
func New(cfg Config, logger *slog.Logger) (*Client, error) {
	creds, err := loadCredentials(cfg.SecretConfigPath, logger)
	if err != nil {
		return nil, err
	}

	conn, err := establishConnection(cfg.ServerAddr, cfg.TLSConfigPath, logger)
	if err != nil {
		return nil, err
	}

	client := pk.NewPolykeyServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DefaultTimeout)

	return &Client{
		service:         client,
		logger:          logger,
		creds:           creds,
		ctx:             ctx,
		cancel:          cancel,
		errorClassifier: errors.NewErrorClassifier(logger),
		Conn:            conn,
	}, nil
}

func (c *Client) Close() {
	c.cancel()
	if err := c.Conn.Close(); err != nil {
		c.logger.Error("failed to close gRPC connection", "error", err)
	}
}

func (c *Client) Client() pk.PolykeyServiceClient {
	return c.service
}

func (c *Client) Logger() *slog.Logger {
	return c.logger
}

func (c *Client) Creds() *ClientSecretConfig {
	return c.creds
}

func (c *Client) Ctx() context.Context {
	return c.ctx
}

func (c *Client) Authenticate() (string, error) {
	authResp, err := c.service.Authenticate(c.ctx, &pk.AuthenticateRequest{
		ClientId: c.creds.ID,
		ApiKey:   c.creds.Secret,
	})

	if err != nil {
		c.logger.Error("Authentication failed", "error", err)
		return "", err
	}

	c.logger.Info("Authentication successful", "expires_in", authResp.GetExpiresIn())
	return authResp.GetAccessToken(), nil
}

func (c *Client) CreateAuthenticatedContext(token string) context.Context {
	return metadata.AppendToOutgoingContext(c.ctx, AuthHeader, BearerPrefix+token)
}

func loadCredentials(path string, logger *slog.Logger) (*ClientSecretConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to read client secret file", "path", path, "error", err)
		return nil, err
	}
	var config ClientSecretConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		logger.Error("failed to unmarshal client secret file", "path", path, "error", err)
		return nil, err
	}
	return &config, nil
}

func establishConnection(serverAddr, tlsConfigPath string, logger *slog.Logger) (*grpc.ClientConn, error) {
	tlsConfig, err := wiring.ConfigureClientTLS(tlsConfigPath)
	if err != nil {
		logger.Error("failed to configure client TLS", "error", err)
		return nil, err
	}

	creds := credentials.NewTLS(tlsConfig)

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		logger.Error("gRPC connection failed", "error", err)
		return nil, err
	}

	logger.Info("gRPC connection established successfully")
	return conn, nil
}
