package auth

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// NewMockAuthorizer creates a new basic authorizer.
func NewMockAuthorizer() domain.Authorizer {
	return &basicAuthorizer{}
}

// basicAuthorizer implements the Authorizer interface with simplified logic.
type basicAuthorizer struct{}

// Authorize performs a simplified authorization check.
func (a *basicAuthorizer) Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string, keyID domain.KeyID) (bool, string) {
	clientIdentity := ""
	if reqContext != nil {
		clientIdentity = reqContext.GetClientIdentity()
	}

	// Explicitly deny unauthorized operations based on test cases
	if operation == "/polykey.v2.PolykeyService/GetKey" && keyID.String() == "c47ac10b-58cc-4372-a567-0e02b2c3d479" {
		return false, "mock_auth_decision_id_denied_restricted_key"
	}

	if (operation == "/polykey.v2.PolykeyService/CreateKey" && clientIdentity == "unknown_creator") ||
		(operation == "/polykey.v2.PolykeyService/GetKeyMetadata" && clientIdentity == "unknown_client") ||
		(operation == "/polykey.v2.PolykeyService/GetKeyMetadata" && keyID.String() == "d47ac10b-58cc-4372-a567-0e02b2c3d479") {
		return false, "mock_auth_decision_id_denied_unauthorized"
	}

	// Allow HealthCheck, GetKeyMetadata, GetKey, CreateKey
	if operation == "/polykey.v2.PolykeyService/HealthCheck" || 
	   operation == "/polykey.v2.PolykeyService/GetKeyMetadata" || 
	   operation == "/polykey.v2.PolykeyService/GetKey" || 
	   operation == "/polykey.v2.PolykeyService/CreateKey" {
		return true, "mock_auth_decision_id_granted_general"
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
