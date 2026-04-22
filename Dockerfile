# syntax=docker/dockerfile:1

# --- Builder: compile SCSS + Go binary ---
FROM golang:1.23-bookworm AS builder

WORKDIR /build

# Install dart-sass to compile SCSS at build time.
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN curl -L https://github.com/sass/dart-sass/releases/download/1.83.4/dart-sass-1.83.4-linux-x64.tar.gz \
        -o /tmp/dart-sass.tar.gz \
    && tar -xzf /tmp/dart-sass.tar.gz -C /opt \
    && ln -s /opt/dart-sass/sass /usr/local/bin/sass \
    && rm /tmp/dart-sass.tar.gz

# Deps first (better layer cache).
COPY src/go.mod src/go.sum ./
RUN go mod download

# Sources.
COPY src/ ./

# Compile SCSS to CSS so it ends up in the embedded filesystem.
RUN sass --no-source-map --style=compressed ./wwwroot/styles.scss ./wwwroot/styles.css

# Build a fully static pure-Go binary (modernc.org/sqlite → no CGO needed).
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/homelab-browser .

# --- Runtime: minimal image ---
FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --home /app --shell /usr/sbin/nologin app

WORKDIR /app
COPY --from=builder /out/homelab-browser /app/homelab-browser

ENV BIND_URL=:8080
ENV DATA_DIR=/data

RUN mkdir -p /data && chown -R app:app /app /data
USER app

VOLUME ["/data"]
EXPOSE 8080

CMD ["/app/homelab-browser"]
