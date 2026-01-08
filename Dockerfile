# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api cmd/api/main.go

# Production stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS calls and tzdata for timezone
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
# Using UID 1000 to match typical Linux user for easier volume permissions
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Copy binary from builder
COPY --from=builder /app/bin/api .

# Copy migrations (needed for auto-migration on startup)
COPY --from=builder /app/migrations ./migrations

# Change ownership
RUN chown -R appuser:appgroup /app

USER appuser

# Expose port (default 8080, configurable via PORT env)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/health || exit 1

CMD ["./api"]
