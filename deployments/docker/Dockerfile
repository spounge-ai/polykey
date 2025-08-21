FROM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG COMPRESS_BINARIES=false

# Install build dependencies in single layer
RUN apk --no-cache add git make upx ca-certificates

WORKDIR /app

# Copy dependency files first for better caching
COPY ../../go.mod ../../go.sum ./

# Download dependencies with build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY ../.. .

# Build polykey binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-w -s -buildid=" -trimpath -o bin/polykey cmd/polykey/main.go

# Download grpc-health-probe in parallel-friendly way
RUN --mount=type=cache,target=/tmp/downloads \
    if [ ! -f /tmp/downloads/grpc_health_probe-linux-amd64 ]; then \
        wget -O /tmp/downloads/grpc_health_probe-linux-amd64 \
        https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.39/grpc_health_probe-linux-amd64; \
    fi && \
    cp /tmp/downloads/grpc_health_probe-linux-amd64 /bin/grpc_health_probe && \
    chmod +x /bin/grpc_health_probe

# Conditional UPX compression
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
COPY --from=builder /app/bin/polykey /polykey
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe
COPY ../../certs /etc/polykey/certs
EXPOSE 50051
ENTRYPOINT ["/polykey"]

# --- Server Image (development/staging) ---
FROM alpine:3.20 AS server
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /bin/grpc_health_probe /bin/
COPY --from=builder /app/bin/polykey .
COPY ../../certs /etc/polykey/certs
EXPOSE 50051
USER appuser:appgroup
ENTRYPOINT ["./polykey"]