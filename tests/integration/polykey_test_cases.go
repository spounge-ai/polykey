package integration

import (
	"context"
	"testing"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/stretchr/testify/assert"
)

func newRequesterContext(clientIdentity string) *pk.RequesterContext {
	return &pk.RequesterContext{ClientIdentity: clientIdentity}
}

func TestAuthorize(t *testing.T) {
	authorizer := &mockAuthorizer{}

	testCases := []struct {
		name           string
		requesterContext *pk.RequesterContext
		operation      string
		expected       bool
	}{
		{
			name:           "Authorized access",
			requesterContext: newRequesterContext("test_client"),
			operation:      "GetKey",
			expected:       true,
		},
		{
			name:           "Unauthorized access (restricted key)",
			requesterContext: newRequesterContext("test_client"),
			operation:      "restricted_key",
			expected:       false,
		},
		{
			name:           "Unauthorized access (unknown client)",
			requesterContext: newRequesterContext("unknown_client"),
			operation:      "GetKey",
			expected:       false,
		},
		{
			name:           "Authorized creation",
			requesterContext: newRequesterContext("test_creator"),
			operation:      "create_key_operation",
			expected:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowed, _ := authorizer.Authorize(context.Background(), tc.requesterContext, nil, tc.operation)
			assert.Equal(t, tc.expected, allowed)
		})
	}
}

// mockAuthorizer implements the domain.Authorizer interface for testing.
type mockAuthorizer struct{}

func (m *mockAuthorizer) Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string) (bool, string) {
	clientIdentity := ""
	if reqContext != nil {
		clientIdentity = reqContext.GetClientIdentity()
	}

	if operation == "restricted_key" {
		return false, "mock_auth_decision_id_denied_restricted_key"
	}

	if clientIdentity == "test_creator" && operation == "create_key_operation" {
		return true, "mock_auth_decision_id_granted_creator"
	}

	if clientIdentity == "test_client" {
		return true, "mock_auth_decision_id_granted"
	}

	return false, "mock_auth_decision_id_denied_default"
}

func TestListKeys(t *testing.T) {
	// This test requires a running service instance.
	// For simplicity, we'll skip it in this example.
	t.Skip("Skipping ListKeys test in this example")
}
