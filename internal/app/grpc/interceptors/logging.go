package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		h, err := handler(ctx, req)
		duration := time.Since(start)

		var statusCode string
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code().String()
			} else {
				statusCode = "UNKNOWN"
			}
		} else {
			statusCode = "OK"
		}

		if err != nil {
			logger.WarnContext(ctx, "gRPC request failed",
				"method", info.FullMethod,
				"duration", duration,
				"status_code", statusCode,
				"error", err.Error(),
			)
		} else {
			logger.InfoContext(ctx, "gRPC request",
				"method", info.FullMethod,
				"duration", duration,
				"status_code", statusCode,
			)
		}

		return h, err
	}
}