# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git protoc protobuf-dev

# Install protoc-gen-go
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code and proto files
COPY . .

# Generate protobuf code (if needed)
# RUN protoc --go_out=. --go-grpc_out=. proto/task/v1/task.proto

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o grpc-service .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/grpc-service .

# Expose ports (gRPC and metrics)
EXPOSE 9090 9091

# Run the binary
CMD ["./grpc-service"]
