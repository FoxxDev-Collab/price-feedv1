# Build stage
FROM docker.io/library/golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o price-feed ./cmd/server/

# Runtime stage
FROM docker.io/library/alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 appuser

# Copy binary from builder
COPY --from=builder /app/price-feed .

# Copy static web files
COPY --from=builder /app/web ./web

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

CMD ["./price-feed"]
