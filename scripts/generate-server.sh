#!/bin/bash
set -e

SERVER_CN=${1:-localhost} # Default server CN if not provided

if [ ! -f "certs/ca.pem" ] || [ ! -f "certs/ca-key.pem" ]; then
  echo "CA not found. Please run generate-ca.sh first."
  exit 1
fi

# --- 2. Generate Server Certificate ---
echo "Generating Server Certificate for CN: $SERVER_CN..."
openssl genpkey -algorithm RSA -out "certs/server-key.pem" -pkeyopt rsa_keygen_bits:4096
openssl req -new -key "certs/server-key.pem" -out "certs/server-csr.pem" -subj "/CN=${SERVER_CN}"
openssl x509 -req -in "certs/server-csr.pem" -CA certs/ca.pem -CAkey certs/ca-key.pem -CAcreateserial -out "certs/server-cert.pem" -days 365 -sha256 -extfile <(printf "subjectAltName=DNS:%s" "$SERVER_CN")

rm certs/server-csr.pem

echo "
✅ Server certificate setup complete.
Server Cert: certs/server-cert.pem
Server Key: certs/server-key.pem
"