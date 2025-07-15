# Use a pinned version of the golang image for reproducible builds
FROM golang:1.24.1-alpine3.20 AS builder

# Add a non-root user and group for the final production image
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Install git for private modules and build tools
RUN apk --no-cache add git make

# Install gRPC health probe using the correct, versioned path
RUN go install github.com/grpc-ecosystem/grpc-health-probe@latest


WORKDIR /app

# Copy go mod files first to leverage Docker layer caching
COPY go.mod go.sum ./

# Download dependencies with cache mounts
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy the rest of the source code
COPY . .

# Build both binaries in a single, cached RUN command
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -ldflags="-w -s" -a -installsuffix cgo -o bin/polykey cmd/polykey/main.go && \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -ldflags="-w -s" -a -installsuffix cgo -o bin/dev_client cmd/dev_client/main.go

# --- Test Stage ---
# This stage includes all necessary tools for running tests
FROM builder AS tester

# Pre-install testing tools to a separate layer for better caching
RUN go install github.com/mfridman/tparse@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# The entrypoint can be a make target that runs all tests
CMD ["make", "test"]


# --- Development Server Stage ---
# An Alpine-based image for development, includes netcat and health probe
FROM alpine:3.20 AS server
RUN apk --no-cache add ca-certificates netcat-openbsd
WORKDIR /app/
# Copy the compiled binary and the health probe
COPY --from=builder /go/bin/grpc-health-probe /bin/
COPY --from=builder /app/bin/polykey .
EXPOSE 50051
CMD ["./polykey"]


# --- Development Client Stage ---
# An Alpine-based image for the development client
FROM alpine:3.20 AS dev-client
RUN apk --no-cache add ca-certificates netcat-openbsd
WORKDIR /app/
COPY --from=builder /app/bin/dev_client .
CMD ["./dev_client"]


# --- Production Stage ---
# A minimal scratch image for the final production artifact
FROM scratch AS production

# Copy only the necessary files for a minimal image
COPY --from=alpine:3.20 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the binary with the correct non-root owner
COPY --from=builder --chown=appuser:appgroup /app/bin/polykey /polykey

# Run as the non-root user
USER appuser:appgroup

# Expose the correct port
EXPOSE 50051

# Set the entrypoint
CMD ["/polykey"]
