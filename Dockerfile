# Stage 1: Builder with Go and OpenCV
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# Install OpenCV and build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    wget \
    unzip \
    curl \
    ca-certificates \
    pkg-config \
    libopencv-dev \
    libopencv-contrib-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Download models before building
RUN chmod +x download_models.sh && ./download_models.sh

# Build the binary with CGO enabled (required for OpenCV)
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Stage 2: Runtime with OpenCV
FROM debian:bookworm-slim

# Install runtime dependencies (OpenCV libraries with contrib)
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    libopencv-core4.6 \
    libopencv-imgproc4.6 \
    libopencv-imgcodecs4.6 \
    libopencv-objdetect4.6 \
    libopencv-dnn4.6 \
    libopencv-videoio4.6 \
    libopencv-contrib4.6 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash -m appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appgroup /app/main .

# Copy models from builder
COPY --from=builder --chown=appuser:appgroup /app/models ./models

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