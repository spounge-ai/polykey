FROM golang:1.24.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /polykey-server ./cmd/polykey

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /polykey-server /usr/local/bin/polykey-server

EXPOSE 50051

ENTRYPOINT ["/usr/local/bin/polykey-server"]
