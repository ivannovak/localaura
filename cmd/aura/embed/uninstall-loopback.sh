#!/bin/bash

# Remove custom loopback address for Aura proxy
set -e

LOOPBACK_IP="127.0.0.2"
PLATFORM=$(uname -s)

echo "Removing custom loopback address $LOOPBACK_IP..."

if [ "$PLATFORM" = "Darwin" ]; then
    # macOS
    echo "Detected macOS"

    # Remove loopback address
    if ifconfig lo0 | grep -q "$LOOPBACK_IP"; then
        echo "Removing loopback address $LOOPBACK_IP..."
        sudo ifconfig lo0 -alias $LOOPBACK_IP
        echo "✓ Loopback address removed"
    else
        echo "✓ Loopback address not configured"
    fi

    # Remove launch daemon
    PLIST_FILE="/Library/LaunchDaemons/com.aura.loopback.plist"
    if [ -f "$PLIST_FILE" ]; then
        echo "Removing launch daemon..."
        sudo launchctl unload "$PLIST_FILE" 2>/dev/null || true
        sudo rm -f "$PLIST_FILE"
        echo "✓ Launch daemon removed"
    else
        echo "✓ Launch daemon not found"
    fi

elif [ "$PLATFORM" = "Linux" ]; then
    # Linux
    echo "Detected Linux"

    # Remove loopback address
    if ip addr show lo | grep -q "$LOOPBACK_IP"; then
        echo "Removing loopback address $LOOPBACK_IP..."
        sudo ip addr del $LOOPBACK_IP/32 dev lo 2>/dev/null || true
        echo "✓ Loopback address removed"
    else
        echo "✓ Loopback address not configured"
    fi

    # Remove systemd service
    SERVICE_FILE="/etc/systemd/system/aura-loopback.service"
    if [ -f "$SERVICE_FILE" ]; then
        echo "Removing systemd service..."
        sudo systemctl stop aura-loopback.service 2>/dev/null || true
        sudo systemctl disable aura-loopback.service 2>/dev/null || true
        sudo rm -f "$SERVICE_FILE"
        sudo systemctl daemon-reload
        echo "✓ Systemd service removed"
    else
        echo "✓ Systemd service not found"
    fi
fi

echo "✓ Loopback address cleanup complete"
