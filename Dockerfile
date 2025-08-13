# syntax=docker/dockerfile:1.7

# ---------- Builder ----------
FROM cgr.dev/chainguard/go:latest AS builder

WORKDIR /app

# Leverage build cache for deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source
COPY . .

# Prepare default zones path (mirrors runtime path)
RUN mkdir -p /zones

# Build statically linked binary (multi-arch ready)
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /app/rr-dnsd ./cmd/rr-dnsd

# ---------- Runtime ----------
FROM cgr.dev/chainguard/static:latest

# OCI labels
LABEL org.opencontainers.image.title="rr-dns" \
      org.opencontainers.image.description="Small DNS server with upstream and zone support" \
      org.opencontainers.image.source="https://github.com/haukened/rr-dns" \
      org.opencontainers.image.licenses="MIT"

# Default configuration
ENV DNS_PORT=8053 \
    DNS_ENV=prod \
    DNS_LOG_LEVEL=info \
    DNS_ZONE_DIR=/zones

# Expose UDP port
EXPOSE 8053/udp

# Copy artifacts with correct ownership
COPY --from=builder --chown=nonroot:nonroot /app/rr-dnsd /usr/local/bin/rr-dnsd
COPY --from=builder --chown=nonroot:nonroot /zones /zones

# Allow mounting zones at runtime
VOLUME ["/zones"]

# Non-root runtime
USER nonroot:nonroot

# Entrypoint
ENTRYPOINT ["/usr/local/bin/rr-dnsd"]