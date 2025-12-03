# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for module downloads
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /logistic .

# Runtime stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /logistic /app/logistic

# Run as non-root user
USER nonroot:nonroot

# Expose port
EXPOSE 80

ENTRYPOINT ["/app/logistic"]
CMD ["serve"]
