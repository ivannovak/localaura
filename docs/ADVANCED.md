# Advanced Configuration

Advanced topics for power users and production-like local environments.

## Architecture Overview

```
┌──────────────────────────────────────────────────────────┐
│ Browser/Client                                            │
└────────────────┬─────────────────────────────────────────┘
                 │ HTTPS Request to myapp.aura
                 ▼
┌──────────────────────────────────────────────────────────┐
│ DNS Resolution (via CoreDNS on 127.0.0.2:53)             │
│ myapp.aura → 127.0.0.2                                    │
└────────────────┬─────────────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────────────┐
│ Caddy Reverse Proxy (127.0.0.2:443)                      │
│ - TLS termination                                         │
│ - Auto-discovery via Docker labels                       │
│ - HTTP/2, HTTP/3 support                                 │
└────────────────┬─────────────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────────────┐
│ Docker Network (aura-proxy)                              │
│ Caddy connects to containers by name                     │
└────────────────┬─────────────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────────────┐
│ Your Service Container                                    │
│ Running on internal port (e.g., 80, 3000, 8080)          │
└──────────────────────────────────────────────────────────┘
```

---

## Custom Caddy Configuration

### Using Custom Caddyfile Snippets

For complex configurations, you can use Caddy's advanced directives:

```yaml
services:
  myapp:
    networks:
      - aura-proxy
    labels:
      # Custom matcher for API endpoints
      caddy: myapp.aura
      caddy.@api.path: /api/*
      caddy.@api.reverse_proxy: "{{upstreams 8080}}"

      # Different handler for static files
      caddy.@static.path: /static/*
      caddy.@static.file_server: ""
      caddy.@static.file_server.root: /var/www/static

      # Default handler
      caddy.reverse_proxy: "{{upstreams 3000}}"
      caddy.tls: "/certs/domains/myapp/cert.pem /certs/domains/myapp/key.pem"
```

### Custom Response Headers

Add security headers or custom metadata:

```yaml
labels:
  # Security headers
  caddy.header.Strict-Transport-Security: "max-age=31536000; includeSubDomains"
  caddy.header.X-Content-Type-Options: "nosniff"
  caddy.header.X-Frame-Options: "SAMEORIGIN"
  caddy.header.X-XSS-Protection: "1; mode=block"
  caddy.header.Referrer-Policy: "strict-origin-when-cross-origin"

  # Custom headers
  caddy.header.X-App-Version: "1.0.0"
  caddy.header.X-Environment: "development"
```

### Request/Response Manipulation

```yaml
labels:
  # Remove headers from response
  caddy.header.-Server: ""
  caddy.header.-X-Powered-By: ""

  # Add CORS with credentials
  caddy.header.Access-Control-Allow-Origin: "https://frontend.aura"
  caddy.header.Access-Control-Allow-Credentials: "true"
  caddy.header.Access-Control-Allow-Methods: "GET, POST, PUT, PATCH, DELETE, OPTIONS"
  caddy.header.Access-Control-Allow-Headers: "Content-Type, Authorization, X-Requested-With"
```

---

## Performance Tuning

### Enable HTTP/3 (QUIC)

HTTP/3 is enabled by default in Caddy. Verify it's working:

```bash
# Check if HTTP/3 is enabled
curl -I --http3 https://myapp.aura

# Should show:
# alt-svc: h3=":443"; ma=2592000
```

### Compression Configuration

```yaml
labels:
  # Aggressive compression for APIs
  caddy.encode: "zstd gzip"

  # Or fine-tune compression
  caddy.encode.zstd: ""
  caddy.encode.gzip: "level 9"  # Maximum compression
```

### Connection Pooling

Caddy handles connection pooling automatically, but you can tune upstream connections:

```yaml
labels:
  # Configure upstream behavior
  caddy.reverse_proxy: "{{upstreams 8080}}"
  caddy.reverse_proxy.transport: "http"
  caddy.reverse_proxy.transport.max_conns_per_host: "100"
  caddy.reverse_proxy.transport.idle_conn_timeout: "90s"
```

### Caching Responses

```yaml
labels:
  # Cache static assets
  caddy.@static.path: /assets/* /images/* /css/* /js/*
  caddy.@static.header.Cache-Control: "public, max-age=31536000, immutable"

  # Don't cache API responses
  caddy.@api.path: /api/*
  caddy.@api.header.Cache-Control: "no-store, no-cache, must-revalidate"
```

---

## Advanced Load Balancing

### Multiple Container Instances

```yaml
services:
  api-1:
    image: my-api
    container_name: api-1
    networks:
      - aura-proxy
    labels:
      caddy: api.aura
      caddy.reverse_proxy: "{{upstreams 8080}}"
      caddy.tls: "/certs/domains/api/cert.pem /certs/domains/api/key.pem"

  api-2:
    image: my-api
    container_name: api-2
    networks:
      - aura-proxy
    labels:
      caddy: api.aura
      caddy.reverse_proxy: "{{upstreams 8080}}"
      caddy.tls: "/certs/domains/api/cert.pem /certs/domains/api/key.pem"

  api-3:
    image: my-api
    container_name: api-3
    networks:
      - aura-proxy
    labels:
      caddy: api.aura
      caddy.reverse_proxy: "{{upstreams 8080}}"
      caddy.tls: "/certs/domains/api/cert.pem /certs/domains/api/key.pem"

networks:
  aura-proxy:
    external: true
```

Caddy automatically load balances across all three containers using round-robin.

### Health Checks

```yaml
labels:
  caddy.reverse_proxy: "{{upstreams 8080}}"
  caddy.reverse_proxy.health_uri: "/health"
  caddy.reverse_proxy.health_interval: "30s"
  caddy.reverse_proxy.health_timeout: "5s"
```

---

## Custom DNS Configuration

### Additional TLDs

Want to use multiple TLDs beyond `.aura`?

**Edit CoreDNS Corefile:**
```bash
vim ~/.aura/coredns/Corefile
```

```
# Add another TLD
aura:53 {
    template IN A aura {
        answer "{{ .Name }} 0 IN A 127.0.0.2"
        rcode NOERROR
    }
    errors
    log
}

# New TLD
local:53 {
    template IN A local {
        answer "{{ .Name }} 0 IN A 127.0.0.2"
        rcode NOERROR
    }
    errors
    log
}
```

**Add resolver (macOS):**
```bash
echo "nameserver 127.0.0.2" | sudo tee /etc/resolver/local
```

**Restart CoreDNS:**
```bash
docker restart aura-coredns
```

### Custom DNS Records

```bash
# Edit Corefile
vim ~/.aura/coredns/Corefile
```

```
aura:53 {
    # Specific override for one domain
    file /config/db.aura {
        api.aura
    }

    # Wildcard for everything else
    template IN A aura {
        answer "{{ .Name }} 0 IN A 127.0.0.2"
        rcode NOERROR
    }
}
```

**Create zone file:**
```bash
cat > ~/.aura/coredns/db.aura << 'EOF'
api.aura.  IN A 127.0.0.3
EOF
```

---

## SSL/TLS Advanced Configuration

### Custom Certificate Authority

Use your own CA instead of mkcert:

```bash
# Generate your own CA
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt

# Generate service cert
openssl genrsa -out myapp.key 2048
openssl req -new -key myapp.key -out myapp.csr
openssl x509 -req -in myapp.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out myapp.crt -days 365

# Use in docker-compose
caddy.tls: "/custom/certs/myapp.crt /custom/certs/myapp.key"
```

### Client Certificate Authentication

```yaml
labels:
  caddy.tls: "/certs/domains/admin/cert.pem /certs/domains/admin/key.pem"
  caddy.tls.client_auth: "request"
  caddy.tls.client_auth.mode: "require_and_verify"
  caddy.tls.client_auth.trusted_ca_cert_file: "/certs/client-ca.pem"
```

---

## Multi-Environment Setup

### Environment-Specific Configuration

```yaml
# docker-compose.dev.yml
services:
  app:
    environment:
      - APP_ENV=development
      - DEBUG=true
    labels:
      caddy: app.dev.aura

# docker-compose.staging.yml
services:
  app:
    environment:
      - APP_ENV=staging
      - DEBUG=false
    labels:
      caddy: app.staging.aura
```

**Run specific environment:**
```bash
# Generate certs for both
aura cert dev
aura cert staging

# Run dev
docker compose -f docker-compose.dev.yml up -d

# Or staging
docker compose -f docker-compose.staging.yml up -d
```

---

## Monitoring and Observability

### Caddy Metrics

Caddy exports Prometheus metrics:

```yaml
services:
  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    networks:
      - aura-proxy
    labels:
      caddy: metrics.aura
      caddy.reverse_proxy: "{{upstreams 9090}}"
      caddy.tls: "/certs/domains/metrics/cert.pem /certs/domains/metrics/key.pem"
```

**prometheus.yml:**
```yaml
scrape_configs:
  - job_name: 'caddy'
    static_configs:
      - targets: ['aura-caddy:2019']
```

### Access Logging

```yaml
labels:
  # Enable access logging with custom format
  caddy.log: ""
  caddy.log.output: "file /var/log/access.log"
  caddy.log.format: json
  caddy.log.format.time_format: "iso8601"
```

### Request Tracing

```yaml
labels:
  # Add request ID to all requests
  caddy.header.X-Request-ID: "{http.request.uuid}"

  # Log request ID
  caddy.log.format.message: "{request>headers>X-Request-Id} {request>method} {request>uri}"
```

---

## Security Hardening

### Rate Limiting per Endpoint

```yaml
labels:
  # Public endpoints: strict limits
  caddy.@public.path: /public/*
  caddy.@public.rate_limit: "remote_ip 10r/m"

  # Authenticated endpoints: relaxed limits
  caddy.@auth.path: /api/*
  caddy.@auth.rate_limit: "remote_ip 100r/m"
```

### IP Whitelisting

```yaml
labels:
  # Only allow from specific IPs
  caddy.@admin.path: /admin/*
  caddy.@admin.remote_ip: "192.168.1.0/24 10.0.0.0/8"
  caddy.@admin.reverse_proxy: "{{upstreams 8080}}"

  # Block all others
  caddy.@admin.respond: "403"
```

### Request Size Limits

```yaml
labels:
  # Limit request body size to 10MB
  caddy.request_body.max_size: "10MB"
```

---

## Custom Error Pages

```yaml
labels:
  # Custom error pages
  caddy.handle_errors: ""
  caddy.handle_errors.@404.expression: "{http.error.status_code} == 404"
  caddy.handle_errors.@404.rewrite: "/errors/404.html"
  caddy.handle_errors.@500.expression: "{http.error.status_code} >= 500"
  caddy.handle_errors.@500.rewrite: "/errors/500.html"
  caddy.handle_errors.file_server.root: "/var/www/errors"
```

---

## Integration with External Tools

### ngrok Alternative for External Access

```bash
# Expose local Aura to internet via SSH tunnel
ssh -R 80:127.0.0.2:443 user@your-server.com

# Or use Cloudflare Tunnel
cloudflared tunnel --url https://127.0.0.2:443
```

### CI/CD Integration

```yaml
# .github/workflows/test.yml
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      docker:
        image: docker:dind
    steps:
      - uses: actions/checkout@v2

      - name: Install Aura
        run: |
          make install
          aura install

      - name: Start services
        run: |
          aura cert myapp
          docker compose up -d

      - name: Run tests
        run: |
          curl -k https://myapp.aura
          npm test
```

---

## Performance Benchmarking

### Load Testing

```bash
# Install hey
go install github.com/rakyll/hey@latest

# Test throughput
hey -n 10000 -c 100 https://myapp.aura

# Test with custom headers
hey -n 1000 -H "Authorization: Bearer token" https://api.aura/endpoint
```

### Profiling Caddy

```bash
# Enable pprof in Caddy
docker exec aura-caddy caddy run --config /etc/caddy/Caddyfile --debug

# Access profiling data
curl http://localhost:2019/debug/pprof/
```

---

## Backup and Restore

### Backup Configuration

```bash
#!/bin/bash
# backup-aura.sh

BACKUP_DIR=~/aura-backup-$(date +%Y%m%d)

mkdir -p $BACKUP_DIR

# Backup certificates
cp -r ~/.aura/certs $BACKUP_DIR/

# Backup configuration
cp ~/.aura/docker-compose.yml $BACKUP_DIR/
cp ~/.aura/coredns/Corefile $BACKUP_DIR/

# Backup Docker volumes
docker run --rm -v aura_caddy_data:/data -v $BACKUP_DIR:/backup alpine tar czf /backup/caddy_data.tar.gz -C /data .

echo "Backup complete: $BACKUP_DIR"
```

### Restore

```bash
#!/bin/bash
# restore-aura.sh

BACKUP_DIR=$1

# Restore certificates
cp -r $BACKUP_DIR/certs ~/.aura/

# Restore configuration
cp $BACKUP_DIR/docker-compose.yml ~/.aura/
cp $BACKUP_DIR/Corefile ~/.aura/coredns/

# Restore Docker volumes
docker run --rm -v aura_caddy_data:/data -v $BACKUP_DIR:/backup alpine tar xzf /backup/caddy_data.tar.gz -C /data

# Restart Aura
aura stop
aura start
```

---

## Extending Aura

### Adding Custom Caddy Modules

Create a custom Caddy build with additional modules:

```dockerfile
# Dockerfile.caddy-custom
FROM caddy:builder AS builder

RUN xcaddy build \
    --with github.com/caddy-dns/cloudflare \
    --with github.com/greenpau/caddy-security

FROM caddy:latest
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

**Build and use:**
```bash
docker build -t custom-caddy -f Dockerfile.caddy-custom .

# Update docker-compose.yml
services:
  caddy:
    image: custom-caddy
```

---

## Next Steps

- [Installation guide →](INSTALL.md)
- [Examples →](EXAMPLES.md)
- [Troubleshooting →](TROUBLESHOOTING.md)
- [Caddy Docker Proxy Documentation](https://github.com/lucaslorentz/caddy-docker-proxy)
- [CoreDNS Documentation](https://coredns.io/manual/toc/)
