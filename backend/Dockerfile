# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates ffmpeg

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Create storage directories
RUN mkdir -p /app/storage/media/images/originals \
    /app/storage/media/images/thumbnails \
    /app/storage/media/videos/originals \
    /app/storage/media/videos/previews

# Expose port
EXPOSE 8080

# Set environment variables (can be overridden)
ENV DATABASE_URL="host=postgres user=postgres password=postgres dbname=pim_prod port=5432 sslmode=disable"
ENV STORAGE_PATH="/app/storage/media"
ENV BASE_URL="http://localhost:8080/media"

# Run the server
CMD ["./server"]

