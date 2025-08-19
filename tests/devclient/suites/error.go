package suites

import (
	"context"
	"time"

	"github.com/spounge-ai/polykey/tests/devclient/core"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const (
	InvalidToken = "this-is-not-a-valid-token"
	AuthHeader   = "authorization"
	BearerPrefix = "Bearer "
)

type ErrorSuite struct{}

func (s *ErrorSuite) Name() string {
	return "Error Conditions"
}

func (s *ErrorSuite) Run(tc core.TestClient) error {
	cases := []core.TestCase[*pk.ListKeysRequest, *pk.ListKeysResponse]{
		{
			Name: "Unauthenticated Access",
			Setup: func(tc core.TestClient) (context.Context, *pk.ListKeysRequest, bool) {
				return tc.Ctx(), &pk.ListKeysRequest{}, false // Use unauthenticated context
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
				return client.ListKeys(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.ListKeysResponse, err error, duration time.Duration) {
				core.ExpectGrpcError(tc, "Unauthenticated access test", err, codes.Unauthenticated, duration)
			},
		},
		{
			Name: "Invalid Token",
			Setup: func(tc core.TestClient) (context.Context, *pk.ListKeysRequest, bool) {
				invalidCtx := metadata.AppendToOutgoingContext(tc.Ctx(), AuthHeader, BearerPrefix+InvalidToken)
				return invalidCtx, &pk.ListKeysRequest{}, false
			},
			RPC: func(ctx context.Context, client pk.PolykeyServiceClient, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
				return client.ListKeys(ctx, req)
			},
			Validate: func(tc core.TestClient, resp *pk.ListKeysResponse, err error, duration time.Duration) {
				core.ExpectGrpcError(tc, "Invalid token test", err, codes.Unauthenticated, duration)
			},
		},
	}

	core.RunTestCases(tc, cases)
	return nil
}