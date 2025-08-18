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
		return nil // Skip suite if auth fails
	}
	authedCtx := tc.CreateAuthenticatedContext(authToken)

	var createdKeys []*pk.BatchCreateKeysResult

	batchCreateCases := []core.TestCase[*pk.BatchCreateKeysRequest, *pk.BatchCreateKeysResponse]{
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
					tc.Logger().Error("BatchCreateKeys failed", "error", err, "duration", duration)
				} else {
					tc.Logger().Info("BatchCreateKeys successful", "duration", duration)
					createdKeys = resp.GetResults()
				}
			},
		},
	}
	core.RunTestCases(tc, batchCreateCases)

	if len(createdKeys) > 0 {
		batchGetCases := []core.TestCase[*pk.BatchGetKeysRequest, *pk.BatchGetKeysResponse]{
			{
				Name: "BatchGetKeys",
				Setup: func(tc core.TestClient) (context.Context, *pk.BatchGetKeysRequest, bool) {
					getItems := make([]*pk.KeyRequestItem, 0, len(createdKeys))
					for _, result := range createdKeys {
						if success := result.GetSuccess(); success != nil {
							getItems = append(getItems, &pk.KeyRequestItem{KeyId: success.GetKeyId()})
						}
					}
					req := &pk.BatchGetKeysRequest{
						RequesterContext: core.DefaultRequesterContext(tc.Creds().ID),
						Keys:             getItems,
					}
					return authedCtx, req, len(getItems) == 0
				},
				RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.BatchGetKeysRequest) (*pk.BatchGetKeysResponse, error) {
					return client.BatchGetKeys(ctx, req)
				},
				Validate: func(tc core.TestClient, resp *pk.BatchGetKeysResponse, err error, duration time.Duration) {
					if err != nil {
						tc.Logger().Error("BatchGetKeys failed", "error", err, "duration", duration)
					} else {
						tc.Logger().Info("BatchGetKeys successful", "duration", duration)
					}
				},
			},
		}
		core.RunTestCases(tc, batchGetCases)
	}

	return nil
}