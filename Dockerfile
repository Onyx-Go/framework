# Build stage
FROM golang:1.21-alpine AS builder

# Set build arguments
ARG VERSION=dev
ARG BUILD_DATE
ARG GIT_COMMIT

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${GIT_COMMIT}" \
    -o onyx ./cmd/onyx

# Runtime stage
FROM alpine:3.22

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S onyx && \
    adduser -u 1001 -S onyx -G onyx

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/onyx .

# Copy configuration files (if any)
COPY --from=builder /app/config ./config

# Copy static assets and views (if any)
COPY --from=builder /app/resources ./resources

# Create directories for logs and storage
RUN mkdir -p storage/logs storage/cache && \
    chown -R onyx:onyx /app

# Switch to non-root user
USER onyx

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set default command
CMD ["./onyx", "serve", "--port=8080"]