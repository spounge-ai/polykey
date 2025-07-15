FROM golang:1.24.1-alpine3.20 AS builder

ARG TARGETOS TARGETARCH

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN apk --no-cache add git make upx

# Use the official, recommended method to install the health probe
COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.38 /ko-app/grpc-health-probe /bin/grpc_health_probe

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -ldflags="-w -s" -trimpath -a -installsuffix cgo -o bin/polykey cmd/polykey/main.go && \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -ldflags="-w -s" -trimpath -a -installsuffix cgo -o bin/dev_client cmd/dev_client/main.go

# --- UPX Compression Stage ---
FROM builder AS upx_compressor
RUN upx --best --lzma /app/bin/polykey && \
    upx --best --lzma /app/bin/dev_client

FROM builder AS tester

RUN go install github.com/mfridman/tparse@latest

CMD ["make", "test"]

FROM alpine:3.20 AS server
RUN apk --no-cache add ca-certificates netcat-openbsd
WORKDIR /app/
COPY --from=builder /bin/grpc_health_probe /bin/
COPY --from=builder /app/bin/polykey .
EXPOSE 50051
CMD ["./polykey"]

FROM alpine:3.20 AS dev-client
RUN apk --no-cache add ca-certificates netcat-openbsd
WORKDIR /app/
COPY --from=builder /app/bin/dev_client .
CMD ["./dev_client"]

FROM scratch AS production

COPY --from=alpine:3.20 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
# Also copy the health probe to the production image
COPY --from=builder /bin/grpc_health_probe /bin/
COPY --from=upx_compressor --chown=appuser:appgroup /app/bin/polykey /polykey

USER appuser:appgroup
EXPOSE 50051
CMD ["/polykey"]
