#!/bin/bash

# Setup macOS/Linux DNS resolver for .aura TLD
set -e

# Function to check if running as root or sudo available
check_sudo() {
    if [ "$EUID" -eq 0 ]; then
        return 0
    fi

    if ! command -v sudo &> /dev/null; then
        echo "Error: This script requires sudo, but sudo is not installed"
        exit 1
    fi

    # Test sudo access
    if ! sudo -n true 2>/dev/null; then
        echo "This script requires sudo access for:"
        echo "  - Configuring DNS resolver"
        echo "  - Creating system resolver configuration"
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

# Check sudo access upfront
check_sudo

echo "Configuring DNS resolver for .aura domains..."
echo ""

OS_TYPE=$(uname -s)

if [ "$OS_TYPE" = "Darwin" ]; then
    # macOS: Use /etc/resolver
    echo "Detected macOS - using /etc/resolver"

    # Create /etc/resolver directory if it doesn't exist
    if [ ! -d "/etc/resolver" ]; then
        echo "Creating /etc/resolver directory..."
        sudo mkdir -p /etc/resolver
    fi

    # Create resolver configuration for .aura
    echo "Configuring .aura domain resolver..."
    echo "nameserver 127.0.0.2" | sudo tee /etc/resolver/aura > /dev/null

    echo "✓ DNS resolver configured for .aura domains"
    echo "✓ All *.aura domains will resolve via CoreDNS on 127.0.0.2"

elif [ "$OS_TYPE" = "Linux" ]; then
    # Linux: Multiple options depending on the DNS resolver in use
    echo "Detected Linux"

    # Check if systemd-resolved is in use
    if systemctl is-active --quiet systemd-resolved; then
        echo "Using systemd-resolved configuration..."

        # Create resolved configuration for .aura domain
        sudo mkdir -p /etc/systemd/resolved.conf.d

        cat << EOF | sudo tee /etc/systemd/resolved.conf.d/aura.conf > /dev/null
[Resolve]
DNS=127.0.0.2
Domains=~aura
EOF

        sudo systemctl restart systemd-resolved
        echo "✓ systemd-resolved configured for .aura domains"

    else
        echo "⚠ Manual DNS configuration required"
        echo ""
        echo "Add the following to your network manager or /etc/resolv.conf:"
        echo "  nameserver 127.0.0.2"
        echo ""
        echo "For NetworkManager, create /etc/NetworkManager/dnsmasq.d/aura.conf:"
        echo "  server=/aura/127.0.0.2"
    fi

else
    echo "⚠ Unsupported operating system: $OS_TYPE"
    echo "Manual DNS configuration required"
    exit 1
fi

echo ""
echo "Note: DNS changes may take a few seconds to propagate"
echo "Once the proxy is running, test with: dig whoami.aura"
