#!/bin/bash
set -e

echo "Setting up PKI in ./certs directory..."
mkdir -p certs

if [ -f "certs/ca.pem" ] && [ -f "certs/ca-key.pem" ]; then
  echo "CA already exists. Skipping CA generation."
  exit 0
fi

# --- 1. Generate Root CA ---
echo "Generating Root CA..."
openssl genpkey -algorithm RSA -out certs/ca-key.pem -pkeyopt rsa_keygen_bits:4096
openssl req -x509 -new -nodes -key certs/ca-key.pem -sha256 -days 1024 -out certs/ca.pem -subj "/CN=PolykeyTestCA"

echo "
✅ CA setup complete.
Root CA: certs/ca.pem
Root CA Key: certs/ca-key.pem
"