# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod và go.sum từ workspace root (cần cho go.work)
COPY go.work go.work.sum* ./
COPY document/go.mod document/go.sum* ./document/
COPY shared/go.mod shared/go.sum* ./shared/
COPY user/go.mod user/go.sum* ./user/
COPY consent/go.mod consent/go.sum* ./consent/
COPY gateway/go.mod gateway/go.sum* ./gateway/

# Download dependencies
WORKDIR /app/document
RUN go mod download

# Copy source code
WORKDIR /app
COPY document/ ./document/
COPY shared/ ./shared/

# Build binary
WORKDIR /app/document
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/document ./cmd/server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary từ build stage
COPY --from=builder /app/bin/document .

# Copy .env nếu cần (hoặc dùng environment variables)
# COPY document/.env .env

EXPOSE 50051

CMD ["./document"]