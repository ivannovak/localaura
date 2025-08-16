# Aura Global Reverse Proxy

A Docker-based Caddy reverse proxy that provides HTTPS access to local development services using the `.aura` TLD with automatic SSL certificates via mkcert.

## Features

- 🔒 **Automatic HTTPS** with locally-trusted certificates using mkcert
- 🌐 **Custom TLD** - Use `.aura` domain for all your local services
- 🏷️ **Docker Labels** - Configure services using Docker labels (no config files!)
- 🎯 **Custom Loopback** - Uses `127.0.0.2` to avoid conflicts with existing services
- 🐳 **Docker Integration** - Works seamlessly with Docker containers
- 🔍 **Built-in WhoAmI** - Default debugging service at `https://whoami.aura`
- 📦 **Zero Dependencies** - Just Docker and mkcert (auto-installed)

## Quick Start

### 1. Initial Setup

Run the complete setup script:

```bash
./setup.sh
```

This will:
- Configure the custom loopback address (127.0.0.2)
- Install mkcert and set up the local CA
- Create necessary directories
- Make all scripts executable

### 2. Start the Proxy

```bash
docker-compose up -d
```

### 3. Add Your First Site

```bash
# For a local development server
./add-site.sh myapp.aura http://localhost:3000

# For a Docker container
./add-site.sh api.aura http://my-api-container:8080

# Interactive mode (will prompt for target URL)
./add-site.sh wiki.aura
```

### 4. Access Your Site

Open your browser and navigate to: `https://myapp.aura`

### 5. Test with WhoAmI

The proxy includes a built-in WhoAmI service for testing:
```
https://whoami.aura
```

This service displays request headers and connection information, useful for debugging proxy configuration.

## Architecture

```
aura/
├── docker-compose.yml      # Main proxy and WhoAmI service
├── docker-compose.example.yml # Example configurations with labels
├── certs/                 # SSL certificates
│   ├── ca.pem            # Local CA certificate
│   └── domains/          # Per-domain certificates
│       ├── whoami/       # Default WhoAmI certificate
│       │   ├── cert.pem
│       │   └── key.pem
│       └── myapp/        # Your app certificates
│           ├── cert.pem
│           └── key.pem
└── *.sh                  # Management scripts
```

## Management Commands

### Add a Site

```bash
./add-site.sh <domain> [target-url]

# Examples:
./add-site.sh app.aura http://localhost:3000
./add-site.sh api.aura http://api-container:8080
./add-site.sh admin.aura http://host.docker.internal:8000
```

### Remove a Site

```bash
./remove-site.sh <domain>

# Example:
./remove-site.sh app.aura
```

### List All Sites

```bash
./list-sites.sh
```

Output:
```
========================================
        Aura Proxy Sites
========================================

Caddy Status: 🟢 Running

Configured Sites:
-----------------

  📍 app.aura
     Target:      http://localhost:3000
     Certificate: ✓
     Hosts file:  ✓
     Config:      app.caddy

  📍 api.aura
     Target:      http://api-container:8080
     Certificate: ✓
     Hosts file:  ✓
     Config:      api.caddy
```

## Docker Integration

### Using Docker Labels

The easiest way to configure services is using Docker labels. See `docker-compose.example.yml` for comprehensive examples.

Basic example:
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
    name: aura-proxy
```

Common labels:
- `caddy`: Domain(s) to serve
- `caddy.reverse_proxy`: Proxy configuration
- `caddy.tls`: Certificate paths
- `caddy.encode`: Enable compression
- `caddy.header.<name>`: Set headers
- `caddy.basicauth`: Authentication

### Manual Configuration (Non-Docker Services)

For services not running in Docker, use the add-site.sh script:

### Target URL Formats

- **Host services**: `http://host.docker.internal:PORT`
- **Docker containers**: `http://container-name:PORT`
- **Local services**: `http://localhost:PORT`

## Advanced Features

### Wildcard Subdomains

Each domain certificate includes wildcard support:
- `app.aura` → Also covers `*.app.aura`
- `api.aura` → Also covers `*.api.aura`

### Custom Headers

The proxy automatically adds these headers:
- `X-Real-IP`
- `X-Forwarded-For`
- `X-Forwarded-Proto`

### Health Check

The proxy provides a health endpoint:
```bash
curl http://127.0.0.2:8080/health
```

## Troubleshooting

### Certificate Issues

If you encounter certificate warnings:

1. Ensure mkcert CA is installed:
   ```bash
   mkcert -install
   ```

2. Regenerate the certificate:
   ```bash
   ./remove-site.sh domain.aura
   ./add-site.sh domain.aura http://target
   ```

### Port Conflicts

The proxy uses `127.0.0.2` specifically to avoid conflicts. If you still have issues:

1. Check if the loopback is configured:
   ```bash
   ifconfig lo0 | grep 127.0.0.2  # macOS
   ip addr show lo | grep 127.0.0.2  # Linux
   ```

2. Re-run the loopback setup:
   ```bash
   ./setup-loopback.sh
   ```

### DNS Resolution

Sites are added to `/etc/hosts` automatically. If resolution fails:

1. Check hosts file:
   ```bash
   cat /etc/hosts | grep aura
   ```

2. Manually add if missing:
   ```bash
   echo "127.0.0.2    myapp.aura" | sudo tee -a /etc/hosts
   ```

### Container Connectivity

If containers can't be reached:

1. Ensure they're on the aura-proxy network:
   ```bash
   docker network ls | grep aura-proxy
   docker network inspect aura-proxy
   ```

2. Use the correct target format:
   - Wrong: `http://localhost:3000` (from container context)
   - Right: `http://container-name:3000`
   - Right: `http://host.docker.internal:3000` (for host services)

## Security Notes

- Certificates are generated locally using mkcert
- The CA is only trusted on your local machine
- Never share the certificates or CA with others
- This setup is for development only, not production

## Requirements

- Docker & Docker Compose
- macOS or Linux
- Sudo access (for loopback configuration and hosts file)

## License

MIT