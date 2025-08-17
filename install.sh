#!/bin/bash

# Install Aura CLI globally

set -e

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="aura"

echo "🚀 Installing Aura CLI..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    echo "Please install Go from: https://golang.org/dl/"
    exit 1
fi

# Build the binary
echo "Building Aura CLI..."
go build -o "$BINARY_NAME" ./cmd/aura

# Install to /usr/local/bin
echo "Installing to $INSTALL_DIR (may require sudo)..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_NAME" "$INSTALL_DIR/"
else
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
fi

# Make executable
chmod +x "$INSTALL_DIR/$BINARY_NAME"

echo "✅ Aura CLI installed successfully!"
echo ""
echo "Available commands:"
echo "  aura install  - Set up Aura proxy system"
echo "  aura start    - Start the proxy"
echo "  aura stop     - Stop the proxy"
echo "  aura cert     - Generate certificate for a domain"
echo "  aura status   - Check proxy status"
echo "  aura logs     - View proxy logs"
echo "  aura --help   - Show all commands"
echo ""
echo "Get started with: aura install"