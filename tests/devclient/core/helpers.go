package core

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HandleResult logs the outcome of a test.
func HandleResult(tc TestClient, testName string, err error) {
	if err != nil {
		tc.Logger().Error("Test failed", "test", testName, "error", err)
	} else {
		tc.Logger().Info("Test passed", "test", testName)
	}
}

// ExpectSuccess is a generic validation helper that logs the outcome of a test case.
func ExpectSuccess(tc TestClient, testName string, err error, duration time.Duration) {
	if err != nil {
		tc.Logger().Error(testName+" failed", "error", err, "duration", duration)
	} else {
		tc.Logger().Info(testName+" successful", "duration", duration)
	}
}

// ExpectGrpcError is a generic validation helper for checking specific gRPC error codes.
func ExpectGrpcError(tc TestClient, testName string, err error, expectedCode codes.Code, duration time.Duration) {
	if err == nil {
		tc.Logger().Error(testName+" failed", "error", "request succeeded but should have failed", "duration", duration)
		return
	}
	s, ok := status.FromError(err)
	if !ok || s.Code() != expectedCode {
		tc.Logger().Error(testName+" failed", "error", err, "expected_code", expectedCode, "duration", duration)
	} else {
		tc.Logger().Info(testName+" passed", "code", s.Code().String(), "duration", duration)
	}
}
