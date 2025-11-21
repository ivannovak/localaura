#!/bin/bash

# Setup custom loopback address for Aura proxy
# This avoids conflicts with services already using 127.0.0.1

set -e

LOOPBACK_IP="127.0.0.2"
PLATFORM=$(uname -s)

echo "Setting up custom loopback address $LOOPBACK_IP for Aura proxy..."

if [ "$PLATFORM" = "Darwin" ]; then
    # macOS
    echo "Detected macOS"
    
    # Check if already configured
    if ifconfig lo0 | grep -q "$LOOPBACK_IP"; then
        echo "✓ Loopback address $LOOPBACK_IP already configured"
    else
        echo "Adding loopback address $LOOPBACK_IP..."
        sudo ifconfig lo0 alias $LOOPBACK_IP up
        echo "✓ Loopback address $LOOPBACK_IP added"
    fi
    
    # Make persistent across reboots
    PLIST_FILE="/Library/LaunchDaemons/com.aura.loopback.plist"
    if [ ! -f "$PLIST_FILE" ]; then
        echo "Creating launch daemon for persistent configuration..."
        sudo tee "$PLIST_FILE" > /dev/null << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.aura.loopback</string>
    <key>ProgramArguments</key>
    <array>
        <string>/sbin/ifconfig</string>
        <string>lo0</string>
        <string>alias</string>
        <string>$LOOPBACK_IP</string>
        <string>up</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
EOF
        sudo launchctl load "$PLIST_FILE"
        echo "✓ Launch daemon created and loaded"
    else
        echo "✓ Launch daemon already exists"
    fi
    
elif [ "$PLATFORM" = "Linux" ]; then
    # Linux
    echo "Detected Linux"
    
    # Check if already configured
    if ip addr show lo | grep -q "$LOOPBACK_IP"; then
        echo "✓ Loopback address $LOOPBACK_IP already configured"
    else
        echo "Adding loopback address $LOOPBACK_IP..."
        sudo ip addr add $LOOPBACK_IP/32 dev lo
        echo "✓ Loopback address $LOOPBACK_IP added"
    fi
    
    # Make persistent across reboots (systemd)
    SERVICE_FILE="/etc/systemd/system/aura-loopback.service"
    if [ ! -f "$SERVICE_FILE" ]; then
        echo "Creating systemd service for persistent configuration..."
        sudo tee "$SERVICE_FILE" > /dev/null << EOF
[Unit]
Description=Aura Custom Loopback Address
After=network.target

[Service]
Type=oneshot
ExecStart=/sbin/ip addr add $LOOPBACK_IP/32 dev lo
ExecStop=/sbin/ip addr del $LOOPBACK_IP/32 dev lo
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF
        sudo systemctl daemon-reload
        sudo systemctl enable aura-loopback.service
        sudo systemctl start aura-loopback.service
        echo "✓ Systemd service created and started"
    else
        echo "✓ Systemd service already exists"
    fi
else
    echo "Unsupported platform: $PLATFORM"
    exit 1
fi

echo ""
echo "✓ Custom loopback address setup complete!"
echo "  Aura proxy will be accessible at $LOOPBACK_IP"