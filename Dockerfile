FROM alpine:3.24

RUN apk --no-cache add ca-certificates git

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/cr /usr/local/bin/cr

# Ensure that the binary is available on path and is executable
RUN cr --help

ENTRYPOINT [ "/usr/local/bin/cr" ]
