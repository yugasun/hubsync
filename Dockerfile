FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy only necessary files for go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/yugasun/hubsync/cmd/hubsync.Version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")" -o hubsync ./cmd/hubsync

# Create lightweight runtime image
FROM alpine:3.19

# Add necessary tools
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/hubsync /usr/local/bin/

# Set up working directory
WORKDIR /data

# Create entrypoint script
RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo 'if [ "$1" = "--help" ] || [ -z "$1" ]; then' >> /entrypoint.sh && \
    echo '  hubsync --help' >> /entrypoint.sh && \
    echo 'else' >> /entrypoint.sh && \
    echo '  hubsync "$@"' >> /entrypoint.sh && \
    echo 'fi' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]