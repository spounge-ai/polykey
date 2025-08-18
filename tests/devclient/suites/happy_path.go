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

type HappyPathSuite struct{}

func (s *HappyPathSuite) Name() string {
	return "Happy Path"
}

func (s *HappyPathSuite) Run(tc core.TestClient) error {
	authToken := tc.Authenticate()
	if authToken == "" {
		return nil
	}

	authedCtx := tc.CreateAuthenticatedContext(authToken)

	testHealthCheck(tc, authedCtx)
	keyID := testCreateKey(tc, authedCtx)
	if keyID != "" {
		testGetKey(tc, authedCtx, keyID)
		testKeyExists(tc, authedCtx, keyID)
		testKeyRotation(tc, authedCtx, keyID)
		testCache(tc, authedCtx, keyID)
	}
	testListKeys(tc, authedCtx)
	return nil
}

func testHealthCheck(tc core.TestClient, ctx context.Context) {
	startTime := time.Now()
	healthResp, err := tc.Client().HealthCheck(ctx, &emptypb.Empty{})
	duration := time.Since(startTime)

	if err != nil {
		tc.Logger().Error("HealthCheck failed", "error", err, "duration", duration)
		return
	}
	tc.Logger().Info("HealthCheck successful", "status", healthResp.GetStatus().String(), "version", healthResp.GetServiceVersion(), "duration", duration)
}

func testCreateKey(tc core.TestClient, ctx context.Context) string {
	creds := tc.Creds()

	startTime := time.Now()
	createResp, err := tc.Client().CreateKey(ctx, &pk.CreateKeyRequest{
		KeyType: pk.KeyType_KEY_TYPE_AES_256,
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: creds.ID,

		},
		InitialAuthorizedContexts: []string{creds.ID},
		Description:               "A key created by the test client",
		Tags:                      map[string]string{"source": "test_client"},
	})
	duration := time.Since(startTime)

	if err != nil {
		tc.Logger().Error("CreateKey failed", "error", err, "duration", duration)
		return ""
	}
	keyID := createResp.GetMetadata().GetKeyId()
	tc.Logger().Info("CreateKey successful",
		"keyId", keyID,
		"keyType", createResp.GetMetadata().GetKeyType().String(),
		"duration", duration)
	return keyID
}

func testGetKey(tc core.TestClient, ctx context.Context, keyID string) {
	creds := tc.Creds()

	startTime := time.Now()
	_, err := tc.Client().GetKey(ctx, &pk.GetKeyRequest{
		KeyId: keyID,
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: creds.ID,

		},
	})
	duration := time.Since(startTime)

	if err != nil {
		tc.Logger().Error("GetKey failed", "error", err, "duration", duration)
		return
	}
	tc.Logger().Info("GetKey successful", "keyId", keyID, "duration", duration)
}

func testKeyExists(tc core.TestClient, ctx context.Context, keyID string) {
	creds := tc.Creds()

	startTimePositive := time.Now()
	_, errPositive := tc.Client().GetKey(ctx, &pk.GetKeyRequest{
		KeyId: keyID,
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: creds.ID,

		},
	})
	durationPositive := time.Since(startTimePositive)

	if errPositive != nil {
		tc.Logger().Error("Exists check for created key failed", "error", errPositive, "duration", durationPositive)
	} else {
		tc.Logger().Info("Exists check for created key passed", "keyId", keyID, "duration", durationPositive)
	}

	nonExistentKeyID := "00000000-0000-0000-0000-000000000000"
	startTimeNegative := time.Now()
	_, errNegative := tc.Client().GetKey(ctx, &pk.GetKeyRequest{
		KeyId: nonExistentKeyID,
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: creds.ID,

		},
	})
	durationNegative := time.Since(startTimeNegative)

	if errNegative == nil {
		tc.Logger().Error("Exists check for non-existent key failed", "error", "request succeeded but should have failed", "duration", durationNegative)
		return
	}
	if s, ok := status.FromError(errNegative); ok && s.Code() == codes.NotFound {
		tc.Logger().Info("Exists check for non-existent key passed", "code", s.Code().String(), "duration", durationNegative)
	} else {
		tc.Logger().Error("Exists check for non-existent key failed", "error", errNegative, "duration", durationNegative)
	}
}

func testKeyRotation(tc core.TestClient, ctx context.Context, keyID string) {
	creds := tc.Creds()
	getKeyReq := &pk.GetKeyRequest{
		KeyId:            keyID,
		RequesterContext: &pk.RequesterContext{ClientIdentity: creds.ID},
	}

	startTimePre := time.Now()
	originalKeyResp, errPre := tc.Client().GetKey(ctx, getKeyReq)
	durationPre := time.Since(startTimePre)
	if errPre != nil {
		tc.Logger().Error("GetKey (pre-rotation) failed", "error", errPre, "duration", durationPre)
		return
	}
	tc.Logger().Info("GetKey successful (pre-rotation)",
		"keyId", originalKeyResp.GetMetadata().GetKeyId(),
		"version", originalKeyResp.GetMetadata().GetVersion(),
		"duration", durationPre)

	rotateKeyReq := &pk.RotateKeyRequest{
		KeyId:            keyID,
		RequesterContext: &pk.RequesterContext{ClientIdentity: creds.ID},
	}
	startTimeRotate := time.Now()
	rotateKeyResp, errRotate := tc.Client().RotateKey(ctx, rotateKeyReq)
	durationRotate := time.Since(startTimeRotate)
	if errRotate != nil {
		tc.Logger().Error("RotateKey failed", "error", errRotate, "duration", durationRotate)
		return
	}
	tc.Logger().Info("RotateKey successful",
		"keyId", rotateKeyResp.GetKeyId(),
		"newVersion", rotateKeyResp.GetNewVersion(),
		"previousVersion", rotateKeyResp.GetPreviousVersion(),
		"duration", durationRotate)

	startTimePost := time.Now()
	postRotateKeyResp, errPost := tc.Client().GetKey(ctx, getKeyReq)
	durationPost := time.Since(startTimePost)
	if errPost != nil {
		tc.Logger().Error("GetKey (post-rotation) failed", "error", errPost, "duration", durationPost)
		return
	}
	tc.Logger().Info("GetKey successful (post-rotation)",
		"keyId", postRotateKeyResp.GetMetadata().GetKeyId(),
		"version", postRotateKeyResp.GetMetadata().GetVersion(),
		"duration", durationPost)

	validateKeyRotation(tc, originalKeyResp, postRotateKeyResp)
}

func validateKeyRotation(tc core.TestClient, originalKey, rotatedKey *pk.GetKeyResponse) {
	logger := tc.Logger()
	logger.Info("Starting key rotation validation")

	originalKeyId := originalKey.GetMetadata().GetKeyId()
	rotatedKeyId := rotatedKey.GetMetadata().GetKeyId()

	if originalKeyId == rotatedKeyId {
		logger.Info("Key ID preserved after rotation", "keyId", originalKeyId)
	} else {
		logger.Error("Key ID changed after rotation",
			"originalKeyId", originalKeyId,
			"rotatedKeyId", rotatedKeyId)
	}

	originalVersion := originalKey.GetMetadata().GetVersion()
	rotatedVersion := rotatedKey.GetMetadata().GetVersion()

	if originalVersion != rotatedVersion && rotatedVersion == originalVersion+1 {
		logger.Info("Key version incremented correctly",
			"originalVersion", originalVersion,
			"rotatedVersion", rotatedVersion)
	} else {
		logger.Error("Key version not incremented properly",
			"originalVersion", originalVersion,
			"rotatedVersion", rotatedVersion)
	}

	if !bytes.Equal(originalKey.GetKeyMaterial().GetEncryptedKeyData(), rotatedKey.GetKeyMaterial().GetEncryptedKeyData()) {
		logger.Info("Key material successfully rotated")
	} else {
		logger.Error("Key material unchanged after rotation")
	}

	originalKeyType := originalKey.GetMetadata().GetKeyType()
	rotatedKeyType := rotatedKey.GetMetadata().GetKeyType()

	if originalKeyType == rotatedKeyType {
		logger.Info("Key type preserved", "keyType", originalKeyType.String())
	} else {
		logger.Error("Key type changed unexpectedly",
			"originalKeyType", originalKeyType.String(),
			"rotatedKeyType", rotatedKeyType.String())
	}

	logger.Info("Key rotation validation completed")
}

func testListKeys(tc core.TestClient, ctx context.Context) {
	creds := tc.Creds()
	startTime := time.Now()
	listResp, err := tc.Client().ListKeys(ctx, &pk.ListKeysRequest{
		RequesterContext: &pk.RequesterContext{ClientIdentity: creds.ID},
	})
	duration := time.Since(startTime)
	if err != nil {
		tc.Logger().Error("ListKeys failed", "error", err, "duration", duration)
		return
	}
	tc.Logger().Info("ListKeys successful", "count", len(listResp.GetKeys()), "duration", duration)
}

func testCache(tc core.TestClient, ctx context.Context, keyID string) {
	tc.Logger().Info("--- Cache Test ---")
	creds := tc.Creds()

	startTimeGet := time.Now()
	_, errGet := tc.Client().GetKey(ctx, &pk.GetKeyRequest{
		KeyId: keyID,
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: creds.ID,

		},
	})
	durationGet := time.Since(startTimeGet)
	if errGet != nil {
		tc.Logger().Error("GetKey (cached) failed", "error", errGet, "duration", durationGet)
	} else {
		tc.Logger().Info("GetKey (cached) successful", "keyId", keyID, "duration", durationGet)
	}

	startTimeMeta := time.Now()
	_, errMeta := tc.Client().GetKeyMetadata(ctx, &pk.GetKeyMetadataRequest{
		KeyId: keyID,
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: creds.ID,

		},
	})
	durationMeta := time.Since(startTimeMeta)
	if errMeta != nil {
		tc.Logger().Error("GetKeyMetadata (cached) failed", "error", errMeta, "duration", durationMeta)
	} else {
		tc.Logger().Info("GetKeyMetadata (cached) successful", "keyId", keyID, "duration", durationMeta)
	}
}
