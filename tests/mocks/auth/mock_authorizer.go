package auth

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// MockAuthorizer is a mock implementation of the Authorizer interface.
type MockAuthorizer struct{}

// NewMockAuthorizer creates a new MockAuthorizer.
func NewMockAuthorizer() *MockAuthorizer {
	return &MockAuthorizer{}
}

// Authorize is a mock implementation of the Authorize method.
func (m *MockAuthorizer) Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string, keyID domain.KeyID) (bool, string) {
	return true, "authorized"
}