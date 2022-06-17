FROM alpine:3.16

RUN apk --no-cache add ca-certificates git

COPY cr /usr/local/bin/cr

# Ensure that the binary is available on path and is executable
RUN cr --help

ENTRYPOINT [ "/usr/local/bin/cr" ]
