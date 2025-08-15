package interceptors

import (
	"context"
	"fmt"

	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/validation"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
)

type requestValidator interface {
	ValidateCreateKeyRequest(ctx context.Context, req *pk.CreateKeyRequest) error
	ValidateUpdateKeyMetadataRequest(ctx context.Context, req *pk.UpdateKeyMetadataRequest) error
	ValidateGetKeyRequest(ctx context.Context, req *pk.GetKeyRequest) error
	ValidateRotateKeyRequest(ctx context.Context, req *pk.RotateKeyRequest) error
	ValidateRevokeKeyRequest(ctx context.Context, req *pk.RevokeKeyRequest) error
	ValidateGetKeyMetadataRequest(ctx context.Context, req *pk.GetKeyMetadataRequest) error
	ValidateAuthenticateRequest(ctx context.Context, req *pk.AuthenticateRequest) error
	ValidateRefreshTokenRequest(ctx context.Context, req *pk.RefreshTokenRequest) error
	ValidateRevokeTokenRequest(ctx context.Context, req *pk.RevokeTokenRequest) error
}

type queryValidator interface {
	ValidateListKeysRequest(req *pk.ListKeysRequest) error
}

type validationFunc func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error

var validationRegistry = map[string]validationFunc{
	"*polykey.CreateKeyRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return reqValidator.ValidateCreateKeyRequest(ctx, req.(*pk.CreateKeyRequest))
	},
	"*polykey.UpdateKeyMetadataRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return reqValidator.ValidateUpdateKeyMetadataRequest(ctx, req.(*pk.UpdateKeyMetadataRequest))
	},
	"*polykey.ListKeysRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return queryValidator.ValidateListKeysRequest(req.(*pk.ListKeysRequest))
	},
	"*polykey.GetKeyRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return reqValidator.ValidateGetKeyRequest(ctx, req.(*pk.GetKeyRequest))
	},
	"*polykey.RotateKeyRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return reqValidator.ValidateRotateKeyRequest(ctx, req.(*pk.RotateKeyRequest))
	},
	"*polykey.RevokeKeyRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return reqValidator.ValidateRevokeKeyRequest(ctx, req.(*pk.RevokeKeyRequest))
	},
	"*polykey.GetKeyMetadataRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return reqValidator.ValidateGetKeyMetadataRequest(ctx, req.(*pk.GetKeyMetadataRequest))
	},
	"*polykey.AuthenticateRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return reqValidator.ValidateAuthenticateRequest(ctx, req.(*pk.AuthenticateRequest))
	},
	"*polykey.RefreshTokenRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return reqValidator.ValidateRefreshTokenRequest(ctx, req.(*pk.RefreshTokenRequest))
	},
	"*polykey.RevokeTokenRequest": func(ctx context.Context, req interface{}, reqValidator requestValidator, queryValidator queryValidator) error {
		return reqValidator.ValidateRevokeTokenRequest(ctx, req.(*pk.RevokeTokenRequest))
	},
}

func UnaryValidationInterceptor(errorClassifier *app_errors.ErrorClassifier) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestValidator, err := validation.NewRequestValidator()
		if err != nil {
			classifiedErr := errorClassifier.Classify(err, "NewRequestValidator")
			return nil, errorClassifier.LogAndSanitize(ctx, classifiedErr)
		}

		queryValidator := validation.NewQueryValidator()

		reqTypeName := fmt.Sprintf("%T", req)
		if validationFunc, exists := validationRegistry[reqTypeName]; exists {
			if validationErr := validationFunc(ctx, req, requestValidator, queryValidator); validationErr != nil {
				classifiedErr := errorClassifier.Classify(validationErr, info.FullMethod)
				return nil, errorClassifier.LogAndSanitize(ctx, classifiedErr)
			}
		}

		return handler(ctx, req)
	}
}