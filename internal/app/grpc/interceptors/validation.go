package interceptors

import (
	"context"
	"reflect"

	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/validation"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
)

// validatorFunc defines a generic function type for request validation.
type validatorFunc func(context.Context, any) error

// UnaryValidationInterceptor creates a gRPC unary interceptor that validates incoming requests.
func UnaryValidationInterceptor(errorClassifier *app_errors.ErrorClassifier) grpc.UnaryServerInterceptor {
	requestValidator, err := validation.NewRequestValidator()
	if err != nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			classifiedErr := errorClassifier.Classify(err, "NewRequestValidator")
			return nil, errorClassifier.LogAndSanitize(ctx, classifiedErr)
		}
	}
	queryValidator := validation.NewQueryValidator()

	// validationMap maps request types to their specific validation functions.
	validationMap := map[reflect.Type]validatorFunc{
		reflect.TypeOf(&pk.CreateKeyRequest{}): func(ctx context.Context, r any) error {
			return requestValidator.ValidateCreateKeyRequest(ctx, r.(*pk.CreateKeyRequest))
		},
		reflect.TypeOf(&pk.UpdateKeyMetadataRequest{}): func(ctx context.Context, r any) error {
			return requestValidator.ValidateUpdateKeyMetadataRequest(ctx, r.(*pk.UpdateKeyMetadataRequest))
		},
		reflect.TypeOf(&pk.ListKeysRequest{}): func(ctx context.Context, r any) error {
			return queryValidator.ValidateListKeysRequest(r.(*pk.ListKeysRequest))
		},
		reflect.TypeOf(&pk.GetKeyRequest{}): func(ctx context.Context, r any) error {
			return requestValidator.ValidateGetKeyRequest(ctx, r.(*pk.GetKeyRequest))
		},
		reflect.TypeOf(&pk.RotateKeyRequest{}): func(ctx context.Context, r any) error {
			return requestValidator.ValidateRotateKeyRequest(ctx, r.(*pk.RotateKeyRequest))
		},
		reflect.TypeOf(&pk.RevokeKeyRequest{}): func(ctx context.Context, r any) error {
			return requestValidator.ValidateRevokeKeyRequest(ctx, r.(*pk.RevokeKeyRequest))
		},
		reflect.TypeOf(&pk.GetKeyMetadataRequest{}): func(ctx context.Context, r any) error {
			return requestValidator.ValidateGetKeyMetadataRequest(ctx, r.(*pk.GetKeyMetadataRequest))
		},
		reflect.TypeOf(&pk.AuthenticateRequest{}): func(ctx context.Context, r any) error {
			return requestValidator.ValidateAuthenticateRequest(ctx, r.(*pk.AuthenticateRequest))
		},
		reflect.TypeOf(&pk.RefreshTokenRequest{}): func(ctx context.Context, r any) error {
			return requestValidator.ValidateRefreshTokenRequest(ctx, r.(*pk.RefreshTokenRequest))
		},
		reflect.TypeOf(&pk.RevokeTokenRequest{}): func(ctx context.Context, r any) error {
			return requestValidator.ValidateRevokeTokenRequest(ctx, r.(*pk.RevokeTokenRequest))
		},
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if validator, ok := validationMap[reflect.TypeOf(req)]; ok {
			if err := validator(ctx, req); err != nil {
				classifiedErr := errorClassifier.Classify(err, info.FullMethod)
				return nil, errorClassifier.LogAndSanitize(ctx, classifiedErr)
			}
		}

		return handler(ctx, req)
	}
}