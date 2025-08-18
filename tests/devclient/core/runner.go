package core

import (
	"context"
	"time"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// TestCase defines a single generic test case for a gRPC operation.
// RequestT is the request protobuf type, ResponseT is the response protobuf type.
// It's designed to be used in a table-driven test.
type TestCase[RequestT any, ResponseT any] struct {
	Name      string
	Setup     func(tc TestClient) (context.Context, RequestT, bool) // Returns context, request, and a skip flag.
	RPC       func(ctx context.Context, client pk.PolykeyServiceClient, req RequestT) (ResponseT, error)
	Validate  func(tc TestClient, resp ResponseT, err error, duration time.Duration) // Custom validation and logging.
	Teardown  func(tc TestClient) // Optional teardown logic.
}

// RunTestCases executes a slice of TestCase structs.
func RunTestCases[RequestT any, ResponseT any](tc TestClient, cases []TestCase[RequestT, ResponseT]) {
	for _, tt := range cases {
		tc.Logger().Info("--- Running Test Case ---", "name", tt.Name)

		ctx, req, skip := tt.Setup(tc)
		if skip {
			tc.Logger().Info("Skipping test case", "name", tt.Name)
			continue
		}

		startTime := time.Now()
		resp, err := tt.RPC(ctx, tc.Client(), req)
		duration := time.Since(startTime)

		tt.Validate(tc, resp, err, duration)

		if tt.Teardown != nil {
			tt.Teardown(tc)
		}
	}
}
