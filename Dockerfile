FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o main .

# Use specific Alpine version for consistency
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary and set ownership
COPY --from=builder --chown=appuser:appgroup /app/main .

# Ensure binary is executable
RUN chmod +x main

# Switch to non-root user
USER appuser

# Expose port if your app serves HTTP (adjust as needed)
# EXPOSE 8080

CMD ["./main"]