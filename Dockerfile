# syntax=docker/dockerfile:1

# Stage 1: Build the static Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy Go module files only | so docker cache can be used effectively
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Create the config directory
RUN mkdir -p /etc/rr-dns/zones/

# Build statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o rrdnsd ./cmd/rr-dnsd

# Stage 2: Create minimal distroless image
FROM gcr.io/distroless/static:nonroot

# Copy the binary from builder
COPY --from=builder /app/rrdnsd /rrdnsd

# Copy the configuration directory
COPY --from=builder /etc/rr-dns /etc/rr-dns

# Run as non-root user
USER nonroot:nonroot

# Entrypoint
ENTRYPOINT ["/rrdnsd"]