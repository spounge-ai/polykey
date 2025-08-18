package core

import (
	cmn "github.com/spounge-ai/spounge-proto/gen/go/common/v2"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// DefaultRequesterContext returns a default requester context with no tier specified.
func DefaultRequesterContext(clientID string) *pk.RequesterContext {
	return &pk.RequesterContext{
		ClientIdentity: clientID,
	}
}

// EnterpriseRequesterContext returns an enterprise requester context.
func EnterpriseRequesterContext(clientID string) *pk.RequesterContext {
	return &pk.RequesterContext{
		ClientIdentity: clientID,
		ClientTier:     cmn.ClientTier_CLIENT_TIER_ENTERPRISE,
	}
}
