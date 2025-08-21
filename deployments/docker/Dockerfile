# syntax=docker/dockerfile:1.9

# --- Versions you can bump centrally ---
ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.22
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.40
ARG TPARSE_VERSION=v0.17.0

# --- Builder ---
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder
ARG TARGETOS
ARG TARGETARCH
ARG COMPRESS_BINARIES=false

# Consistent static builds + faster, reproducible linking
ENV CGO_ENABLED=0

# Build deps in a single layer
RUN apk --no-cache add git make upx ca-certificates curl && update-ca-certificates

WORKDIR /app

# Copy dependency files first for better caching
# NOTE: this assumes your build context is the repo root (so ../../ resolves).
COPY --link ../../go.mod ../../go.sum ./

# Download module deps with persistent caches
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download

# Copy source
COPY --link ../.. .

# Build the binary (trim paths, strip symbols; avoid VCS stamping for speed/repro)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -buildvcs=false -trimpath -ldflags="-w -s -buildid=" -o /app/bin/polykey cmd/polykey/main.go

# Optional UPX compression
RUN if [ "$COMPRESS_BINARIES" = "true" ]; then \
      upx --best --lzma /app/bin/polykey; \
    fi

# Install AWS CLI v2 with caching
RUN --mount=type=cache,target=/tmp/awscli-cache \
    set -eux; \
    case "${TARGETARCH:-amd64}" in \
      amd64) arch=x86_64 ;; \
      arm64) arch=aarch64 ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}"; exit 1 ;; \
    esac; \
    apk --no-cache add python3 py3-pip unzip curl; \
    AWS_CLI_ZIP=/tmp/awscli-cache/awscliv2.zip; \
    if [ ! -f "$AWS_CLI_ZIP" ]; then \
        curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${arch}.zip" -o "$AWS_CLI_ZIP"; \
    fi; \
    unzip -o "$AWS_CLI_ZIP" -d /tmp/awscli-cache; \
    /tmp/awscli-cache/aws/install --bin-dir /usr/local/bin --install-dir /usr/local/aws-cli; \
    rm -rf /tmp/awscli-cache/aws

# --- Pull grpc-health-probe from its official image (cache-friendly) ---
FROM ghcr.io/grpc-ecosystem/grpc-health-probe:${GRPC_HEALTH_PROBE_VERSION} AS grpc-health-probe

# --- Testing Stage ---
FROM builder AS tester
ARG TARGETARCH
ARG TPARSE_VERSION

# Use a pinned tparse binary instead of go install (faster & cacheable)
RUN set -eux; \
    case "${TARGETARCH:-amd64}" in \
      amd64) arch=amd64 ;; \
      arm64) arch=arm64 ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}"; exit 1 ;; \
    esac; \
    mkdir -p /usr/local/bin; \
    url="https://github.com/mfridman/tparse/releases/download/${TPARSE_VERSION}/tparse_linux_${arch}.tar.gz"; \
    curl -fsSL "$url" -o /tmp/tparse.tgz; \
    tar -xzf /tmp/tparse.tgz -C /usr/local/bin tparse; \
    chmod +x /usr/local/bin/tparse
CMD ["make", "test"]

# --- Production Image (smallest) ---
FROM gcr.io/distroless/static-debian12:nonroot AS production
COPY --link --from=builder /app/bin/polykey /polykey
COPY --link --from=grpc-health-probe --chmod=0755 /ko-app/grpc-health-probe /bin/grpc_health_probe
COPY --link ../../configs /etc/polykey/configs

EXPOSE 50053
HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD ["/bin/grpc_health_probe","-addr=localhost:50051"]
ENTRYPOINT ["/polykey"]

# --- Server Image (development/staging) ---
FROM alpine:${ALPINE_VERSION} AS server
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --link --from=grpc-health-probe --chmod=0755 /ko-app/grpc-health-probe /bin/grpc_health_probe
COPY --link --from=builder /app/bin/polykey ./polykey

EXPOSE 50053
HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD ["/bin/grpc_health_probe","-addr=localhost:50051"]
USER appuser:appgroup
ENTRYPOINT ["./polykey"]
