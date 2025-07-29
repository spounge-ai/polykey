package authz

import (
	"context"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// Authorizer defines the interface for authorization checks.
type Authorizer interface {
	Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string) (bool, string)
}

// NewAuthorizer creates a new basic authorizer.
func NewAuthorizer() Authorizer {
	return &basicAuthorizer{}
}

// basicAuthorizer implements the Authorizer interface with simplified logic.
type basicAuthorizer struct{}

// Authorize performs a simplified authorization check.
func (a *basicAuthorizer) Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string) (bool, string) {
	clientIdentity := ""
	if reqContext != nil {
		clientIdentity = reqContext.GetClientIdentity()
	}

	// Deny access if operation is "restricted_key"
	if operation == "restricted_key" {
		return false, "mock_auth_decision_id_denied_restricted_key"
	}

	// Allow "test_creator" to perform "create_key_operation"
	if clientIdentity == "test_creator" && operation == "create_key_operation" {
		return true, "mock_auth_decision_id_granted_creator"
	}

	// Allow "test_client" for general operations
	if clientIdentity == "test_client" {
		return true, "mock_auth_decision_id_granted"
	}

	return false, "mock_auth_decision_id_denied_default"
}
