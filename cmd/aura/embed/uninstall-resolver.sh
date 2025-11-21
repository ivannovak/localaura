#!/bin/bash

# Remove macOS/Linux DNS resolver for .aura TLD
set -e

echo "Removing DNS resolver for .aura domains..."
echo ""

OS_TYPE=$(uname -s)

if [ "$OS_TYPE" = "Darwin" ]; then
    # macOS: Remove /etc/resolver/aura
    if [ -f "/etc/resolver/aura" ]; then
        echo "Removing /etc/resolver/aura..."
        sudo rm -f /etc/resolver/aura
        echo "✓ DNS resolver configuration removed"
    else
        echo "✓ DNS resolver configuration not found (already removed)"
    fi

elif [ "$OS_TYPE" = "Linux" ]; then
    # Linux: Remove systemd-resolved configuration
    if [ -f "/etc/systemd/resolved.conf.d/aura.conf" ]; then
        echo "Removing systemd-resolved configuration..."
        sudo rm -f /etc/systemd/resolved.conf.d/aura.conf
        sudo systemctl restart systemd-resolved
        echo "✓ systemd-resolved configuration removed"
    else
        echo "✓ DNS resolver configuration not found (already removed)"
    fi
fi

echo "✓ DNS resolver cleanup complete"
