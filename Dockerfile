FROM gocv/opencv:4.11.0 AS builder
WORKDIR /app
# Install TBB library
RUN apt-get update && apt-get install -y libtbb-dev && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Download models
RUN chmod +x download_models.sh && ./download_models.sh
# Set environment variables for proper linking
ENV CGO_LDFLAGS="-L/usr/lib/x86_64-linux-gnu -ltbb"
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .
FROM gocv/opencv:4.11.0
RUN apt-get update && apt-get install -y ca-certificates libtbb2 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/models ./models
CMD ["./main"]