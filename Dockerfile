FROM golang:alpine AS build
WORKDIR /app
ADD . /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ingress-controller ./cmd/caddy

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=build /app/bin/ingress-controller .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 80 443
ENTRYPOINT ["/ingress-controller"]
