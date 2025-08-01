package domain

import (
	"context"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// Authorizer defines the interface for authorization checks.
type Authorizer interface {
	Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string) (bool, string)
}
