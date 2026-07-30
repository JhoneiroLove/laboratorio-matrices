FROM alpine:3.22

RUN apk add --no-cache openssl
COPY generate.sh /usr/local/bin/generate-jwt-keys
RUN chmod 0555 /usr/local/bin/generate-jwt-keys

ENTRYPOINT ["/usr/local/bin/generate-jwt-keys"]
