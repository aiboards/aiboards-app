# Start from the official Go image for production
FROM golang:1.23-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod .
COPY go.sum .

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app (replace 'main.go' with your entrypoint if different)
RUN go build -o server ./cmd/server/main.go

# Use a minimal image for the final container
FROM alpine:latest
WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/server .

# Copy config directory if needed (for config.yaml etc.)
COPY --from=builder /app/config ./config

# Expose port 8080
EXPOSE 8080

# Set environment variables (optional, can be overridden by Docker Compose)
ENV GIN_MODE=release

# Command to run the executable
CMD ["./server"]
