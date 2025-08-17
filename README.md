# Aura Global Reverse Proxy

Local HTTPS development proxy using Docker labels and the `.aura` TLD.

## Quick Start

```bash
# 1. Setup
./setup.sh

# 2. Start proxy
docker-compose up -d

# 3. Test
open https://whoami.aura
```

## Adding Services

### 1. Generate Certificate

```bash
./add-cert.sh myapp.aura
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
docker-compose up -d myapp
```

Service is immediately available at `https://myapp.aura`

## How It Works

- **Caddy Docker Proxy** auto-discovers containers via labels
- **mkcert** provides trusted local certificates  
- **Custom loopback** (127.0.0.2) avoids port conflicts
- **No restart needed** when adding/removing services

## Examples

See `docker-compose.example.yml` for:
- Multiple domains
- Custom headers
- WebSocket support
- Basic auth
- Rate limiting

## Commands

- `./setup.sh` - Initial setup
- `./add-cert.sh <domain>` - Generate certificate
- `docker-compose logs -f` - View logs

## Requirements

- Docker & Docker Compose
- macOS or Linux
- Sudo access