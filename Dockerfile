FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM golang:1.13.5 as builder
WORKDIR /build
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN mkdir -p ./bin
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ./bin/ingress-controller ./cmd/caddy

FROM scratch
COPY --from=builder /build/bin/ingress-controller .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 80 443
ENTRYPOINT ["/ingress-controller"]
