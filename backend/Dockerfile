# syntax=docker/dockerfile:1
FROM golang:1.27-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=1.0.0
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -g 10001 app && adduser -D -u 10001 -G app app && \
    mkdir -p /data/scripts && chown -R app:app /data
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/seed /app/seed
USER app
EXPOSE 8080
HEALTHCHECK --interval=20s --timeout=3s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/server"]
