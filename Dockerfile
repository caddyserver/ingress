FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY ./bin/ingress-controller .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 80 443
ENTRYPOINT ["/ingress-controller"]