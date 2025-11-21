#!/bin/bash

# Generate SSL certificate for a .aura domain
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERTS_DIR="$SCRIPT_DIR/certs/domains"

# Check arguments
if [ $# -lt 1 ]; then
    echo "Usage: $0 <domain>"
    echo ""
    echo "Examples:"
    echo "  $0 app.aura"
    echo "  $0 api.aura"
    echo "  $0 admin.dashboard.aura"
    exit 1
fi

DOMAIN=$1

# Validate domain ends with .aura
if [[ ! "$DOMAIN" =~ \.aura$ ]]; then
    echo "Error: Domain must end with .aura"
    exit 1
fi

# Check if mkcert is installed
if ! command -v mkcert &> /dev/null; then
    echo "Error: mkcert is not installed. Please run ./setup-mkcert.sh first"
    exit 1
fi

# Create directories if they don't exist
mkdir -p "$CERTS_DIR"

# Domain name without .aura for file naming
DOMAIN_NAME="${DOMAIN%.aura}"
CERT_DIR="$CERTS_DIR/$DOMAIN_NAME"

# Check if certificate already exists
if [ -d "$CERT_DIR" ]; then
    echo "Certificate for $DOMAIN already exists in $CERT_DIR"
    read -p "Do you want to regenerate it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Keeping existing certificate"
        exit 0
    else
        echo "Regenerating certificate for $DOMAIN..."
        rm -rf "$CERT_DIR"
    fi
fi

# Generate certificate
echo "Generating SSL certificate for $DOMAIN..."
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

# Generate certificate with wildcard for subdomains
echo "Generating certificate with mkcert..."
mkcert "$DOMAIN" "*.$DOMAIN" localhost 127.0.0.1 127.0.0.2 ::1

# Find generated files explicitly
CERT_FILE=""
KEY_FILE=""

for file in *.pem; do
    if [[ "$file" == *"-key.pem" ]]; then
        KEY_FILE="$file"
    else
        CERT_FILE="$file"
    fi
done

# Verify we found both files
if [[ -z "$CERT_FILE" ]]; then
    echo "Error: Certificate file not found after mkcert generation"
    ls -la
    exit 1
fi

if [[ -z "$KEY_FILE" ]]; then
    echo "Error: Private key file not found after mkcert generation"
    ls -la
    exit 1
fi

# Rename to standard names
echo "Renaming certificate files..."
mv "$CERT_FILE" "cert.pem"
mv "$KEY_FILE" "key.pem"

# Verify final files exist
if [[ ! -f "cert.pem" ]] || [[ ! -f "key.pem" ]]; then
    echo "Error: Failed to create cert.pem or key.pem"
    ls -la
    exit 1
fi

echo "✓ Certificate files created successfully"
chmod 600 cert.pem key.pem

echo "✓ Certificate generated in $CERT_DIR"
echo ""
echo "✓ Certificate ready for $DOMAIN!"
echo "  DNS resolution handled automatically via CoreDNS (all *.aura domains resolve to 127.0.0.2)"
echo ""
echo "Certificate paths for Docker labels:"
echo "  caddy.tls: \"/certs/domains/$DOMAIN_NAME/cert.pem /certs/domains/$DOMAIN_NAME/key.pem\""
echo ""
echo "Example Docker labels:"
echo "  labels:"
echo "    caddy: $DOMAIN"
echo "    caddy.reverse_proxy: \"{{upstreams 80}}\""
echo "    caddy.tls: \"/certs/domains/$DOMAIN_NAME/cert.pem /certs/domains/$DOMAIN_NAME/key.pem\""