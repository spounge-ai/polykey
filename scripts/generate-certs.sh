#!/bin/bash
set -e

echo "Generating self-signed certificates for localhost..."

# Create certs directory if it doesn't exist
mkdir -p certs

# Generate self-signed certificate and private key
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout certs/key.pem \
  -out certs/cert.pem \
  -days 365 \
  -subj "/CN=localhost" \
  -addext "subjectAltName = DNS:localhost"

echo "✅ Certificates generated successfully in certs/"
