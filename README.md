<p align="center">
  <img src=".github/assets/aura-header.png" alt="Aura" width="800">
</p>

# Aura

[![CI](https://github.com/ivannovak/aura/actions/workflows/ci.yml/badge.svg)](https://github.com/ivannovak/aura/actions/workflows/ci.yml)
[![Release](https://github.com/ivannovak/aura/actions/workflows/release.yml/badge.svg)](https://github.com/ivannovak/aura/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ivannovak/aura)](https://goreportcard.com/report/github.com/ivannovak/aura)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Local HTTPS development proxy with automatic DNS and certificates.**

Stop fighting with `/etc/hosts` and self-signed certificate warnings. Aura gives you instant `*.aura` domains with trusted HTTPS for all your Docker services.

---

## What is Aura?

**Before Aura:**
```bash
# Edit /etc/hosts for every service
sudo echo "127.0.0.1 myapp.local" >> /etc/hosts
sudo echo "127.0.0.1 api.local" >> /etc/hosts

# Generate certificates manually
openssl req -x509 -newkey rsa:4096 ...

# Configure nginx/apache
# Deal with "Not Secure" warnings
# Repeat for every new service
```

**With Aura:**
```bash
# One-time setup
aura install

# For each service:
aura cert myapp
docker compose up -d

# Done! https://myapp.aura just works
```

---

## Quick Start

### Install

```bash
git clone https://github.com/ivannovak/aura.git
cd aura
make install && aura install && aura start
```

**Test it:** Open https://whoami.aura

### Add Your First Service

**1. Generate certificate:**
```bash
aura cert myapp
```

**2. Create docker-compose.yml:**
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

**3. Start service:**
```bash
docker compose up -d
```

**4. Access:** https://myapp.aura ✨

---

## How It Works

- 🌐 **CoreDNS** - All `*.aura` domains automatically resolve to `127.0.0.2`
- 🔒 **mkcert** - Generates locally-trusted HTTPS certificates
- 🔄 **Caddy** - Reverse proxy with automatic service discovery via Docker labels
- 🚀 **Custom loopback** - Uses `127.0.0.2` to avoid conflicts with other local services

No `/etc/hosts` editing. No certificate warnings. Just add a label and go.

---

## Common Tasks

```bash
# Manage Aura
aura install              # Set up proxy system
aura start                # Start proxy
aura stop                 # Stop proxy
aura status               # Check status
aura logs                 # View logs
aura logs -f              # Follow logs
aura uninstall            # Remove completely

# Certificates
aura cert myapp           # Generate cert for myapp.aura
aura cert api             # Short form works too

# Help
aura --help               # All commands
aura cert --help          # Command help
```

---

## Requirements

- **Docker & Docker Compose** (with daemon running)
- **macOS** or **Linux**
- **Sudo access** (for network and DNS configuration)
- **Platform-specific:**
  - macOS: Homebrew (for mkcert installation)
  - Linux: systemd-resolved (for automatic DNS setup)

[→ Detailed installation requirements](docs/INSTALL.md)

---

## Documentation

- **[Installation Guide](docs/INSTALL.md)** - Platform-specific setup instructions
- **[Service Examples](docs/EXAMPLES.md)** - Common configurations and patterns
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Solutions to common issues
- **[Advanced Usage](docs/ADVANCED.md)** - Performance tuning, custom configs, monitoring

---

## Quick Troubleshooting

### Service returns 502 Bad Gateway
```bash
# Check if service is running and on aura-proxy network
docker ps | grep myservice
docker network inspect aura-proxy | grep myservice
```

### Domain doesn't resolve
```bash
# Test DNS directly
dig whoami.aura

# Should return 127.0.0.2
# If not, see: docs/TROUBLESHOOTING.md#dns-resolution-issues
```

### Certificate not trusted
```bash
# Reinstall mkcert CA
mkcert -install

# Restart browser
```

[→ Full troubleshooting guide](docs/TROUBLESHOOTING.md)

---

## Examples

<details>
<summary><b>API with CORS headers</b></summary>

```yaml
services:
  api:
    image: my-api
    networks:
      - aura-proxy
    labels:
      caddy: api.aura
      caddy.reverse_proxy: "{{upstreams 3000}}"
      caddy.tls: "/certs/domains/api/cert.pem /certs/domains/api/key.pem"
      caddy.header.Access-Control-Allow-Origin: "*"
      caddy.encode: gzip
```
</details>

<details>
<summary><b>Multiple domains for one service</b></summary>

```yaml
services:
  web:
    image: nginx
    networks:
      - aura-proxy
    labels:
      caddy: "app.aura www.app.aura admin.app.aura"
      caddy.reverse_proxy: "{{upstreams 80}}"
      caddy.tls: "/certs/domains/app/cert.pem /certs/domains/app/key.pem"
```
</details>

<details>
<summary><b>Protected admin panel</b></summary>

```yaml
services:
  admin:
    image: phpmyadmin
    networks:
      - aura-proxy
    labels:
      caddy: admin.aura
      caddy.basicauth: /*
      caddy.basicauth.admin: "$2a$14$YourHashedPasswordHere"
      caddy.reverse_proxy: "{{upstreams 80}}"
      caddy.tls: "/certs/domains/admin/cert.pem /certs/domains/admin/key.pem"
```
</details>

[→ More examples](docs/EXAMPLES.md)

---

## Why Aura?

✅ **Zero configuration DNS** - All `*.aura` domains work automatically
✅ **Trusted certificates** - No browser warnings
✅ **Auto-discovery** - Add a Docker label, get HTTPS
✅ **Herd compatible** - Works alongside Laravel Herd's `.test` domains
✅ **Clean uninstall** - `aura uninstall` removes everything
✅ **Modern stack** - HTTP/2, HTTP/3, WebSocket support

---

## File Locations

| Path | Purpose |
|------|---------|
| `~/.aura/` | Configuration and certificates |
| `~/.aura/certs/domains/` | Generated certificates |
| `/usr/local/bin/aura` | CLI binary |
| `/etc/resolver/aura` | DNS resolver (macOS) |
| `/etc/systemd/resolved.conf.d/aura.conf` | DNS resolver (Linux) |

[→ Complete file structure](docs/INSTALL.md#file-locations)

---

## Contributing

Issues and pull requests welcome! Please see:
- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](.github/CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)

---

## License

MIT

---

<p align="center">
  <img src=".github/assets/aura-orb.png" alt="Aura" width="400">
</p>
