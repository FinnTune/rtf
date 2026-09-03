FROM node:22-bookworm-slim AS frontend-builder

WORKDIR /src/webapp

COPY webapp/package.json webapp/package-lock.json ./
RUN npm ci

COPY webapp .
RUN npm run build

FROM golang:1.25-bookworm AS builder

WORKDIR /src
ENV CGO_ENABLED=1
ENV CGO_CFLAGS=-Wno-discarded-qualifiers

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/rtforum .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends sqlite3 ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    # Fixed uid/gid 1000 rather than a dynamically-assigned one: the app's
    # working data (database/, uploads/, logfiles/) is bind-mounted from the
    # host (see docker-compose.yml), and 1000 is the default first-user
    # uid/gid on Linux — matching it here means those host directories are
    # writable by this container user without any extra chown step, for the
    # common single-developer setup this compose file targets.
    && groupadd --gid 1000 rtforum \
    && useradd --uid 1000 --gid rtforum --no-create-home --shell /usr/sbin/nologin rtforum

WORKDIR /app
COPY --from=builder /out/rtforum ./rtforum
COPY --from=frontend-builder /src/webapp/dist ./webapp/dist
COPY database ./database
COPY docker-entrypoint.sh ./docker-entrypoint.sh
RUN chmod +x ./docker-entrypoint.sh

EXPOSE 8443
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -kfs https://localhost:8443/healthz || exit 1
USER rtforum
ENTRYPOINT ["./docker-entrypoint.sh"]
CMD ["./rtforum"]
