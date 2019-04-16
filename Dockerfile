FROM scratch
COPY ./bin/ingress-controller .
ENTRYPOINT ["/ingress-controller"]