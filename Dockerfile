# Multi-stage Dockerfile for both server and dev client
FROM golang:1.24 AS builder


WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/polykey cmd/polykey/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/dev_client cmd/dev_client/main.go

# Server stage
FROM alpine:latest AS server
RUN apk --no-cache add ca-certificates netcat-openbsd
WORKDIR /root/
COPY --from=builder /app/bin/polykey .
EXPOSE 50051
CMD ["./polykey"]

# Dev client stage
FROM alpine:latest AS dev-client
RUN apk --no-cache add ca-certificates netcat-openbsd
WORKDIR /root/
COPY --from=builder /app/bin/dev_client .
CMD ["./dev_client"]

# Development stage (includes both binaries)
FROM alpine:latest AS development
RUN apk --no-cache add ca-certificates netcat-openbsd
WORKDIR /root/
COPY --from=builder /app/bin/polykey .
COPY --from=builder /app/bin/dev_client .
CMD ["./polykey"]