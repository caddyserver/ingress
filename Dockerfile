FROM alpine:latest AS certs
RUN apk --update add ca-certificates

FROM alpine:latest

EXPOSE 80 443
ENTRYPOINT ["/ingress-controller"]

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ingress-controller /
