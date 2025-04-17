# Simple, minimal Dockerfile for Go backend
FROM golang:1.23-alpine AS builder
WORKDIR /backend
COPY . .
RUN go build -o server ./cmd/server/main.go

FROM alpine:latest
WORKDIR /backend
COPY --from=builder /backend/server .
COPY --from=builder /backend/config ./config
COPY --from=builder /backend/migrations ./migrations
ENV GIN_MODE=release
EXPOSE 8080
CMD ["./server"]
