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
echo "  2. Install and configure mkcert"
echo "  3. Create necessary directories"
echo "  4. Make scripts executable"
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

# Step 2: Setup mkcert
echo "Step 2: Setting up mkcert..."
echo "----------------------------------------"
bash "$SCRIPT_DIR/setup-mkcert.sh"
echo ""

# Step 3: Create necessary directories
echo "Step 3: Creating directories..."
echo "----------------------------------------"
mkdir -p "$SCRIPT_DIR/certs/domains"
echo "✓ Created certs/domains directory"
echo ""

# Step 3.5: Generate default WhoAmI certificate
echo "Step 3.5: Generating WhoAmI certificate..."
echo "----------------------------------------"
if ! command -v mkcert &> /dev/null; then
    echo "⚠ mkcert not found, skipping WhoAmI certificate generation"
else
    mkdir -p "$SCRIPT_DIR/certs/domains/whoami"
    cd "$SCRIPT_DIR/certs/domains/whoami"
    mkcert "whoami.aura" "*.whoami.aura" localhost 127.0.0.1 127.0.0.2 ::1
    # Rename to standard names
    for file in *.pem; do
        if [[ "$file" == *-key.pem ]]; then
            mv "$file" "key.pem"
        else
            mv "$file" "cert.pem"
        fi
    done
    echo "✓ WhoAmI certificate generated"
    
    # Add to hosts file
    if ! grep -q "whoami.aura" /etc/hosts; then
        echo "127.0.0.2    whoami.aura" | sudo tee -a /etc/hosts > /dev/null
        echo "✓ Added whoami.aura to /etc/hosts"
    fi
fi
echo ""

# Step 4: Make scripts executable
echo "Step 4: Making scripts executable..."
echo "----------------------------------------"
chmod +x "$SCRIPT_DIR"/*.sh
echo "✓ All scripts are now executable"
echo ""

# Step 5: Check Docker
echo "Step 5: Checking Docker..."
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