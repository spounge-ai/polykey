package errors

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorClass int

const (
	ClassInternal ErrorClass = iota
	ClassValidation
	ClassAuthentication
	ClassAuthorization
	ClassNotFound
	ClassConflict
	ClassRateLimit
	ClassExternal
)

type ClassifiedError struct {
	Class         ErrorClass
	InternalError error
	ClientMessage string
	OperationName string
	KeyID         string // Store but never expose
	Metadata      map[string]interface{}
}

type ErrorClassifier struct {
	logger *slog.Logger
}

func NewErrorClassifier(logger *slog.Logger) *ErrorClassifier {
	return &ErrorClassifier{logger: logger}
}

var errorPool = sync.Pool{
	New: func() interface{} {
		return &ClassifiedError{
			Metadata: make(map[string]interface{}, 4), // Pre-size for common case
		}
	},
}

func (ec *ErrorClassifier) Classify(err error, operation string) *ClassifiedError {
	// Get a pooled error object
	classified := errorPool.Get().(*ClassifiedError)
	classified.InternalError = err
	classified.OperationName = operation

	switch {
	case errors.Is(err, ErrKeyNotFound):
		classified.Class = ClassNotFound
		classified.ClientMessage = "The requested resource was not found"
	case errors.Is(err, ErrInvalidInput):
		classified.Class = ClassValidation
		classified.ClientMessage = "The request contains invalid parameters"
	case errors.Is(err, ErrKMSFailure):
		classified.Class = ClassInternal
		classified.ClientMessage = "An internal error occurred. Please try again later"
	case errors.Is(err, ErrAuthentication):
		classified.Class = ClassAuthentication
		classified.ClientMessage = "Authentication failed"
	case errors.Is(err, ErrAuthorization):
		classified.Class = ClassAuthorization
		classified.ClientMessage = "Permission denied"
	case errors.Is(err, ErrConflict):
		classified.Class = ClassConflict
		classified.ClientMessage = "A conflict occurred"
	case errors.Is(err, ErrRateLimit):
		classified.Class = ClassRateLimit
		classified.ClientMessage = "You have exceeded the rate limit"
	default:
		classified.Class = ClassInternal
		classified.ClientMessage = "An unexpected internal error occurred"
	}

	return classified
}

func (ec *ErrorClassifier) LogAndSanitize(ctx context.Context, classified *ClassifiedError) error {
	defer ec.putError(classified) // Return the object to the pool

	ec.logger.ErrorContext(ctx, "operation failed",
		"operation", classified.OperationName,
		"error_class", classified.Class,
		"internal_error", classified.InternalError.Error(),
		"key_id", classified.KeyID, // Log internally only
		"metadata", classified.Metadata,
	)

	return ec.toGRPCError(classified)
}

func (ec *ErrorClassifier) toGRPCError(classified *ClassifiedError) error {
	var code codes.Code

	switch classified.Class {
	case ClassNotFound:
		code = codes.NotFound
	case ClassValidation:
		code = codes.InvalidArgument
	case ClassAuthentication:
		code = codes.Unauthenticated
	case ClassAuthorization:
		code = codes.PermissionDenied
	case ClassRateLimit:
		code = codes.ResourceExhausted
	case ClassConflict:
		code = codes.AlreadyExists
	default:
		code = codes.Internal
	}

	return status.Error(code, classified.ClientMessage)
}

func (ec *ErrorClassifier) putError(err *ClassifiedError) {
	// Clear sensitive data before pooling
	err.KeyID = ""
	err.InternalError = nil
	for k := range err.Metadata {
		delete(err.Metadata, k)
	}
	err.OperationName = ""
	errorPool.Put(err)
}
