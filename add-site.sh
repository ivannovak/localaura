#!/bin/bash

# Add a new site with SSL certificate
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERTS_DIR="$SCRIPT_DIR/certs/domains"
SITES_DIR="$SCRIPT_DIR/sites"

# Check arguments
if [ $# -lt 1 ]; then
    echo "Usage: $0 <domain> [target-url]"
    echo ""
    echo "Examples:"
    echo "  $0 app.aura                    # Will prompt for target URL"
    echo "  $0 app.aura http://localhost:3000"
    echo "  $0 api.aura http://localhost:8080"
    echo "  $0 wiki.aura http://wiki-container:80"
    exit 1
fi

DOMAIN=$1
TARGET_URL=$2

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
mkdir -p "$SITES_DIR"

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
    else
        echo "Regenerating certificate for $DOMAIN..."
        rm -rf "$CERT_DIR"
    fi
fi

# Generate certificate if it doesn't exist
if [ ! -d "$CERT_DIR" ]; then
    echo "Generating SSL certificate for $DOMAIN..."
    mkdir -p "$CERT_DIR"
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
fi

# Get target URL if not provided
if [ -z "$TARGET_URL" ]; then
    echo ""
    echo "Enter the target URL for $DOMAIN"
    echo "Examples:"
    echo "  - http://localhost:3000"
    echo "  - http://container-name:80"
    echo "  - http://host.docker.internal:8080"
    read -p "Target URL: " TARGET_URL
    
    if [ -z "$TARGET_URL" ]; then
        echo "Error: Target URL is required"
        exit 1
    fi
fi

# Create Caddy site configuration
SITE_CONFIG="$SITES_DIR/${DOMAIN_NAME}.caddy"
echo "Creating Caddy configuration for $DOMAIN..."

cat > "$SITE_CONFIG" << EOF
# Configuration for $DOMAIN
# Target: $TARGET_URL
# Generated: $(date)

https://$DOMAIN {
    tls /certs/domains/$DOMAIN_NAME/cert.pem /certs/domains/$DOMAIN_NAME/key.pem
    
    # Reverse proxy to target
    reverse_proxy $TARGET_URL {
        # Add common headers
        header_up Host {host}
        header_up X-Real-IP {remote}
        header_up X-Forwarded-For {remote}
        header_up X-Forwarded-Proto {scheme}
        
        # Health check (optional)
        # health_uri /health
        # health_interval 30s
    }
    
    # Enable compression
    encode gzip
    
    # Logging
    log {
        output stdout
        format console
        level INFO
    }
}

# Redirect HTTP to HTTPS
http://$DOMAIN {
    redir https://{host}{uri} permanent
}
EOF

echo "✓ Site configuration created: $SITE_CONFIG"

# Update hosts file
echo ""
echo "Updating /etc/hosts..."
if grep -q "$DOMAIN" /etc/hosts; then
    echo "✓ $DOMAIN already in /etc/hosts"
else
    echo "127.0.0.2    $DOMAIN" | sudo tee -a /etc/hosts > /dev/null
    echo "✓ Added $DOMAIN to /etc/hosts"
fi

# Check if Caddy is running and reload
if docker ps | grep -q "aura-caddy"; then
    echo ""
    echo "Reloading Caddy configuration..."
    docker exec aura-caddy caddy reload --config /etc/caddy/Caddyfile
    echo "✓ Caddy configuration reloaded"
fi

echo ""
echo "✓ Site $DOMAIN successfully configured!"
echo ""
echo "Details:"
echo "  Domain:      https://$DOMAIN"
echo "  Target:      $TARGET_URL"
echo "  Certificate: $CERT_DIR"
echo "  Config:      $SITE_CONFIG"
echo ""
echo "The site will be accessible at https://$DOMAIN once Caddy is running."
echo "To start Caddy: docker-compose up -d"