FROM golang:1.16.7-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY ./internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ingress-controller ./cmd/caddy

FROM alpine:latest AS certs
RUN apk --update add ca-certificates

FROM alpine:latest AS coraza-waf
ARG CORAZA_CONF_VERSION="v2.0.0"
ARG OWASP_MODSECURITY_CRS_VERSION="v3.3.2"

RUN apk --update add git

RUN mkdir -p /etc/caddy-ingress-controller/
# Use recommanded config for coraza
RUN wget https://raw.githubusercontent.com/corazawaf/coraza/$CORAZA_CONF_VERSION/coraza.conf-recommended -P /etc/caddy-ingress-controller/

# Pack OWASP CRS into image
RUN git clone -b $OWASP_MODSECURITY_CRS_VERSION https://github.com/coreruleset/coreruleset /etc/caddy-ingress-controller/coreruleset
WORKDIR /etc/caddy-ingress-controller/coreruleset
RUN mv crs-setup.conf.example crs-setup.conf
RUN mv rules/REQUEST-900-EXCLUSION-RULES-BEFORE-CRS.conf.example rules/REQUEST-900-EXCLUSION-RULES-BEFORE-CRS.conf
RUN mv rules/RESPONSE-999-EXCLUSION-RULES-AFTER-CRS.conf.example rules/RESPONSE-999-EXCLUSION-RULES-AFTER-CRS.conf


FROM alpine:latest
COPY --from=builder /app/bin/ingress-controller .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=coraza-waf /etc/caddy-ingress-controller/ /etc/caddy-ingress-controller/
EXPOSE 80 443
ENTRYPOINT ["/ingress-controller"]
