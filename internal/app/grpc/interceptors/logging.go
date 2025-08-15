package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

func UnaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		h, err := handler(ctx, req)

		logger.InfoContext(ctx, "gRPC request",
			"method", info.FullMethod,
			"duration", time.Since(start),
			"error", err,
		)

		return h, err
	}
}
