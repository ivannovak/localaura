# Aura Global Reverse Proxy

Local HTTPS development proxy with a powerful CLI.

## Requirements

- Docker & Docker Compose
- macOS or Linux  
- Sudo access (for loopback address and /usr/local/bin)
- Go 1.19+ (only for building from source)

## Installation

### Option 1: Quick Install (Recommended)

```bash
# Clone and install
git clone https://github.com/yourusername/aura.git
cd aura
make install

# Set up the proxy system
aura install

# Start the proxy
aura start

# Test it works
open https://whoami.aura
```

### Option 2: Manual Build

```bash
# Clone the repository
git clone https://github.com/yourusername/aura.git
cd aura

# Build the CLI
go build -o aura ./cmd/aura

# Install to /usr/local/bin (requires sudo)
sudo mv aura /usr/local/bin/

# Set up the proxy system
aura install
```

### Option 3: Install from Release

```bash
# Download the latest release (example for macOS ARM64)
curl -L https://github.com/yourusername/aura/releases/latest/download/aura-darwin-arm64 -o aura

# Make executable and install
chmod +x aura
sudo mv aura /usr/local/bin/

# Set up the proxy system
aura install
```

### Enable Shell Completion (Optional)

For better CLI experience with tab completion:

```bash
# Bash (add to ~/.bashrc)
source <(aura completion bash)

# Zsh (add to ~/.zshrc)  
source <(aura completion zsh)

# Fish
aura completion fish | source
```

## Quick Start

```bash
# 1. Install the CLI (if not done)
make install

# 2. Set up Aura proxy system
aura install

# 3. Start the proxy
aura start

# 4. Test with built-in WhoAmI service
open https://whoami.aura
```

## CLI Commands

```bash
# System management
aura install              # Set up Aura proxy system
aura start                # Start proxy
aura stop                 # Stop proxy
aura status               # Check status
aura uninstall            # Remove Aura completely

# Certificate management
aura cert myapp           # Generate cert for myapp.aura
aura cert api.aura        # Full domain also works

# Debugging
aura logs                 # View logs
aura logs -f              # Follow logs

# Help
aura --help               # Show all commands
aura [command] --help     # Show command details
```

## Adding Services

### 1. Generate Certificate

```bash
aura cert myapp
```

### 2. Add Docker Labels

```yaml
services:
  myapp:
    image: nginx:alpine
    networks:
      - aura-proxy
    labels:
      caddy: myapp.aura
      caddy.reverse_proxy: "{{upstreams 80}}"
      caddy.tls: "/certs/domains/myapp/cert.pem /certs/domains/myapp/key.pem"

networks:
  aura-proxy:
    external: true
```

### 3. Start Service

```bash
docker compose up -d myapp
```

Service is immediately available at `https://myapp.aura`

## How It Works

- **CLI** manages the proxy system from anywhere
- **Caddy Docker Proxy** auto-discovers containers via labels
- **mkcert** provides locally-trusted certificates
- **Custom loopback** (127.0.0.2) avoids port conflicts

## File Locations

- `~/.aura/` - Configuration, certificates, and docker-compose files
- `/usr/local/bin/aura` - CLI binary

## Troubleshooting

### Permission Denied
```bash
# If you get permission errors during install
sudo make install
```

### Proxy Won't Start
```bash
# Check Docker is running
docker ps

# Check status
aura status

# View logs for errors
aura logs
```

### Certificate Issues
```bash
# Regenerate a certificate
aura cert myapp

# Check certificate location
ls ~/.aura/certs/domains/
```

### Completely Reset
```bash
# Remove everything and start fresh
aura uninstall
aura install
```

## Development

```bash
# Build CLI locally
make build

# Run without installing
./aura status

# Install development version
make install

# Clean build artifacts
make clean
```

## Examples

See `~/.aura/docker-compose.example.yml` after installation for:
- Multiple domains
- Custom headers
- WebSocket support
- Basic auth
- Rate limiting

## License

MIT