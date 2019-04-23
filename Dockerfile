FROM scratch
COPY ./bin/ingress-controller .
EXPOSE 80 443
ENTRYPOINT ["/ingress-controller"]