# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -a -installsuffix cgo -ldflags="-s -w" -o snap-link .

# Final stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates wget && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/snap-link .

# Run as non-root (least privilege)
USER appuser

EXPOSE 8081

# Liveness only: a DB blip must not restart the process. Readiness (/health)
# is the platform's routing probe.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8081}/livez || exit 1

CMD ["./snap-link"]
