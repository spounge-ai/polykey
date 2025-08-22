 # --------------------------------------------------------------------
# Version Arguments  
# --------------------------------------------------------------------
ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.22
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.40
ARG TPARSE_VERSION=v0.17.0

# --------------------------------------------------------------------
# Builder Stage
# --------------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder
ARG TARGETOS
ARG TARGETARCH
ARG COMPRESS_BINARIES=false

# Ensure reproducible static builds
ENV CGO_ENABLED=0

# Install build dependencies
RUN apk --no-cache add git make upx ca-certificates curl && update-ca-certificates

WORKDIR /app

# Copy Go dependency files and download modules (cache-enabled)
COPY --link ../../go.mod ../../go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download

# Copy source code
COPY --link ../.. .

# Build Polykey binary
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -buildvcs=false -trimpath -ldflags="-w -s -buildid=" \
        -o /app/bin/polykey cmd/polykey/main.go

# Optionally compress binary with UPX
RUN if [ "$COMPRESS_BINARIES" = "true" ]; then \
      upx --best --lzma /app/bin/polykey; \
    fi

# --------------------------------------------------------------------
# AWS CLI Stage
# --------------------------------------------------------------------
FROM alpine:${ALPINE_VERSION} AS awscli
ARG AWSCLI_VERSION=2.22.35
ARG TARGETARCH

RUN apk add --no-cache curl unzip && \
    case "${TARGETARCH:-amd64}" in \
      amd64) arch=x86_64 ;; \
      arm64) arch=aarch64 ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${arch}-${AWSCLI_VERSION}.zip" -o awscliv2.zip && \
    unzip awscliv2.zip && \
    ./aws/install --install-dir /aws-cli --bin-dir /aws-cli-bin

# --------------------------------------------------------------------
# gRPC Health Probe Stage
# --------------------------------------------------------------------
FROM builder AS grpc-health-probe
ARG GRPC_HEALTH_PROBE_VERSION
ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go install -trimpath -ldflags="-w -s -buildid=" \
        github.com/grpc-ecosystem/grpc-health-probe@${GRPC_HEALTH_PROBE_VERSION}

# --------------------------------------------------------------------
# Tester Stage
# --------------------------------------------------------------------
FROM builder AS tester
ARG TARGETARCH
ARG TPARSE_VERSION

# Install tparse binary from release (instead of go install)
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

# --------------------------------------------------------------------
# Production Image
# --------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS production

# Copy binaries and configuration files
COPY --link --from=builder /app/bin/polykey /polykey
COPY --link --from=grpc-health-probe /go/bin/grpc-health-probe /bin/grpc_health_probe
COPY --link ../../configs /etc/polykey/configs

EXPOSE 50053
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD ["/bin/grpc_health_probe","-addr=localhost:50051"]

ENTRYPOINT ["/polykey"]

# --------------------------------------------------------------------
# Server Image (development and staging)
# --------------------------------------------------------------------
FROM alpine:${ALPINE_VERSION} AS server

# Add runtime user and install certificates
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binaries
COPY --link --from=grpc-health-probe /go/bin/grpc-health-probe /bin/grpc_health_probe
COPY --link --from=builder /app/bin/polykey ./polykey
COPY --link --from=builder /app/bin/dev_client ./dev_client

EXPOSE 50053
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD ["/bin/grpc_health_probe","-addr=localhost:50053"]

USER appuser:appgroup
ENTRYPOINT ["./polykey"]
