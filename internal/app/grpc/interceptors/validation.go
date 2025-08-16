package interceptors

import (
	"context"

	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/validation"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
)

// UnaryValidationInterceptor creates a gRPC unary interceptor that validates incoming requests.
func UnaryValidationInterceptor(errorClassifier *app_errors.ErrorClassifier) grpc.UnaryServerInterceptor {
	// These validators can be initialized once and reused as they are stateless.
	requestValidator, err := validation.NewRequestValidator()
	if err != nil {
		// If the validator fails to initialize, we return a function that will always fail.
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			classifiedErr := errorClassifier.Classify(err, "NewRequestValidator")
			return nil, errorClassifier.LogAndSanitize(ctx, classifiedErr)
		}
	}
	queryValidator := validation.NewQueryValidator()

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// TODO: Start OpenTelemetry span for validation
		// span := trace.SpanFromContext(ctx)
		// span.AddEvent("validation_start")
		// defer span.AddEvent("validation_end")

		var validationErr error

		switch r := req.(type) {
		case *pk.CreateKeyRequest:
			validationErr = requestValidator.ValidateCreateKeyRequest(ctx, r)
		case *pk.UpdateKeyMetadataRequest:
			validationErr = requestValidator.ValidateUpdateKeyMetadataRequest(ctx, r)
		case *pk.ListKeysRequest:
			validationErr = queryValidator.ValidateListKeysRequest(r)
		case *pk.GetKeyRequest:
			validationErr = requestValidator.ValidateGetKeyRequest(ctx, r)
		case *pk.RotateKeyRequest:
			validationErr = requestValidator.ValidateRotateKeyRequest(ctx, r)
		case *pk.RevokeKeyRequest:
			validationErr = requestValidator.ValidateRevokeKeyRequest(ctx, r)
		case *pk.GetKeyMetadataRequest:
			validationErr = requestValidator.ValidateGetKeyMetadataRequest(ctx, r)
		case *pk.AuthenticateRequest:
			validationErr = requestValidator.ValidateAuthenticateRequest(ctx, r)
		case *pk.RefreshTokenRequest:
			validationErr = requestValidator.ValidateRefreshTokenRequest(ctx, r)
		case *pk.RevokeTokenRequest:
			validationErr = requestValidator.ValidateRevokeTokenRequest(ctx, r)
		}

		if validationErr != nil {
			classifiedErr := errorClassifier.Classify(validationErr, info.FullMethod)
			return nil, errorClassifier.LogAndSanitize(ctx, classifiedErr)
		}

		return handler(ctx, req)
	}
}
