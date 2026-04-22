# syntax=docker/dockerfile:1

# --- Builder: compile the Go binary ---
FROM golang:1.26.2-bookworm AS builder

WORKDIR /build

# Deps first (better layer cache).
COPY src/go.mod src/go.sum ./
RUN go mod download

# Sources.
COPY src/ ./

# Static pure-Go binary — modernc.org/sqlite needs no CGO.
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/homelab-browser .

# --- Runtime: minimal image ---
FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates gosu \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --home /app --shell /usr/sbin/nologin app

WORKDIR /app
COPY --from=builder /out/homelab-browser /app/homelab-browser
COPY --chmod=0755 docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

ENV DATA_DIR=/data

RUN mkdir -p /data && chown -R app:app /app /data

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/homelab-browser"]
