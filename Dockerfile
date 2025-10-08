# Stage 1: Builder with GoCV 0.42.0 (OpenCV 4.12.0)
FROM ghcr.io/emekarr/gateman-face-base-image/gateman-face-base-image:sha-b38d8e3 AS builder

WORKDIR /app

# Install auxiliary build tools and Go toolchain
ARG GO_VERSION=1.22.9
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    curl \
    ca-certificates \
    unzip \
    && rm -rf /var/lib/apt/lists/* \
    && ARCH=$(dpkg --print-architecture) \
    && case "$ARCH" in \
        amd64) GO_ARCH=amd64 ;; \
        arm64) GO_ARCH=arm64 ;; \
        *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;; \
    esac \
    && curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz -o /tmp/go.tgz \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm /tmp/go.tgz \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go \
    && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

ENV PATH="/usr/local/go/bin:${PATH}"

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Download models before building
RUN chmod +x download_models.sh && ./download_models.sh

# Build the binary with CGO enabled (required for OpenCV)
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Stage 2: Runtime with OpenCV 4.12.0
FROM ghcr.io/hybridgroup/opencv:4.12.0

# Install runtime utilities
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

# Ensure binary is executable
RUN chmod +x main

# Switch to non-root user
USER appuser

# Expose port (adjust as needed)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/ping || exit 1

CMD ["./main"]