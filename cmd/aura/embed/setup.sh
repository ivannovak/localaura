#!/bin/bash

# Complete setup script for Aura proxy
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "========================================"
echo "     Aura Proxy Setup"
echo "========================================"
echo ""
echo "This script will set up the complete Aura proxy system:"
echo "  1. Configure custom loopback address (127.0.0.2)"
echo "  2. Configure DNS resolver for .aura domains"
echo "  3. Install and configure mkcert"
echo "  4. Create necessary directories"
echo "  5. Make scripts executable"
echo ""
read -p "Continue with setup? (y/N): " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Setup cancelled"
    exit 0
fi

echo ""

# Step 1: Setup loopback address
echo "Step 1: Setting up custom loopback address..."
echo "----------------------------------------"
bash "$SCRIPT_DIR/setup-loopback.sh"
echo ""

# Step 2: Setup DNS resolver for .aura domains
echo "Step 2: Setting up DNS resolver for .aura domains..."
echo "----------------------------------------"
bash "$SCRIPT_DIR/setup-resolver.sh"
echo ""

# Step 3: Setup mkcert
echo "Step 3: Setting up mkcert..."
echo "----------------------------------------"
bash "$SCRIPT_DIR/setup-mkcert.sh"
echo ""

# Step 4: Create necessary directories
echo "Step 4: Creating directories..."
echo "----------------------------------------"
mkdir -p "$SCRIPT_DIR/certs/domains"
echo "✓ Created certs/domains directory"
echo ""

# Step 5: Generate default WhoAmI certificate
echo "Step 5: Generating WhoAmI certificate..."
echo "----------------------------------------"
if ! command -v mkcert &> /dev/null; then
    echo "⚠ mkcert not found, skipping WhoAmI certificate generation"
else
    mkdir -p "$SCRIPT_DIR/certs/domains/whoami"
    cd "$SCRIPT_DIR/certs/domains/whoami"

    echo "Generating certificate with mkcert..."
    mkcert "whoami.aura" "*.whoami.aura" localhost 127.0.0.1 127.0.0.2 ::1

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
    echo "✓ WhoAmI certificate generated"
fi
echo ""

# Step 6: Make scripts executable
echo "Step 6: Making scripts executable..."
echo "----------------------------------------"
chmod +x "$SCRIPT_DIR"/*.sh
echo "✓ All scripts are now executable"
echo ""

# Step 7: Check Docker
echo "Step 7: Checking Docker..."
echo "----------------------------------------"
if command -v docker &> /dev/null && command -v docker-compose &> /dev/null; then
    echo "✓ Docker is installed"
    echo "✓ Docker Compose is installed"
else
    echo "⚠ Docker or Docker Compose is not installed"
    echo "  Please install Docker Desktop from: https://www.docker.com/products/docker-desktop"
fi
echo ""

echo "========================================"
echo "     Setup Complete!"
echo "========================================"
echo ""
echo "Next steps:"
echo ""
echo "1. Start the proxy (includes WhoAmI service):"
echo "   docker-compose up -d"
echo ""
echo "2. Test the proxy:"
echo "   https://whoami.aura"
echo ""
echo "3. To add a new service:"
echo "   a) Generate certificate: ./add-cert.sh myapp.aura"
echo "   b) Add Docker labels to your service (see docker-compose.example.yml)"
echo "   c) Start your service - it's automatically detected!"
echo ""
echo "The proxy runs on 127.0.0.2 to avoid conflicts with"
echo "services already using 127.0.0.1"