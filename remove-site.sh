#!/bin/bash

# Remove a site and its certificate
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERTS_DIR="$SCRIPT_DIR/certs/domains"
SITES_DIR="$SCRIPT_DIR/sites"

# Check arguments
if [ $# -lt 1 ]; then
    echo "Usage: $0 <domain>"
    echo ""
    echo "Example:"
    echo "  $0 app.aura"
    exit 1
fi

DOMAIN=$1

# Validate domain ends with .aura
if [[ ! "$DOMAIN" =~ \.aura$ ]]; then
    echo "Error: Domain must end with .aura"
    exit 1
fi

# Domain name without .aura for file naming
DOMAIN_NAME="${DOMAIN%.aura}"
CERT_DIR="$CERTS_DIR/$DOMAIN_NAME"
SITE_CONFIG="$SITES_DIR/${DOMAIN_NAME}.caddy"

echo "Removing site: $DOMAIN"
echo ""

# Remove certificate
if [ -d "$CERT_DIR" ]; then
    echo "Removing certificate directory: $CERT_DIR"
    rm -rf "$CERT_DIR"
    echo "✓ Certificate removed"
else
    echo "⚠ Certificate directory not found: $CERT_DIR"
fi

# Remove site configuration
if [ -f "$SITE_CONFIG" ]; then
    echo "Removing site configuration: $SITE_CONFIG"
    rm -f "$SITE_CONFIG"
    echo "✓ Site configuration removed"
else
    echo "⚠ Site configuration not found: $SITE_CONFIG"
fi

# Remove from hosts file
echo ""
echo "Removing from /etc/hosts..."
if grep -q "$DOMAIN" /etc/hosts; then
    sudo sed -i.bak "/$DOMAIN/d" /etc/hosts
    echo "✓ Removed $DOMAIN from /etc/hosts"
else
    echo "⚠ $DOMAIN not found in /etc/hosts"
fi

# Check if Caddy is running and reload
if docker ps | grep -q "aura-caddy"; then
    echo ""
    echo "Reloading Caddy configuration..."
    docker exec aura-caddy caddy reload --config /etc/caddy/Caddyfile
    echo "✓ Caddy configuration reloaded"
fi

echo ""
echo "✓ Site $DOMAIN has been removed"