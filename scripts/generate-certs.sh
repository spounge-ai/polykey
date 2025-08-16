#!/bin/bash
set -e

# This script sets up a simple Public Key Infrastructure (PKI) for mTLS.
# It generates a root CA, a server certificate, and a client certificate.

CLIENT_ID=${1:-polykey-dev-client} # Default client ID if not provided

echo "Setting up PKI in ./certs directory..."
mkdir -p certs

# --- 1. Generate Root CA --- 
echo "Generating Root CA..."
openssl genpkey -algorithm RSA -out certs/ca-key.pem -pkeyopt rsa_keygen_bits:4096
openssl req -x509 -new -nodes -key certs/ca-key.pem -sha256 -days 1024 -out certs/ca.pem -subj "/CN=PolykeyTestCA"

# --- 2. Generate Server Certificate --- 
echo "Generating Server Certificate for localhost..."
openssl genpkey -algorithm RSA -out certs/server-key.pem -pkeyopt rsa_keygen_bits:4096
openssl req -new -key certs/server-key.pem -out certs/server-csr.pem -subj "/CN=localhost"
openssl x509 -req -in certs/server-csr.pem -CA certs/ca.pem -CAkey certs/ca-key.pem -CAcreateserial -out certs/server-cert.pem -days 365 -sha256 -extfile <(printf "subjectAltName=DNS:localhost")

# --- 3. Generate Client Certificate --- 
echo "Generating Client Certificate for client ID: $CLIENT_ID..."
openssl genpkey -algorithm RSA -out "certs/client-key.pem" -pkeyopt rsa_keygen_bits:4096
openssl req -new -key "certs/client-key.pem" -out "certs/client-csr.pem" -subj "/CN=${CLIENT_ID}"
openssl x509 -req -in "certs/client-csr.pem" -CA certs/ca.pem -CAkey certs/ca-key.pem -CAcreateserial -out "certs/client-cert.pem" -days 365 -sha256

# --- 4. Clean up CSRs and CA private key --- 
rm certs/server-csr.pem certs/client-csr.pem certs/ca-key.pem

echo "
✅ PKI setup complete.
Root CA: certs/ca.pem
Server Cert: certs/server-cert.pem
Server Key: certs/server-key.pem
Client Cert: certs/client-cert.pem (CN=${CLIENT_ID})
Client Key: certs/client-key.pem
"