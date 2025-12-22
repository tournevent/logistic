# Build stage
FROM golang:1.25.5-alpine3.22 AS builder

WORKDIR /app

# Install git for module downloads
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Tidy and build binary
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /logistic .

# Runtime stage
FROM alpine:3.23.2

WORKDIR /app

# Install ca-certificates for HTTPS calls
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /logistic /app/logistic

# Create non-root user
RUN adduser -D -u 1000 appuser
USER appuser

# Expose port
EXPOSE 80

ENTRYPOINT ["/app/logistic"]
CMD ["serve"]
