# ABMCY Core Payment — image du service (mêmes conventions que le backend
# DIARRA : build statique CGO_ENABLED=0, image alpine minimale, migrations
# appliquées au démarrage via l'entrypoint).
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1001 app
WORKDIR /app
COPY --from=builder /out/server ./server
COPY --from=builder /out/migrate ./migrate
COPY migrations ./migrations
COPY docker-entrypoint.sh ./
RUN chmod +x ./docker-entrypoint.sh
USER app
EXPOSE 9090
ENTRYPOINT ["./docker-entrypoint.sh"]
