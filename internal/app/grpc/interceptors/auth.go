package interceptors

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type requestData struct {
	requesterContext *pk.RequesterContext
	accessAttributes *pk.AccessAttributes
	keyID            domain.KeyID
}

type requesterContextGetter interface {
	GetRequesterContext() *pk.RequesterContext
}

type keyIDGetter interface {
	GetKeyId() string
}

func NewUnaryAuthInterceptor(
	authorizer domain.Authorizer,
	exemptMethods map[string]bool,
) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if exemptMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		reqData, err := extractRequestData(req)
		if err != nil {
			return nil, err
		}

		if reqData.requesterContext == nil {
			return nil, status.Error(codes.Unauthenticated, "missing requester context")
		}

		if isAuthorized, reason := authorizer.Authorize(
			ctx,
			reqData.requesterContext,
			reqData.accessAttributes,
			info.FullMethod,
			reqData.keyID,
		); !isAuthorized {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied: %s", reason)
		}

		return handler(ctx, req)
	}
}

func extractRequestData(req any) (*requestData, error) {
	data := &requestData{}

	rcReq, ok := req.(requesterContextGetter)
	if !ok {
		return nil, status.Errorf(codes.Unimplemented, "unsupported request type: %T", req)
	}
	data.requesterContext = rcReq.GetRequesterContext()

	if keyReq, ok := req.(keyIDGetter); ok {
		if err := setKeyFields(data, keyReq); err != nil {
			return nil, err
		}
	}

	return data, nil
}

func setKeyFields(data *requestData, keyReq keyIDGetter) error {
	keyIDStr := keyReq.GetKeyId()
	if keyIDStr == "" {
		return nil
	}

	parsedID, err := domain.KeyIDFromString(keyIDStr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid key id: %v", err)
	}

	data.keyID = parsedID
	data.accessAttributes = &pk.AccessAttributes{Environment: keyIDStr}
	return nil
}
