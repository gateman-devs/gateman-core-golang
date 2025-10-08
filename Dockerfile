# Multi-stage build for Gateman Backend with manual OpenCV installation
# Stage 1: Build OpenCV 4.12.0 from source with contrib modules
FROM ubuntu:24.04 AS opencv-builder

# Install OpenCV build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    pkg-config \
    libjpeg-dev \
    libpng-dev \
    libtiff-dev \
    libavcodec-dev \
    libavformat-dev \
    libswscale-dev \
    libv4l-dev \
    libxvidcore-dev \
    libx264-dev \
    libgtk-3-dev \
    libatlas-base-dev \
    gfortran \
    python3-dev \
    libtbb-dev \
    libprotobuf-dev \
    protobuf-compiler \
    libgoogle-glog-dev \
    libgflags-dev \
    libgphoto2-dev \
    libeigen3-dev \
    libhdf5-dev \
    doxygen \
    wget \
    unzip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opencv

# Download OpenCV and contrib
RUN wget -O opencv.zip https://github.com/opencv/opencv/archive/4.12.0.zip \
    && unzip opencv.zip \
    && wget -O opencv_contrib.zip https://github.com/opencv/opencv_contrib/archive/4.12.0.zip \
    && unzip opencv_contrib.zip

# Build OpenCV
RUN cd opencv-4.12.0 && mkdir build && cd build && \
    cmake -D CMAKE_BUILD_TYPE=RELEASE \
          -D CMAKE_INSTALL_PREFIX=/usr/local \
          -D OPENCV_EXTRA_MODULES_PATH=../../opencv_contrib-4.12.0/modules \
          -D WITH_TBB=ON \
          -D ENABLE_FAST_MATH=1 \
          -D CUDA_FAST_MATH=1 \
          -D WITH_CUBLAS=1 \
          -D WITH_OPENGL=ON \
          -D WITH_OPENCL=ON \
          -D WITH_IPP=ON \
          -D WITH_TBB=ON \
          -D WITH_EIGEN=ON \
          -D WITH_V4L=ON \
          -D WITH_LIBV4L=ON \
          -D BUILD_TESTS=OFF \
          -D BUILD_PERF_TESTS=OFF \
          -D BUILD_EXAMPLES=OFF \
          -D BUILD_opencv_java=OFF \
          -D BUILD_opencv_python2=OFF \
          -D BUILD_opencv_python3=ON \
          -D OPENCV_GENERATE_PKGCONFIG=ON \
          .. && \
    make -j$(nproc) && \
    make install && \
    ldconfig

# Stage 2: Builder with Go and OpenCV
FROM ubuntu:24.04 AS builder

# Copy OpenCV from previous stage
COPY --from=opencv-builder /usr/local /usr/local

# Install Go and other build tools
RUN apt-get update && apt-get install -y \
    golang-go \
    build-essential \
    pkg-config \
    curl \
    ca-certificates \
    unzip \
    git \
    libjpeg8 \
    libpng16-16 \
    libtiff6 \
    libavcodec60 \
    libavformat60 \
    libswscale7 \
    libv4l-0 \
    libgtk-3-0 \
    libtbbmalloc2 \
    libprotobuf32t64 \
    libgoogle-glog0v6 \
    libgflags2.2 \
    libgphoto2-6 \
    libhdf5-103-1t64 \
    libglib2.0-0 \
    libwebp7 \
    libwebpdemux2 \
    libgl1 \
    libtbb12 \
    && rm -rf /var/lib/apt/lists/*

# Set PKG_CONFIG_PATH for OpenCV
ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig:$PKG_CONFIG_PATH
ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Download models
RUN chmod +x download_models.sh && ./download_models.sh

# Build the binary with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Stage 3: Runtime
FROM ubuntu:24.04

# Copy OpenCV libraries
COPY --from=opencv-builder /usr/local /usr/local

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    libjpeg8 \
    libpng16-16 \
    libtiff6 \
    libavcodec60 \
    libavformat60 \
    libswscale7 \
    libv4l-0 \
    libgtk-3-0 \
    libtbbmalloc2 \
    libprotobuf32t64 \
    libgoogle-glog0v6 \
    libgflags2.2 \
    libgphoto2-6 \
    libhdf5-103-1t64 \
    libglib2.0-0 \
    libwebp7 \
    libwebpdemux2 \
    libgl1 \
    libtbb12 \
    && rm -rf /var/lib/apt/lists/*

# Set library path
ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash -m appuser

WORKDIR /app

# Copy binary and models
COPY --from=builder --chown=appuser:appgroup /app/main .
COPY --from=builder --chown=appuser:appgroup /app/models ./models

RUN chmod +x main

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/ping || exit 1

CMD ["./main"]