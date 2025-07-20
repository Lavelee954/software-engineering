# Build stage
FROM golang:1.24-alpine AS builder

# Install required packages
RUN apk add --no-cache git ca-certificates tzdata

# Create app directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy only necessary source files
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o bin/system-trading \
    ./cmd/main.go

# Runtime stage
FROM scratch

# Add metadata
LABEL maintainer="trading-team@company.com"
LABEL description="Trading System Core - Go 1.24 Application"
LABEL version="1.0.0"
LABEL go.version="1.24"

# Copy ca-certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/bin/system-trading /system-trading

# Create non-root user (even in scratch)
USER 1001:1001

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD ["/system-trading", "healthcheck"] || exit 1

# Set environment
ENV GIN_MODE=release

# Run the application
ENTRYPOINT ["/system-trading"]