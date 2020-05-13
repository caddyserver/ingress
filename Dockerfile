FROM golang:1.14.2-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY ./internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ingress-controller ./cmd/caddy

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM alpine:latest
COPY --from=builder /app/bin/ingress-controller .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 80 443
ENTRYPOINT ["/ingress-controller"]