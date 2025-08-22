package suites

import (
	"bytes"
	"context"
	"time"

	"github.com/spounge-ai/polykey/tests/devclient/core"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type HappyPathSuite struct{
	originalKey *pk.GetKeyResponse
}

func (s *HappyPathSuite) Name() string {
	return "Happy Path"
}

func (s *HappyPathSuite) Run(tc core.TestClient) error {
	authToken, err := tc.Authenticate()
	if err != nil {
		tc.Logger().Error("suite auth failed, skipping", "suite", s.Name(), "error", err)
		return err
	}
	authedCtx := tc.CreateAuthenticatedContext(authToken)

	core.RunTestCases(tc, s.healthCheckCases(authedCtx))

	var createdKeyID string
	core.RunTestCases(tc, s.createKeyCases(authedCtx, &createdKeyID))
	core.RunTestCases(tc, s.getKeyCases(authedCtx, createdKeyID))
	core.RunTestCases(tc, s.createKeyCases(authedCtx, &createdKeyID))

	if createdKeyID != "" {
		core.RunTestCases(tc, s.getKeyCases(authedCtx, createdKeyID))
		core.RunTestCases(tc, s.keyExistsCases(authedCtx, createdKeyID))
		core.RunTestCases(tc, s.rotateKeyCases(authedCtx, createdKeyID))
		core.RunTestCases(tc, s.cacheCases(authedCtx, createdKeyID))
	}

	core.RunTestCases(tc, s.listKeysCases(authedCtx))

	return nil
}

func (s *HappyPathSuite) healthCheckCases(ctx context.Context) []core.TestCase[*emptypb.Empty, *pk.HealthCheckResponse] {
	return []core.TestCase[*emptypb.Empty, *pk.HealthCheckResponse]{
		{
			Name: "HealthCheck",
			Setup: func(tc core.TestClient) (context.Context, *emptypb.Empty, bool) {
				return ctx, &emptypb.Empty{}, false
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *emptypb.Empty) (*pk.HealthCheckResponse, error) {
				return client.HealthCheck(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.HealthCheckResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("HealthCheck failed", "error", err, "duration", duration)
				} else {
					tc.Logger().Info("HealthCheck successful", "status", resp.GetStatus().String(), "version", resp.GetServiceVersion(), "duration", duration)
				}
			},
		},
	}
}

func (s *HappyPathSuite) createKeyCases(ctx context.Context, createdKeyID *string) []core.TestCase[*pk.CreateKeyRequest, *pk.CreateKeyResponse] {
	return []core.TestCase[*pk.CreateKeyRequest, *pk.CreateKeyResponse]{
		{
			Name: "CreateKey",
			Setup: func(tc core.TestClient) (context.Context, *pk.CreateKeyRequest, bool) {
				req := &pk.CreateKeyRequest{
					KeyType:                   pk.KeyType_KEY_TYPE_AES_256,
					RequesterContext:          core.DefaultRequesterContext(tc.Creds().ID),
					InitialAuthorizedContexts: []string{tc.Creds().ID},
				}
				return ctx, req, false
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
				return client.CreateKey(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.CreateKeyResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("CreateKey failed", "error", err, "duration", duration)
				} else {
					*createdKeyID = resp.GetMetadata().GetKeyId()
					tc.Logger().Info("CreateKey successful", "keyId", *createdKeyID, "duration", duration)
				}
			},
		},
	}
}

func (s *HappyPathSuite) getKeyCases(ctx context.Context, keyID string) []core.TestCase[*pk.GetKeyRequest, *pk.GetKeyResponse] {
	return []core.TestCase[*pk.GetKeyRequest, *pk.GetKeyResponse]{
		{
			Name: "GetKey",
			Setup: func(tc core.TestClient) (context.Context, *pk.GetKeyRequest, bool) {
				return ctx, &pk.GetKeyRequest{KeyId: keyID, RequesterContext: core.DefaultRequesterContext(tc.Creds().ID)}, false
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
				return client.GetKey(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.GetKeyResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("GetKey failed", "error", err, "duration", duration)
				} else {
					s.originalKey = resp
					tc.Logger().Info("GetKey successful", "keyId", keyID, "version", resp.GetMetadata().GetVersion(), "duration", duration)
				}
			},
		},
	}
}

func (s *HappyPathSuite) keyExistsCases(ctx context.Context, keyID string) []core.TestCase[*pk.GetKeyRequest, *pk.GetKeyResponse] {
	return []core.TestCase[*pk.GetKeyRequest, *pk.GetKeyResponse]{
		{
			Name: "KeyExists - Positive",
			Setup: func(tc core.TestClient) (context.Context, *pk.GetKeyRequest, bool) {
				return ctx, &pk.GetKeyRequest{KeyId: keyID, RequesterContext: core.DefaultRequesterContext(tc.Creds().ID)}, false
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
				return client.GetKey(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.GetKeyResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("Exists check for created key failed", "error", err, "duration", duration)
				} else {
					tc.Logger().Info("Exists check for created key passed", "duration", duration)
				}
			},
		},
		{
			Name: "KeyExists - Negative",
			Setup: func(tc core.TestClient) (context.Context, *pk.GetKeyRequest, bool) {
				return ctx, &pk.GetKeyRequest{KeyId: "00000000-0000-0000-0000-000000000000", RequesterContext: core.DefaultRequesterContext(tc.Creds().ID)}, false
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
				return client.GetKey(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.GetKeyResponse, err error, duration time.Duration) {
				if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
					tc.Logger().Info("Exists check for non-existent key passed", "duration", duration)
				} else {
					tc.Logger().Error("Exists check for non-existent key failed", "error", err, "duration", duration)
				}
			},
		},
	}
}

func (s *HappyPathSuite) rotateKeyCases(ctx context.Context, keyID string) []core.TestCase[*pk.RotateKeyRequest, *pk.RotateKeyResponse] {
	return []core.TestCase[*pk.RotateKeyRequest, *pk.RotateKeyResponse]{
		{
			Name: "RotateKey",
			Setup: func(tc core.TestClient) (context.Context, *pk.RotateKeyRequest, bool) {
				return ctx, &pk.RotateKeyRequest{KeyId: keyID, RequesterContext: core.DefaultRequesterContext(tc.Creds().ID)}, s.originalKey == nil
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
				return client.RotateKey(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.RotateKeyResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("RotateKey failed", "error", err, "duration", duration)
					return
				}
				tc.Logger().Info("RotateKey successful", "keyId", resp.GetKeyId(), "newVersion", resp.GetNewVersion(), "duration", duration)

				postRotateKeyResp, postErr := tc.Client().GetKey(ctx, &pk.GetKeyRequest{KeyId: keyID, RequesterContext: core.DefaultRequesterContext(tc.Creds().ID)})
				if postErr == nil {
					s.validateKeyRotation(tc, s.originalKey, postRotateKeyResp)
				}
			},
		},
	}
}

func (s *HappyPathSuite) validateKeyRotation(tc core.TestClient, originalKey, rotatedKey *pk.GetKeyResponse) {
	if !bytes.Equal(originalKey.GetKeyMaterial().GetEncryptedKeyData(), rotatedKey.GetKeyMaterial().GetEncryptedKeyData()) {
		tc.Logger().Info("Key material successfully rotated")
	}
}

func (s *HappyPathSuite) cacheCases(ctx context.Context, keyID string) []core.TestCase[*pk.GetKeyRequest, *pk.GetKeyResponse] {
	return []core.TestCase[*pk.GetKeyRequest, *pk.GetKeyResponse]{
		{
			Name: "GetKey (cached)",
			Setup: func(tc core.TestClient) (context.Context, *pk.GetKeyRequest, bool) {
				return ctx, &pk.GetKeyRequest{KeyId: keyID, RequesterContext: core.DefaultRequesterContext(tc.Creds().ID)}, false
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
				return client.GetKey(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.GetKeyResponse, err error, duration time.Duration) {
				if (err != nil) {
					tc.Logger().Error("GetKey (cached) failed", "error", err, "duration", duration)
				} else {
					tc.Logger().Info("GetKey (cached) successful", "duration", duration)
				}
			},
		},
	}
}

func (s *HappyPathSuite) listKeysCases(ctx context.Context) []core.TestCase[*pk.ListKeysRequest, *pk.ListKeysResponse] {
	return []core.TestCase[*pk.ListKeysRequest, *pk.ListKeysResponse]{
		{
			Name: "ListKeys",
			Setup: func(tc core.TestClient) (context.Context, *pk.ListKeysRequest, bool) {
				return ctx, &pk.ListKeysRequest{RequesterContext: core.DefaultRequesterContext(tc.Creds().ID)}, false
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
				return client.ListKeys(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.ListKeysResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("ListKeys failed", "error", err, "duration", duration)
				} else {
					tc.Logger().Info("ListKeys successful", "count", len(resp.GetKeys()), "duration", duration)
				}
			},
		},
	}
}
