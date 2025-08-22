package suites

import (
	"context"
	"time"

	"github.com/spounge-ai/polykey/tests/devclient/core"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

type BatchSuite struct{}

func (s *BatchSuite) Name() string {
	return "Batch Operations"
}

func (s *BatchSuite) Run(tc core.TestClient) error {
	authToken, err := tc.Authenticate()
	if err != nil {
		tc.Logger().Error("suite auth failed, skipping", "suite", s.Name(), "error", err)
		return err
	}
	authedCtx := tc.CreateAuthenticatedContext(authToken)

	var createdKeys []*pk.BatchCreateKeysResult

	// -----------------------
	// BatchCreateKeys
	// -----------------------
	createCases := []core.TestCase[*pk.BatchCreateKeysRequest, *pk.BatchCreateKeysResponse]{
		{
			Name: "BatchCreateKeys",
			Setup: func(tc core.TestClient) (context.Context, *pk.BatchCreateKeysRequest, bool) {
				createItems := []*pk.CreateKeyItem{
					{KeyType: pk.KeyType_KEY_TYPE_AES_256, Description: "Batch key 1"},
					{KeyType: pk.KeyType_KEY_TYPE_AES_256, Description: "Batch key 2"},
				}
				req := &pk.BatchCreateKeysRequest{
					RequesterContext: core.DefaultRequesterContext(tc.Creds().ID),
					Keys:             createItems,
				}
				return authedCtx, req, false
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.BatchCreateKeysRequest) (*pk.BatchCreateKeysResponse, error) {
				return client.BatchCreateKeys(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.BatchCreateKeysResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("BatchCreateKeys failed", "error", err)
				} else {
					tc.Logger().Info("BatchCreateKeys successful", "duration", duration)
					createdKeys = resp.GetResults()
				}
			},
		},
	}
	core.RunTestCases(tc, createCases)

	if len(createdKeys) == 0 {
		tc.Logger().Warn("No keys created, skipping remaining batch tests")
		return nil
	}

	// -----------------------
	// Helper: get all valid key IDs
	// -----------------------
	getValidKeyIDs := func() []*pk.KeyRequestItem {
		var items []*pk.KeyRequestItem
		for _, r := range createdKeys {
			if s := r.GetSuccess(); s != nil {
				items = append(items, &pk.KeyRequestItem{KeyId: s.GetKeyId()})
			}
		}
		return items
	}

	keyItems := getValidKeyIDs()

	// -----------------------
	// BatchGetKeys
	// -----------------------
	getCases := []core.TestCase[*pk.BatchGetKeysRequest, *pk.BatchGetKeysResponse]{
		{
			Name: "BatchGetKeys",
			Setup: func(tc core.TestClient) (context.Context, *pk.BatchGetKeysRequest, bool) {
				req := &pk.BatchGetKeysRequest{
					RequesterContext: core.DefaultRequesterContext(tc.Creds().ID),
					Keys:             keyItems,
				}
				return authedCtx, req, len(keyItems) == 0
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.BatchGetKeysRequest) (*pk.BatchGetKeysResponse, error) {
				return client.BatchGetKeys(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.BatchGetKeysResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("BatchGetKeys failed", "error", err)
				} else {
					tc.Logger().Info("BatchGetKeys successful", "duration", duration)
				}
			},
		},
	}
	core.RunTestCases(tc, getCases)

	// -----------------------
	// BatchGetKeyMetadata
	// -----------------------
	metaCases := []core.TestCase[*pk.BatchGetKeyMetadataRequest, *pk.BatchGetKeyMetadataResponse]{
		{
			Name: "BatchGetKeyMetadata",
			Setup: func(tc core.TestClient) (context.Context, *pk.BatchGetKeyMetadataRequest, bool) {
				metaItems := make([]*pk.GetKeyMetadataItem, 0, len(keyItems))
				for _, k := range keyItems {
					metaItems = append(metaItems, &pk.GetKeyMetadataItem{KeyId: k.KeyId})
				}

				req := &pk.BatchGetKeyMetadataRequest{
					RequesterContext: core.DefaultRequesterContext(tc.Creds().ID),
					Keys:             metaItems,
				}

				return authedCtx, req, len(keyItems) == 0
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.BatchGetKeyMetadataRequest) (*pk.BatchGetKeyMetadataResponse, error) {
				return client.BatchGetKeyMetadata(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.BatchGetKeyMetadataResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("BatchGetKeyMetadata failed", "error", err)
				} else {
					tc.Logger().Info("BatchGetKeyMetadata successful", "duration", duration)
				}
			},
		},
	}
	core.RunTestCases(tc, metaCases)

	// -----------------------
	// BatchUpdateKeyMetadata
	// -----------------------
	updateCases := []core.TestCase[*pk.BatchUpdateKeyMetadataRequest, *pk.BatchUpdateKeyMetadataResponse]{
		{
			Name: "BatchUpdateKeyMetadata",
			Setup: func(tc core.TestClient) (context.Context, *pk.BatchUpdateKeyMetadataRequest, bool) {
				updates := make([]*pk.UpdateKeyMetadataItem, 0, len(keyItems))
				for _, item := range keyItems {
					desc := "Updated description"
					updates = append(updates, &pk.UpdateKeyMetadataItem{
						KeyId:       item.KeyId,
						Description: &desc,
					})
				}
				req := &pk.BatchUpdateKeyMetadataRequest{
					RequesterContext: core.DefaultRequesterContext(tc.Creds().ID),
					Keys:             updates,  
				}
				return authedCtx, req, len(updates) == 0
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.BatchUpdateKeyMetadataRequest) (*pk.BatchUpdateKeyMetadataResponse, error) {
				return client.BatchUpdateKeyMetadata(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.BatchUpdateKeyMetadataResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("BatchUpdateKeyMetadata failed", "error", err)
				} else {
					tc.Logger().Info("BatchUpdateKeyMetadata successful", "duration", duration)
				}
			},
		},
	}
	core.RunTestCases(tc, updateCases)

	// -----------------------
	// BatchRotateKeys
	// -----------------------
	rotateCases := []core.TestCase[*pk.BatchRotateKeysRequest, *pk.BatchRotateKeysResponse]{
		{
			Name: "BatchRotateKeys",
			Setup: func(tc core.TestClient) (context.Context, *pk.BatchRotateKeysRequest, bool) {
				rotateItems := make([]*pk.RotateKeyItem, 0, len(keyItems))
				for _, k := range keyItems {
					rotateItems = append(rotateItems, &pk.RotateKeyItem{KeyId: k.KeyId})
				}
				req := &pk.BatchRotateKeysRequest{
					RequesterContext: core.DefaultRequesterContext(tc.Creds().ID),
					Keys:             rotateItems,
				}
				return authedCtx, req, len(rotateItems) == 0
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.BatchRotateKeysRequest) (*pk.BatchRotateKeysResponse, error) {
				return client.BatchRotateKeys(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.BatchRotateKeysResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("BatchRotateKeys failed", "error", err)
				} else {
					tc.Logger().Info("BatchRotateKeys successful", "duration", duration)
				}
			},
		},
	}
	core.RunTestCases(tc, rotateCases)

	// -----------------------
	// BatchRevokeKeys
	// -----------------------
	revokeCases := []core.TestCase[*pk.BatchRevokeKeysRequest, *pk.BatchRevokeKeysResponse]{
		{
			Name: "BatchRevokeKeys",
			Setup: func(tc core.TestClient) (context.Context, *pk.BatchRevokeKeysRequest, bool) {
				revokeItems := make([]*pk.RevokeKeyItem, 0, len(keyItems))
				for _, k := range keyItems {
					revokeItems = append(revokeItems, &pk.RevokeKeyItem{KeyId: k.KeyId})
				}
				req := &pk.BatchRevokeKeysRequest{
					RequesterContext: core.DefaultRequesterContext(tc.Creds().ID),
					Keys:             revokeItems,
				}
				return authedCtx, req, len(revokeItems) == 0
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.BatchRevokeKeysRequest) (*pk.BatchRevokeKeysResponse, error) {
				return client.BatchRevokeKeys(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.BatchRevokeKeysResponse, err error, duration time.Duration) {
				if err != nil {
					tc.Logger().Error("BatchRevokeKeys failed", "error", err)
				} else {
					tc.Logger().Info("BatchRevokeKeys successful", "duration", duration)
				}
			},
		},
	}
	core.RunTestCases(tc, revokeCases)

	return nil
}
