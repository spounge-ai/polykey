package suites

import (
	"time"

	"github.com/spounge-ai/polykey/tests/devclient/core"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	testUnauthenticatedAccess(tc)
	testInvalidToken(tc)
	return nil
}

func testUnauthenticatedAccess(tc core.TestClient) {
	startTime := time.Now()
	_, err := tc.Client().ListKeys(tc.Ctx(), &pk.ListKeysRequest{})
	duration := time.Since(startTime)

	if err == nil {
		tc.Logger().Error("Unauthenticated access test failed", "error", "request succeeded but should have failed", "duration", duration)
		return
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.Unauthenticated {
		tc.Logger().Info("Unauthenticated access test passed", "code", s.Code().String(), "duration", duration)
	} else {
		tc.Logger().Error("Unauthenticated access test failed", "error", err, "duration", duration)
	}
}

func testInvalidToken(tc core.TestClient) {
	invalidCtx := metadata.AppendToOutgoingContext(tc.Ctx(), AuthHeader, BearerPrefix+InvalidToken)

	startTime := time.Now()
	_, err := tc.Client().ListKeys(invalidCtx, &pk.ListKeysRequest{})
	duration := time.Since(startTime)

	if err == nil {
		tc.Logger().Error("Invalid token test failed", "error", "request succeeded but should have failed", "duration", duration)
		return
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.Unauthenticated {
		tc.Logger().Info("Invalid token test passed", "code", s.Code().String(), "duration", duration)
	} else {
		tc.Logger().Error("Invalid token test failed", "error", err, "duration", duration)
	}
}
