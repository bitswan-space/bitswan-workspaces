FROM alpine:3.19
COPY bitswan-gitops /usr/bin/bitswan-gitops
ENTRYPOINT ["/usr/bin/bitswan-gitops"]