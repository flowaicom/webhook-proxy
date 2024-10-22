FROM golang:1.23-alpine AS build
WORKDIR /opt/app
ADD main.go store.go client_listener.go webhook.go go.mod go.sum prometheus.go token.go util.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o proxy .

FROM ghcr.io/linuxcontainers/alpine:3.20
LABEL org.opencontainers.image.source=https://github.com/flowaicom/webhook-proxy
WORKDIR /opt/app
EXPOSE 8000
COPY --from=build /opt/app/proxy /opt/app/proxy
ENTRYPOINT ["/opt/app/proxy"]
