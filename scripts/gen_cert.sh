#!/bin/bash
# gen_cert.sh — generates a self-signed TLS certificate for local development
# Run this once before starting the server:  bash scripts/gen_cert.sh
# The generated cert.pem and key.pem are loaded by server.go at startup

set -e  # exit immediately on any error

echo "Generating self-signed TLS certificate..."

openssl req \
  -x509 \
  -newkey rsa:4096 \
  -keyout key.pem \
  -out cert.pem \
  -days 365 \
  -nodes \
  -subj "/CN=localhost"

echo ""
echo "Generated:"
echo "  cert.pem  — TLS certificate (365 days)"
echo "  key.pem   — private key"
echo ""
echo "Start the server:  go run main.go"
echo "Test with curl:    curl -k https://localhost:8443/ping"
echo ""
echo "Note: -k flag tells curl to skip certificate verification"
echo "      Self-signed certificates are fine for development"