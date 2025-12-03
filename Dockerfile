# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for module downloads
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for caching
COPY go.mod go.sum* ./
RUN go mod download && go mod tidy

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /logistic .

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS calls
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /logistic /app/logistic

# Create non-root user
RUN adduser -D -u 65532 appuser
USER appuser

# Expose port
EXPOSE 80

ENTRYPOINT ["/app/logistic"]
CMD ["serve"]
