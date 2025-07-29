package integration

import (
	"context"
	"testing"

	"github.com/spounge-ai/polykey/internal/authz"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/stretchr/testify/assert"
)

func newRequesterContext(clientIdentity string) *pk.RequesterContext {
	return &pk.RequesterContext{ClientIdentity: clientIdentity}
}

func TestAuthorize(t *testing.T) {
	authorizer := authz.NewAuthorizer()

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