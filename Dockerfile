FROM golang:1.13.8-alpine AS builder
WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ingress-controller ./cmd/caddy

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=builder /app/bin/ingress-controller .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 80 443
ENTRYPOINT ["/ingress-controller"]
