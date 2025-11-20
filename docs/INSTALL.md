# Installation Guide

Complete installation instructions for Aura proxy on different platforms.

## Prerequisites

### All Platforms
- Docker Desktop (or Docker daemon + Docker Compose)
- Sudo/root access
- Ports 80 and 443 available on 127.0.0.2
- Git (for cloning repository)

### macOS Specific
- **Homebrew** - Required for mkcert installation
- macOS 10.14 or later recommended

### Linux Specific
- **systemd-resolved** - Required for automatic DNS setup
  - Check if active: `systemctl is-active systemd-resolved`
  - Most modern Linux distributions (Ubuntu 18.04+, Fedora 33+, etc.)
- Alternative: Manual DNS configuration (see below)

---

## Installation Methods

### Option 1: Quick Install (Recommended)

```bash
# Clone the repository
git clone https://github.com/ivannovak/localaura.git
cd localaura

# Build and install the CLI
make install

# Set up the proxy system (requires sudo)
aura install

# Start the proxy
aura start

# Test it works
open https://whoami.aura  # macOS
# or
xdg-open https://whoami.aura  # Linux
```

**What happens during `aura install`:**
1. Creates `~/.aura` directory
2. Configures loopback address (127.0.0.2)
3. Sets up DNS resolver for `.aura` domains
4. Installs mkcert and creates local Certificate Authority
5. Generates certificate for whoami.aura test service
6. Copies Docker Compose configuration

---

### Option 2: Manual Build

For developers or if you want more control:

```bash
# Clone the repository
git clone https://github.com/ivannovak/localaura.git
cd localaura

# Build the CLI
go build -o aura ./cmd/aura

# Install to /usr/local/bin
sudo cp aura /usr/local/bin/
sudo chmod +x /usr/local/bin/aura

# Set up the proxy system
aura install
```

---

## Platform-Specific Setup

### macOS Setup

#### Prerequisites Check
```bash
# Verify Homebrew is installed
brew --version

# Verify Docker Desktop is running
docker ps
```

#### Installation
```bash
make install
aura install
```

**System Changes Made:**
- `/Library/LaunchDaemons/com.aura.loopback.plist` - Loopback persistence
- `/etc/resolver/aura` - DNS configuration for `.aura` domains
- Homebrew installs mkcert
- mkcert CA installed in macOS Keychain

#### Verify Installation
```bash
# Check loopback address
ifconfig lo0 | grep 127.0.0.2

# Check DNS resolver
cat /etc/resolver/aura

# Check CA is trusted
security find-certificate -c "mkcert" -a
```

---

### Linux Setup

#### Prerequisites Check
```bash
# Verify systemd-resolved is active
systemctl is-active systemd-resolved

# Verify Docker is running
docker ps

# Check sudo access
sudo echo "Sudo works"
```

#### Installation
```bash
make install
aura install
```

**System Changes Made:**
- `/etc/systemd/system/aura-loopback.service` - Loopback persistence
- `/etc/systemd/resolved.conf.d/aura.conf` - DNS configuration
- mkcert binary downloaded to `/usr/local/bin/`
- mkcert CA installed in system trust store

#### Verify Installation
```bash
# Check loopback address
ip addr show lo | grep 127.0.0.2

# Check systemd service
systemctl status aura-loopback

# Check DNS configuration
cat /etc/systemd/resolved.conf.d/aura.conf
resolvectl status | grep -A 5 aura
```

---

### Linux Without systemd-resolved

If you're using NetworkManager, dnsmasq, or another DNS resolver:

#### Manual DNS Configuration

**For NetworkManager with dnsmasq:**
```bash
# Create dnsmasq configuration
sudo mkdir -p /etc/NetworkManager/dnsmasq.d
echo "server=/aura/127.0.0.2" | sudo tee /etc/NetworkManager/dnsmasq.d/aura.conf

# Restart NetworkManager
sudo systemctl restart NetworkManager
```

**For standalone dnsmasq:**
```bash
# Add to /etc/dnsmasq.conf
echo "server=/aura/127.0.0.2" | sudo tee -a /etc/dnsmasq.conf

# Restart dnsmasq
sudo systemctl restart dnsmasq
```

**Verify DNS works:**
```bash
dig @127.0.0.2 test.aura
dig test.aura
```

---

## Shell Completion (Optional)

Enable tab completion for the `aura` command:

### Bash
```bash
# Add to ~/.bashrc
echo 'source <(aura completion bash)' >> ~/.bashrc
source ~/.bashrc
```

### Zsh
```bash
# Add to ~/.zshrc
echo 'source <(aura completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

### Fish
```bash
aura completion fish | source
# To make permanent:
aura completion fish > ~/.config/fish/completions/aura.fish
```

---

## Post-Installation Checklist

After installation, verify everything works:

```bash
# 1. Check CLI is installed
aura --version

# 2. Check Docker containers are running
aura status

# 3. Test DNS resolution
dig whoami.aura

# 4. Test HTTPS access
curl -I https://whoami.aura

# 5. Open in browser
open https://whoami.aura  # Should show WhoAmI test service
```

---

## Troubleshooting Installation

### "Docker daemon is not running"
```bash
# Start Docker Desktop (macOS)
open -a Docker

# Start Docker daemon (Linux)
sudo systemctl start docker
```

### "mkcert: command not found" (macOS)
```bash
# Install Homebrew first
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Then run aura install again
aura install
```

### "systemd-resolved is not active" (Linux)
Your system doesn't use systemd-resolved. Follow the [Manual DNS Configuration](#linux-without-systemd-resolved) instructions above.

### "Permission denied" errors
Make sure you're running installation commands with proper permissions:
```bash
# Make install requires sudo
sudo make install

# Aura install will prompt for sudo when needed
aura install
```

### Port conflicts (80/443 already in use)
```bash
# Find what's using the ports
sudo lsof -i :80
sudo lsof -i :443

# If it's on 127.0.0.1, that's fine - Aura uses 127.0.0.2
# If something is on 127.0.0.2, you'll need to stop it
```

---

## Uninstallation

To completely remove Aura:

```bash
# Uninstall Aura (removes all configuration and containers)
aura uninstall

# Remove the CLI binary
sudo rm /usr/local/bin/aura

# Optional: Remove mkcert
# macOS:
brew uninstall mkcert

# Linux:
sudo rm /usr/local/bin/mkcert
```

**What gets removed:**
- All Docker containers and volumes
- `~/.aura` directory (all certificates and configuration)
- DNS resolver configuration
- Loopback address configuration
- **NOT removed:** mkcert binary and CA (manual removal required if desired)

---

## Next Steps

After successful installation:
- [Add your first service →](EXAMPLES.md#your-first-service)
- [See example configurations →](EXAMPLES.md)
- [Troubleshooting guide →](TROUBLESHOOTING.md)
