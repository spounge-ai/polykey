#!/bin/bash
set -e

CLIENT_ID=${1:?Please provide a client ID} # Exit if client ID is not provided

if [ ! -f "certs/ca.pem" ] || [ ! -f "certs/ca-key.pem" ]; then
  echo "CA not found. Please run generate-ca.sh first."
  exit 1
fi

# --- 3. Generate Client Certificate ---
echo "Generating Client Certificate for client ID: $CLIENT_ID..."
openssl genpkey -algorithm RSA -out "certs/client-key.pem" -pkeyopt rsa_keygen_bits:4096
openssl req -new -key "certs/client-key.pem" -out "certs/client-csr.pem" -subj "/CN=${CLIENT_ID}"
openssl x509 -req -in "certs/client-csr.pem" -CA certs/ca.pem -CAkey certs/ca-key.pem -CAcreateserial -out "certs/client-cert.pem" -days 365 -sha256

rm certs/client-csr.pem

echo "
✅ Client certificate setup complete.
Client Cert: certs/client-cert.pem (CN=${CLIENT_ID})
Client Key: certs/client-key.pem
"