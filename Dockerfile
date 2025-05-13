FROM golang:1.23-alpine AS builder

WORKDIR /app

# Accept version as build argument
ARG VERSION=dev

# Copy only necessary files for go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with version from build arg
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w \
    -X main.Version=${VERSION}" \
    -o hubsync ./cmd/hubsync

# Create lightweight runtime image
FROM alpine:3.19

# Add version label to image
ARG VERSION=dev
LABEL org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.authors="Yuga Sun" \
      org.opencontainers.image.source="https://github.com/yugasun/hubsync"

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