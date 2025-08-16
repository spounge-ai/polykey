package interceptors

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type correlationIDKey struct{}

func UnaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		correlationID := uuid.New().String()
		ctx = context.WithValue(ctx, correlationIDKey{}, correlationID)

		resp, err := handler(ctx, req)
		duration := time.Since(start)

		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Unknown
			}
		}

		// Use With() for structured attributes
		logEntry := logger.With(
			slog.String("correlation_id", correlationID),
			slog.String("method", info.FullMethod),
			slog.Duration("duration", duration),
			slog.String("status_code", statusCode.String()),
		)

		if err != nil {
			logEntry.WarnContext(ctx, "gRPC request failed", slog.String("error", err.Error()))
		} else {
			logEntry.InfoContext(ctx, "gRPC request completed")
		}

		return resp, err
	}
}

func CorrelationIDFromContext(ctx context.Context) string {
	if correlationID, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return correlationID
	}
	return ""
}