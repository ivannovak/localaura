#!/bin/bash

# Setup mkcert for local certificate generation
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERTS_DIR="$SCRIPT_DIR/certs"
PLATFORM=$(uname -s)

# Function to check if running as root or sudo available (Linux only)
check_sudo_linux() {
    if [ "$PLATFORM" != "Linux" ]; then
        return 0
    fi

    if [ "$EUID" -eq 0 ]; then
        return 0
    fi

    if ! command -v sudo &> /dev/null; then
        echo "Error: This script requires sudo on Linux, but sudo is not installed"
        exit 1
    fi

    # Test sudo access
    if ! sudo -n true 2>/dev/null; then
        echo "This script requires sudo access for:"
        echo "  - Installing mkcert binary"
        echo ""
        echo "You may be prompted for your password."
        echo ""

        # Prompt for sudo
        sudo -v || {
            echo "Error: sudo authentication failed"
            exit 1
        }
    fi
}

# Check sudo access upfront (Linux only)
check_sudo_linux

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

        # mkcert version configuration
        # Update this when new versions are released
        # Check: https://github.com/FiloSottile/mkcert/releases
        # Last updated: 2025-01-21
        # Reason: v1.4.4 is latest stable as of this date
        MKCERT_VERSION="v1.4.4"

        ARCH=$(uname -m)
        if [ "$ARCH" = "x86_64" ]; then
            MKCERT_ARCH="amd64"
            EXPECTED_SHA256="6d31c65b03972c6dc4a14ab429f2928300518b26503f58723e532d1b0a3bbb52"
        elif [ "$ARCH" = "aarch64" ]; then
            MKCERT_ARCH="arm64"
            EXPECTED_SHA256="b98f2cc69fd9147fe4d405d859c57504571adec0d3611c3eefd04107c7ac00d0"
        else
            echo "Unsupported architecture: $ARCH"
            exit 1
        fi

        MKCERT_URL="https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-${MKCERT_ARCH}"

        echo "Downloading from: $MKCERT_URL"
        curl -L "$MKCERT_URL" -o /tmp/mkcert

        echo "Verifying checksum..."
        ACTUAL_SHA256=$(sha256sum /tmp/mkcert | cut -d' ' -f1)

        if [ "$EXPECTED_SHA256" != "$ACTUAL_SHA256" ]; then
            echo "ERROR: Checksum verification failed!"
            echo "Expected: $EXPECTED_SHA256"
            echo "Got:      $ACTUAL_SHA256"
            echo ""
            echo "This could indicate:"
            echo "  1. The download was corrupted"
            echo "  2. A man-in-the-middle attack"
            echo "  3. The checksum in this script is outdated"
            echo ""
            echo "Aborting installation for security."
            rm -f /tmp/mkcert
            exit 1
        fi

        echo "✓ Checksum verified successfully"

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
echo "  ./add-cert.sh <domain-name>"
echo ""
echo "Example:"
echo "  ./add-cert.sh app.aura"
echo "  ./add-cert.sh api.aura"