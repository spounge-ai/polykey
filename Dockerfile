# --- Builder Stage ---
# Use a Go image with the necessary tools
FROM golang:1.24.1-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Install protoc and Go plugins if you generate code during the build
# If you commit your generated code (polykey.pb.go, polykey_grpc.pb.go), you can skip this step
RUN apk add --no-cache protobuf
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate protobuf code (if not committed)
# RUN protoc --go_out=. --go_opt=paths=source_relative \
#            --go-grpc_out=. --go-grpc_opt=paths=source_relative \
#            proto/polykey.proto

# Build the server executable
# CGO_ENABLED=0 is often used for static binaries, good for smaller images
RUN CGO_ENABLED=0 go build -o /polykey-server ./cmd/polykey

# --- Final Stage ---
# Use a minimal base image
FROM alpine:latest

# Install ca-certificates for TLS connections if needed (even for gRPC)
RUN apk --no-cache add ca-certificates

# Copy the built executable from the builder stage
COPY --from=builder /polykey-server /usr/local/bin/polykey-server

# Expose the gRPC port
EXPOSE 50051

# Set the entrypoint to run the server executable
ENTRYPOINT ["/usr/local/bin/polykey-server"]