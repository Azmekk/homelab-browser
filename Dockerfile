# syntax=docker/dockerfile:1

# --- Builder: cross-compile the Go binary ---
# Pin the builder to the build platform (always the runner's native arch —
# amd64 in CI). This skips QEMU for the expensive compile step and cuts
# multi-arch build times from ~14min to ~2min.
FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS builder

WORKDIR /build

# Buildx injects these for each requested --platform.
ARG TARGETOS
ARG TARGETARCH

# Deps first (better layer cache).
COPY src/go.mod src/go.sum ./
RUN go mod download

# Sources.
COPY src/ ./

# Static pure-Go binary — modernc.org/sqlite needs no CGO, so the same
# source cross-compiles to any GOOS/GOARCH on a native amd64 runner.
ENV CGO_ENABLED=0
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/homelab-browser .

# --- Runtime: minimal image ---
FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --home /app --shell /usr/sbin/nologin app

WORKDIR /app
COPY --from=builder /out/homelab-browser /app/homelab-browser

ENV DATA_DIR=/data

RUN mkdir -p /data && chown -R app:app /app /data
USER app

VOLUME ["/data"]
EXPOSE 8080

CMD ["/app/homelab-browser"]
