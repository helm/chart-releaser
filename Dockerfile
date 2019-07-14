FROM alpine:3.9

RUN apk update && \
    apk add ca-certificates

COPY cr /usr/local/bin/cr

# Ensure that the binary is available on path and is executable
RUN cr --help
