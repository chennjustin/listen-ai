# ListenAI Go statistics service (SQLite)
# Env: STAT_PORT (default 8002), SQLITE_PATH (default /data/listenai.db)
# Mount a volume on /data with your listenai.db, or let the service create an empty DB file there.
FROM golang:1.24-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /listenai-stat .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=build /listenai-stat /usr/local/bin/listenai-stat

ENV STAT_PORT=8002 \
    SQLITE_PATH=/data/listenai.db

RUN mkdir -p /data
WORKDIR /data

EXPOSE 8002

ENTRYPOINT ["/usr/local/bin/listenai-stat"]
