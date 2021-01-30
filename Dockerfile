FROM golang:1.15.7-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY ./internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ingress-controller ./cmd/caddy

FROM alpine:latest AS certs
RUN apk --update add ca-certificates

FROM alpine:3.13.1
COPY --from=builder /app/bin/ingress-controller .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN mkdir -p /etc/caddy/certs
RUN mkdir -p ~/.config/caddy/
EXPOSE 80 443
ENTRYPOINT ["/ingress-controller"]
