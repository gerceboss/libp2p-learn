# LibP2P Node Dockerfile
# Multi-stage build for optimized production image

# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install necessary packages
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o libp2p-node .

# Production stage
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/libp2p-node /libp2p-node

# Create non-root user
USER 1000:1000

# Expose ports
# TCP port for libp2p
EXPOSE 8080
# UDP port for QUIC
EXPOSE 8080/udp

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD /libp2p-node --help > /dev/null || exit 1

# Default command
ENTRYPOINT ["/libp2p-node"]
CMD ["--port", "8080"]

# Metadata
LABEL maintainer="LibP2P Learn Project"
LABEL description="A libp2p node with TCP/UDP support and hole punching"
LABEL version="1.0.0" 