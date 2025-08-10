package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/spounge-ai/polykey/tests/utils"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/yaml.v3"
)

const (
	defaultPort      = "50053"
	certPath         = "certs/cert.pem"
	secretConfigPath = "configs/dev_client/secret.dev.yaml"
	defaultTimeout   = 30 * time.Second
	authHeader       = "authorization"
	bearerPrefix     = "Bearer "
	invalidToken     = "this-is-not-a-valid-token"
)

type clientSecretConfig struct {
	ID     string `yaml:"id"`
	Secret string `yaml:"secret"`
}

type PolykeyTestClient struct {
	client pk.PolykeyServiceClient
	logger *slog.Logger
	creds  *clientSecretConfig
	ctx    context.Context
	cancel context.CancelFunc
}

func main() {
	logBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuf, nil))

	testClient, err := NewPolykeyTestClient(logger)
	if err != nil {
		utils.PrintJestReport(logBuf.String())
		os.Exit(1)
	}
	defer testClient.Close()

	testClient.RunAllTests()

	if utils.PrintJestReport(logBuf.String()) {
		os.Exit(1)
	}
}

func NewPolykeyTestClient(logger *slog.Logger) (*PolykeyTestClient, error) {
	port := getPort()
	logger.Info("Configuration loaded", "server", "localhost:"+port)

	creds, err := loadCredentials(secretConfigPath, logger)
	if err != nil {
		return nil, err
	}

	conn, err := establishConnection(port, logger)
	if err != nil {
		return nil, err
	}

	client := pk.NewPolykeyServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return &PolykeyTestClient{
		client: client,
		logger: logger,
		creds:  creds,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (tc *PolykeyTestClient) Close() {
	tc.cancel()
}

func (tc *PolykeyTestClient) RunAllTests() {
	tc.runHappyPathTests()
	tc.runErrorConditionTests()
}

func (tc *PolykeyTestClient) runHappyPathTests() {
	authToken := tc.authenticate()
	if authToken == "" {
		return
	}

	authedCtx := tc.createAuthenticatedContext(authToken)

	tc.testHealthCheck(authedCtx)
	keyID := tc.testCreateKey(authedCtx)
	if keyID != "" {
		tc.testGetKey(authedCtx, keyID)
		tc.testKeyRotation(authedCtx, keyID)
	}
	tc.testListKeys(authedCtx)
}

func (tc *PolykeyTestClient) runErrorConditionTests() {
	tc.testUnauthenticatedAccess()
	tc.testInvalidToken()
}

func (tc *PolykeyTestClient) authenticate() string {
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

func (tc *PolykeyTestClient) createAuthenticatedContext(token string) context.Context {
	return metadata.AppendToOutgoingContext(tc.ctx, authHeader, bearerPrefix+token)
}

func (tc *PolykeyTestClient) testHealthCheck(ctx context.Context) {
	healthResp, err := tc.client.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		tc.logger.Error("HealthCheck failed", "error", err)
		return
	}
	tc.logger.Info("HealthCheck successful", "status", healthResp.GetStatus().String(), "version", healthResp.GetServiceVersion())
}

func (tc *PolykeyTestClient) testCreateKey(ctx context.Context) string {
	createResp, err := tc.client.CreateKey(ctx, &pk.CreateKeyRequest{
		KeyType: pk.KeyType_KEY_TYPE_AES_256,
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: tc.creds.ID,
		},
		InitialAuthorizedContexts: []string{tc.creds.ID},
		Description:               "A key created by the test client",
		Tags:                      map[string]string{"source": "test_client"},
	})
	if err != nil {
		tc.logger.Error("CreateKey failed", "error", err)
		return ""
	}
	keyID := createResp.GetMetadata().GetKeyId()
	tc.logger.Info("CreateKey successful", 
		"keyId", keyID,
		"keyType", createResp.GetMetadata().GetKeyType().String())
	return keyID
}

func (tc *PolykeyTestClient) testGetKey(ctx context.Context, keyID string) {
	_, err := tc.client.GetKey(ctx, &pk.GetKeyRequest{
		KeyId: keyID,
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: tc.creds.ID,
		},
	})
	if err != nil {
		tc.logger.Error("GetKey failed", "error", err)
		return
	}
	tc.logger.Info("GetKey successful", "keyId", keyID)
}

func (tc *PolykeyTestClient) testKeyRotation(ctx context.Context, keyID string) {
	// Get original key before rotation
	getKeyReq := &pk.GetKeyRequest{
		KeyId:            keyID,
		RequesterContext: &pk.RequesterContext{ClientIdentity: tc.creds.ID},
	}
	originalKeyResp, err := tc.client.GetKey(ctx, getKeyReq)
	if err != nil {
		tc.logger.Error("GetKey (pre-rotation) failed", "error", err)
		return
	}
	tc.logger.Info("GetKey successful (pre-rotation)",
		"keyId", originalKeyResp.GetMetadata().GetKeyId(),
		"version", originalKeyResp.GetMetadata().GetVersion())

	// Rotate the key
	rotateKeyReq := &pk.RotateKeyRequest{
		KeyId:            keyID,
		RequesterContext: &pk.RequesterContext{ClientIdentity: tc.creds.ID},
	}
	rotateKeyResp, err := tc.client.RotateKey(ctx, rotateKeyReq)
	if err != nil {
		tc.logger.Error("RotateKey failed", "error", err)
		return
	}
	tc.logger.Info("RotateKey successful",
		"keyId", rotateKeyResp.GetKeyId(),
		"newVersion", rotateKeyResp.GetNewVersion(),
		"previousVersion", rotateKeyResp.GetPreviousVersion())

	// Get key after rotation
	postRotateKeyResp, err := tc.client.GetKey(ctx, getKeyReq)
	if err != nil {
		tc.logger.Error("GetKey (post-rotation) failed", "error", err)
		return
	}
	tc.logger.Info("GetKey successful (post-rotation)",
		"keyId", postRotateKeyResp.GetMetadata().GetKeyId(),
		"version", postRotateKeyResp.GetMetadata().GetVersion())

	// Validate rotation
	tc.validateKeyRotation(originalKeyResp, postRotateKeyResp)
}

func (tc *PolykeyTestClient) validateKeyRotation(originalKey, rotatedKey *pk.GetKeyResponse) {
	tc.logger.Info("Starting key rotation validation")

	// Compare key IDs (should be the same)
	originalKeyId := originalKey.GetMetadata().GetKeyId()
	rotatedKeyId := rotatedKey.GetMetadata().GetKeyId()
	
	if originalKeyId == rotatedKeyId {
		tc.logger.Info("Key ID preserved after rotation", "keyId", originalKeyId)
	} else {
		tc.logger.Error("Key ID changed after rotation", 
			"originalKeyId", originalKeyId, 
			"rotatedKeyId", rotatedKeyId)
	}

	// Compare versions (should be incremented)
	originalVersion := originalKey.GetMetadata().GetVersion()
	rotatedVersion := rotatedKey.GetMetadata().GetVersion()
	
	if originalVersion != rotatedVersion && rotatedVersion == originalVersion+1 {
		tc.logger.Info("Key version incremented correctly", 
			"originalVersion", originalVersion, 
			"rotatedVersion", rotatedVersion)
	} else {
		tc.logger.Error("Key version not incremented properly", 
			"originalVersion", originalVersion, 
			"rotatedVersion", rotatedVersion)
	}

	// Compare key material (should be different)
	originalKeyMaterial := originalKey.GetKeyMaterial().GetEncryptedKeyData()
	rotatedKeyMaterial := rotatedKey.GetKeyMaterial().GetEncryptedKeyData()
	
	if !bytes.Equal(originalKeyMaterial, rotatedKeyMaterial) {
		tc.logger.Info("Key material successfully rotated")
	} else {
		tc.logger.Error("Key material unchanged after rotation")
	}

	// Compare key types (should be the same)
	originalKeyType := originalKey.GetMetadata().GetKeyType()
	rotatedKeyType := rotatedKey.GetMetadata().GetKeyType()
	
	if originalKeyType == rotatedKeyType {
		tc.logger.Info("Key type preserved", "keyType", originalKeyType.String())
	} else {
		tc.logger.Error("Key type changed unexpectedly", 
			"originalKeyType", originalKeyType.String(), 
			"rotatedKeyType", rotatedKeyType.String())
	}

	tc.logger.Info("Key rotation validation completed")
}

func (tc *PolykeyTestClient) testListKeys(ctx context.Context) {
	listResp, err := tc.client.ListKeys(ctx, &pk.ListKeysRequest{
		RequesterContext: &pk.RequesterContext{ClientIdentity: tc.creds.ID},
	})
	if err != nil {
		tc.logger.Error("ListKeys failed", "error", err)
		return
	}
	tc.logger.Info("ListKeys successful", "count", len(listResp.GetKeys()))
}

func (tc *PolykeyTestClient) testUnauthenticatedAccess() {
	_, err := tc.client.ListKeys(tc.ctx, &pk.ListKeysRequest{})
	if err == nil {
		tc.logger.Error("Unauthenticated access test failed", "error", "request succeeded but should have failed")
		return
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.Unauthenticated {
		tc.logger.Info("Unauthenticated access test passed", "code", s.Code().String())
	} else {
		tc.logger.Error("Unauthenticated access test failed", "error", err)
	}
}

func (tc *PolykeyTestClient) testInvalidToken() {
	invalidCtx := metadata.AppendToOutgoingContext(tc.ctx, authHeader, bearerPrefix+invalidToken)
	_, err := tc.client.ListKeys(invalidCtx, &pk.ListKeysRequest{})
	if err == nil {
		tc.logger.Error("Invalid token test failed", "error", "request succeeded but should have failed")
		return
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.Unauthenticated {
		tc.logger.Info("Invalid token test passed", "code", s.Code().String())
	} else {
		tc.logger.Error("Invalid token test failed", "error", err)
	}
}

func getPort() string {
	if port := os.Getenv("POLYKEY_GRPC_PORT"); port != "" {
		return port
	}
	return defaultPort
}

func loadCredentials(path string, logger *slog.Logger) (*clientSecretConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to read client secret file", "path", path, "error", err)
		return nil, err
	}
	var config clientSecretConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		logger.Error("failed to unmarshal client secret file", "path", path, "error", err)
		return nil, err
	}
	return &config, nil
}

func establishConnection(port string, logger *slog.Logger) (*grpc.ClientConn, error) {
	creds, err := credentials.NewClientTLSFromFile(certPath, "")
	if err != nil {
		logger.Error("failed to load TLS credentials", "error", err)
		return nil, err
	}

	conn, err := grpc.NewClient("localhost:"+port, grpc.WithTransportCredentials(creds))
	if err != nil {
		logger.Error("gRPC connection failed", "error", err)
		return nil, err
	}

	logger.Info("gRPC connection established successfully")
	return conn, nil
}