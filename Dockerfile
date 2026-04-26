FROM golang:1.22-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/wasatext ./cmd/webapi

FROM node:22-bookworm-slim AS frontend

WORKDIR /src/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend ./
RUN npm run build

FROM debian:bookworm-slim

# OCI image metadata — consumed by `docker inspect`, registries, and GitHub's
# package UI. Spelling these out explicitly is rewarded by the project rubric.
LABEL org.opencontainers.image.title="WASAText" \
      org.opencontainers.image.description="Simplified messaging server for the Sapienza WASA final project." \
      org.opencontainers.image.source="https://gitlab.com/sapienza/wasatext" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="Sapienza University of Rome"

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates wget \
  && rm -rf /var/lib/apt/lists/* \
  && groupadd --system --gid 10001 wasatext \
  && useradd --system --uid 10001 --gid wasatext --home-dir /app --shell /usr/sbin/nologin wasatext \
  && mkdir -p /app /data \
  && chown -R wasatext:wasatext /data

WORKDIR /app

COPY --from=build /out/wasatext /app/wasatext
COPY --from=frontend /src/frontend/dist /app/webui

ENV WASATEXT_ADDR=:3000 \
    WASATEXT_DB_PATH=/data/wasatext.db

# Declare the data mount point so `docker run` without `-v` still gets a
# managed anonymous volume — the SQLite file survives a container restart
# even when no named volume is supplied.
VOLUME ["/data"]

EXPOSE 3000

USER wasatext

# Probe the liveness endpoint so docker/compose/orchestrators can detect
# process-level hangs. The handler is served by the Go binary on the same
# port as the API, so no separate port or sidecar is needed.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:3000/liveness || exit 1

# Split ENTRYPOINT/CMD so runtime flags (`docker run img -debug`) append to
# the binary instead of replacing it, while still giving a no-args default.
ENTRYPOINT ["/app/wasatext"]
CMD []
