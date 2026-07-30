#!/bin/sh
set -eu

private_key=/keys/jwt-private.pem
public_key=/keys/jwt-public.pem

if [ ! -s "$private_key" ] || [ ! -s "$public_key" ]; then
  umask 077
  openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out /keys/jwt-private.pem.tmp
  openssl pkey -in /keys/jwt-private.pem.tmp -pubout -out /keys/jwt-public.pem.tmp
  mv /keys/jwt-private.pem.tmp "$private_key"
  mv /keys/jwt-public.pem.tmp "$public_key"
fi

# The distroless nonroot runtime uses UID/GID 65532.
chown 65532:65532 "$private_key"
chmod 0600 "$private_key"
chmod 0644 "$public_key"
