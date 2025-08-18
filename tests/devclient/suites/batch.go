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
	authToken := tc.Authenticate()
	if authToken == "" {
		return nil
	}

	authedCtx := tc.CreateAuthenticatedContext(authToken)

	createdKeys := testBatchCreateKeys(tc, authedCtx)
	testBatchGetKeys(tc, authedCtx, createdKeys)
	return nil
}

func testBatchCreateKeys(tc core.TestClient, ctx context.Context) []*pk.BatchCreateKeysResult {
	tc.Logger().Info("--- Batch Create Keys Test ---")
	creds := tc.Creds()
	createItems := []*pk.CreateKeyItem{
		{
			KeyType:     pk.KeyType_KEY_TYPE_AES_256,
			Description: "Batch key 1",
			Tags:        map[string]string{"batch": "true", "index": "1"},
		},
		{
			KeyType:     pk.KeyType_KEY_TYPE_AES_256,
			Description: "Batch key 2",
			Tags:        map[string]string{"batch": "true", "index": "2"},
		},
	}

	req := &pk.BatchCreateKeysRequest{
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: creds.ID,
			
		},
		Keys: createItems,
	}

	startTime := time.Now()
	resp, err := tc.Client().BatchCreateKeys(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		tc.Logger().Error("BatchCreateKeys failed", "error", err, "duration", duration)
		return nil
	}

	tc.Logger().Info("BatchCreateKeys successful", "results_count", len(resp.GetResults()), "duration", duration)
	for i, result := range resp.GetResults() {
		switch r := result.Result.(type) {
		case *pk.BatchCreateKeysResult_Success:
			tc.Logger().Info("Key created successfully", "index", i, "keyId", r.Success.GetKeyId())
		case *pk.BatchCreateKeysResult_Error:
			tc.Logger().Error("Failed to create key", "index", i, "error", r.Error)
		}
	}
	return resp.GetResults()
}

func testBatchGetKeys(tc core.TestClient, ctx context.Context, createdKeys []*pk.BatchCreateKeysResult) {
	tc.Logger().Info("--- Batch Get Keys Test ---")
	if len(createdKeys) == 0 {
		tc.Logger().Info("No keys to get in batch")
		return
	}

	creds := tc.Creds()
	getItems := make([]*pk.KeyRequestItem, 0, len(createdKeys))
	for _, result := range createdKeys {
		if success := result.GetSuccess(); success != nil {
			getItems = append(getItems, &pk.KeyRequestItem{
				KeyId: success.GetKeyId(),
			})
		}
	}

	req := &pk.BatchGetKeysRequest{
		RequesterContext: &pk.RequesterContext{
			ClientIdentity: creds.ID,
			
		},
		Keys: getItems,
	}

	startTime := time.Now()
	resp, err := tc.Client().BatchGetKeys(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		tc.Logger().Error("BatchGetKeys failed", "error", err, "duration", duration)
		return
	}

	tc.Logger().Info("BatchGetKeys successful", "results_count", len(resp.GetResults()), "duration", duration)
	for i, result := range resp.GetResults() {
		switch r := result.Result.(type) {
		case *pk.BatchGetKeysResult_Success:
			tc.Logger().Info("Key retrieved successfully", "index", i, "keyId", r.Success.GetMetadata().GetKeyId())
		case *pk.BatchGetKeysResult_Error:
			tc.Logger().Error("Failed to retrieve key", "index", i, "error", r.Error)
		}
	}
}
