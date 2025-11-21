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
mkdir -p "$CERTS_DIR" || {
    echo "Error: Failed to create certificate directory"
    exit 1
}

# Domain name without .aura for file naming
DOMAIN_NAME="${DOMAIN%.aura}"
CERT_DIR="$CERTS_DIR/$DOMAIN_NAME"

# Create certificate directory atomically
mkdir -p "$CERT_DIR" || {
    echo "Error: Failed to create certificate directory for $DOMAIN"
    exit 1
}

# Use a lock file to prevent concurrent generation
LOCK_FILE="$CERT_DIR/.lock"

# Acquire lock
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
    echo "Error: Certificate generation already in progress for $DOMAIN"
    echo "If you're sure no other process is running, remove: $LOCK_FILE"
    exit 1
fi

# Cleanup lock on exit
trap 'rm -f "$LOCK_FILE"' EXIT

# Check if certificate already exists after acquiring lock
if [ -f "$CERT_DIR/cert.pem" ] && [ -f "$CERT_DIR/key.pem" ]; then
    echo "Certificate for $DOMAIN already exists in $CERT_DIR"
    read -p "Do you want to regenerate it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Keeping existing certificate"
        exit 0
    else
        echo "Regenerating certificate for $DOMAIN..."
        rm -f "$CERT_DIR/cert.pem" "$CERT_DIR/key.pem"
    fi
fi

# Generate certificate
echo "Generating SSL certificate for $DOMAIN..."
cd "$CERT_DIR"

# Generate certificate with wildcard for subdomains
mkcert "$DOMAIN" "*.$DOMAIN" localhost 127.0.0.1 127.0.0.2 ::1

# Rename to standard names
for file in *.pem; do
    if [[ "$file" == *-key.pem ]]; then
        mv "$file" "key.pem"
    else
        mv "$file" "cert.pem"
    fi
done

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