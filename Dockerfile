# Stage 1: Builder with OpenCV
FROM gocv/opencv:4.10.0 AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y \
    curl \
    unzip \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Download models before building
RUN chmod +x download_models.sh && ./download_models.sh

# Build the binary with CGO enabled (required for OpenCV)
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o main .

# Stage 2: Runtime with OpenCV
FROM gocv/opencv:4.10.0-runtime

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash -m appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appgroup /app/main .

# Copy models from builder
COPY --from=builder --chown=appuser:appgroup /app/models ./models

# Copy environment files if they exist
COPY --chown=appuser:appgroup .env* ./

# Ensure binary is executable
RUN chmod +x main

# Switch to non-root user
USER appuser

# Expose port (adjust as needed)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./main"]