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
	Metadata      map[string]any
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

var classificationRules = []struct {
	targetErr     error
	class         ErrorClass
	clientMessage string
}{
	{ErrKeyNotFound, ClassNotFound, "The requested resource was not found"},
	{ErrInvalidInput, ClassValidation, "The request contains invalid parameters"},
	{ErrKMSFailure, ClassInternal, "An internal error occurred. Please try again later"},
	{ErrAuthentication, ClassAuthentication, "Authentication failed"},
	{ErrAuthorization, ClassAuthorization, "Permission denied"},
	{ErrConflict, ClassConflict, "A conflict occurred"},
	{ErrRateLimit, ClassRateLimit, "You have exceeded the rate limit"},
	{ErrExternal, ClassExternal, "External service temporarily unavailable"},
}

func (ec *ErrorClassifier) Classify(err error, operation string) *ClassifiedError {
	if err == nil {
		return nil
	}

	classified := errorPool.Get().(*ClassifiedError)
	classified.InternalError = err
	classified.OperationName = operation

	for _, rule := range classificationRules {
		if errors.Is(err, rule.targetErr) {
			classified.Class = rule.class
			classified.ClientMessage = rule.clientMessage
			return classified
		}
	}

	classified.Class = ClassInternal
	classified.ClientMessage = "An unexpected internal error occurred"
	return classified
}

func (ec *ErrorClassifier) LogAndSanitize(ctx context.Context, classified *ClassifiedError) error {
	if classified == nil {
		return nil
	}

	defer ec.putError(classified) 

	attrs := []slog.Attr{
		slog.String("operation", classified.OperationName),
		slog.Int("error_class", int(classified.Class)),
		slog.String("internal_error", classified.InternalError.Error()),
	}

	if classified.KeyID != "" {
		attrs = append(attrs, slog.String("key_id", classified.KeyID))
	}
	if len(classified.Metadata) > 0 {
		attrs = append(attrs, slog.Any("metadata", classified.Metadata))
	}

	ec.logger.LogAttrs(ctx, slog.LevelError, "operation failed", attrs...)

	return ec.toGRPCError(classified)
}

var grpcCodeMap = map[ErrorClass]codes.Code{
	ClassNotFound:       codes.NotFound,
	ClassValidation:     codes.InvalidArgument,
	ClassAuthentication: codes.Unauthenticated,
	ClassAuthorization:  codes.PermissionDenied,
	ClassRateLimit:      codes.ResourceExhausted,
	ClassConflict:       codes.AlreadyExists,
	ClassExternal:       codes.Unavailable,
	ClassInternal:       codes.Internal, 
}

func (ec *ErrorClassifier) toGRPCError(classified *ClassifiedError) error {
	code, exists := grpcCodeMap[classified.Class]
	if !exists {
		code = codes.Internal
	}

	return status.Error(code, classified.ClientMessage)
}

func (ec *ErrorClassifier) putError(err *ClassifiedError) {
	err.KeyID = ""
	err.InternalError = nil
	
	for k := range err.Metadata {
		delete(err.Metadata, k)
	}
	
	err.OperationName = ""
	err.ClientMessage = ""
	err.Class = 0
	
	errorPool.Put(err)
}