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
    && apt-get install -y --no-install-recommends sqlite3 ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/rtforum ./rtforum
COPY --from=frontend-builder /src/webapp/dist ./webapp/dist
COPY database ./database
COPY docker-entrypoint.sh ./docker-entrypoint.sh
RUN chmod +x ./docker-entrypoint.sh

EXPOSE 8443
ENTRYPOINT ["./docker-entrypoint.sh"]
CMD ["./rtforum"]
