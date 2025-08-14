package interceptors

import (
	"context"

	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/validation"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
)

func UnaryValidationInterceptor(errorClassifier *app_errors.ErrorClassifier) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestValidator, err := validation.NewRequestValidator()
		if err != nil {
			// This is a server configuration error, should be handled appropriately
			return nil, errorClassifier.LogAndSanitize(ctx, errorClassifier.Classify(err, "NewRequestValidator"))
		}

		queryValidator := validation.NewQueryValidator()

		var validationErr error

		switch r := req.(type) {
		case *pk.CreateKeyRequest:
			validationErr = requestValidator.ValidateCreateKeyRequest(ctx, r)
		case *pk.UpdateKeyMetadataRequest:
			validationErr = requestValidator.ValidateUpdateKeyMetadataRequest(ctx, r)
		case *pk.ListKeysRequest:
			validationErr = queryValidator.ValidateListKeysRequest(r)
		}

		if validationErr != nil {
			classifiedErr := errorClassifier.Classify(validationErr, info.FullMethod)
			return nil, errorClassifier.LogAndSanitize(ctx, classifiedErr)
		}

		return handler(ctx, req)
	}
}
