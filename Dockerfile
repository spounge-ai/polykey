# syntax=docker/dockerfile:1
FROM golang:1.24.1-alpine3.20 AS builder

ARG TARGETOS
ARG TARGETARCH

# Install build dependencies in single layer
RUN apk --no-cache add git make upx ca-certificates

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./

# Download dependencies with build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . .

# Build server binary only (dev_client runs locally)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s -buildid=" -trimpath -o bin/polykey cmd/polykey/main.go

# Install grpc-health-probe - use smaller binary
RUN wget -O /bin/grpc_health_probe \
    https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.38/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

# --- UPX Compression Stage (conditional) ---
FROM builder AS upx_compressor
ARG COMPRESS_BINARIES=false
RUN if [ "$COMPRESS_BINARIES" = "true" ]; then \
        upx --best --lzma /app/bin/polykey; \
    fi

# --- Testing Stage ---
FROM builder AS tester
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/mfridman/tparse@latest
CMD ["make", "test"]

# --- Production Image (smallest) ---
FROM gcr.io/distroless/static-debian12:nonroot AS production
ARG COMPRESS_BINARIES=false
COPY --from=upx_compressor /app/bin/polykey /polykey
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe
EXPOSE 50051
ENTRYPOINT ["/polykey"]

# --- Server Image (development/staging) ---
FROM alpine:3.20 AS server
# Only install what's actually needed
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /bin/grpc_health_probe /bin/
COPY --from=builder /app/bin/polykey .
EXPOSE 50051
USER appuser:appgroup
ENTRYPOINT ["./polykey"]