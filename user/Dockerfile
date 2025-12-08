# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go.work and all module files first
COPY go.work .
COPY document/go.mod document/go.sum ./document/
COPY user/go.mod user/go.sum ./user/
COPY consent/go.mod consent/go.sum ./consent/
COPY gateway/go.mod gateway/go.sum ./gateway/
COPY shared/go.mod shared/go.sum ./shared/

# Download dependencies for user service
WORKDIR /build/user
RUN go mod download

# Copy shared proto generated files
WORKDIR /build
COPY shared/ ./shared/

# Copy user service source code
COPY user/ ./user/

# Build the application
WORKDIR /build/user
RUN CGO_ENABLED=0 GOOS=linux go build -o user cmd/server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/user/user .

# Expose gRPC port
EXPOSE 50052

# Run the application
CMD ["./user"]
