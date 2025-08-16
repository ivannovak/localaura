#!/bin/bash

# Setup mkcert for local certificate generation
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERTS_DIR="$SCRIPT_DIR/certs"
PLATFORM=$(uname -s)

echo "Setting up mkcert for local certificate generation..."

# Check if mkcert is installed
if ! command -v mkcert &> /dev/null; then
    echo "mkcert is not installed. Installing..."
    
    if [ "$PLATFORM" = "Darwin" ]; then
        # macOS - install via Homebrew
        if command -v brew &> /dev/null; then
            brew install mkcert
        else
            echo "Error: Homebrew is not installed. Please install Homebrew first."
            echo "Visit: https://brew.sh"
            exit 1
        fi
    elif [ "$PLATFORM" = "Linux" ]; then
        # Linux - download binary
        echo "Downloading mkcert for Linux..."
        MKCERT_VERSION="v1.4.4"
        curl -L "https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-amd64" -o /tmp/mkcert
        chmod +x /tmp/mkcert
        sudo mv /tmp/mkcert /usr/local/bin/mkcert
    else
        echo "Unsupported platform: $PLATFORM"
        echo "Please install mkcert manually: https://github.com/FiloSottile/mkcert"
        exit 1
    fi
    
    echo "✓ mkcert installed successfully"
else
    echo "✓ mkcert is already installed"
fi

# Install the local CA
echo "Installing mkcert local CA..."
mkcert -install
echo "✓ Local CA installed"

# Create certs directory structure
mkdir -p "$CERTS_DIR"
mkdir -p "$CERTS_DIR/domains"
echo "✓ Created certificates directory: $CERTS_DIR"

# Get CA certificate location
CA_CERT=$(mkcert -CAROOT)/rootCA.pem
if [ -f "$CA_CERT" ]; then
    cp "$CA_CERT" "$CERTS_DIR/ca.pem"
    echo "✓ CA certificate copied to $CERTS_DIR/ca.pem"
fi

echo ""
echo "✓ mkcert setup complete!"
echo "  Certificates location: $CERTS_DIR"
echo "  - ca.pem (CA certificate)"
echo "  - domains/ (individual domain certificates)"
echo ""
echo "To generate a certificate for a new .aura domain, use:"
echo "  ./add-site.sh <domain-name>"
echo ""
echo "Example:"
echo "  ./add-site.sh app.aura"
echo "  ./add-site.sh api.aura"