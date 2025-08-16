#!/bin/bash

# List all configured sites
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SITES_DIR="$SCRIPT_DIR/sites"
CERTS_DIR="$SCRIPT_DIR/certs/domains"

echo "========================================"
echo "        Aura Proxy Sites"
echo "========================================"
echo ""

if [ ! -d "$SITES_DIR" ] || [ -z "$(ls -A "$SITES_DIR" 2>/dev/null)" ]; then
    echo "No sites configured yet."
    echo ""
    echo "To add a site, run:"
    echo "  ./add-site.sh <domain> [target-url]"
    exit 0
fi

# Check if Caddy is running
CADDY_STATUS="🔴 Stopped"
if docker ps | grep -q "aura-caddy"; then
    CADDY_STATUS="🟢 Running"
fi

echo "Caddy Status: $CADDY_STATUS"
echo ""
echo "Configured Sites:"
echo "-----------------"

# List all site configurations
for config in "$SITES_DIR"/*.caddy; do
    if [ -f "$config" ]; then
        # Extract domain and target from config file
        DOMAIN=$(grep "^# Configuration for" "$config" | cut -d' ' -f4)
        TARGET=$(grep "^# Target:" "$config" | cut -d' ' -f3-)
        CONFIG_NAME=$(basename "$config")
        DOMAIN_NAME="${CONFIG_NAME%.caddy}"
        
        # Check certificate status
        CERT_STATUS="✓"
        if [ ! -d "$CERTS_DIR/$DOMAIN_NAME" ]; then
            CERT_STATUS="✗"
        fi
        
        # Check hosts file status
        HOSTS_STATUS="✓"
        if ! grep -q "$DOMAIN" /etc/hosts 2>/dev/null; then
            HOSTS_STATUS="✗"
        fi
        
        echo ""
        echo "  📍 $DOMAIN"
        echo "     Target:      $TARGET"
        echo "     Certificate: $CERT_STATUS"
        echo "     Hosts file:  $HOSTS_STATUS"
        echo "     Config:      $CONFIG_NAME"
    fi
done

echo ""
echo "========================================"
echo ""
echo "Commands:"
echo "  Add site:    ./add-site.sh <domain> [target-url]"
echo "  Remove site: ./remove-site.sh <domain>"
echo "  Start proxy: docker-compose up -d"
echo "  Stop proxy:  docker-compose down"
echo "  View logs:   docker-compose logs -f"