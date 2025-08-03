# Multi-stage Dockerfile for KubeGraph
# This Dockerfile expects a pre-built binary to be copied in

# Build stage (optional - can be used for building if needed)
FROM golang:1.24.4-alpine AS builder

WORKDIR /app

# Install git and build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Copy source code
COPY . .

# Build the application (this stage can be used if building in Docker is preferred)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kubegraph .

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy the binary from builder stage (or from local build)
COPY --from=builder /app/kubegraph .

# Run as non-root user
RUN adduser -D -g '' appuser
USER appuser

ENTRYPOINT ["./kubegraph"] 
