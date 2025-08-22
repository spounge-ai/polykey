#!/bin/bash
set -e

CLIENT_ID="polykey-dev-client"
CLIENT_SECRET="supersecretdevpassword"

echo "--- Setting up Development Client: ${CLIENT_ID} ---"

# Step 1: Generate all necessary certificates
# The generate-certs script now creates a CA, and server/client certs signed by it.
# The client certificate CN is set to the CLIENT_ID.
echo "
[1/3] Generating PKI for mTLS..."
bash scripts/generate-ca.sh
bash scripts/generate-server.sh localhost
bash scripts/generate-client.sh "${CLIENT_ID}"

# Step 2: Generate the server-side configuration for the client
# This tells the server about the new client, its ID, and its hashed secret.
echo "
[2/3] Generating server-side client record..."
go run scripts/generate_client_config.go "${CLIENT_ID}" "${CLIENT_SECRET}" "Development Client"

# Step 3: Generate the client-side TLS configuration
# This tells the client how to connect to the mTLS-enabled server.
echo "
[3/3] Generating client-side TLS configuration..."
cat <<EOF > configs/dev_client/tls.yaml
cert_file: "certs/client-cert.pem"
key_file: "certs/client-key.pem"
ca_file: "certs/ca.pem"
EOF

echo "
✅ Development client setup complete.
   Client ID: ${CLIENT_ID}
   Client Secret: ${CLIENT_SECRET}
"