# Service Configuration Examples

Complete examples for configuring services with Aura proxy.

## Your First Service

Let's add a simple web application to Aura. This example shows the complete workflow.

### Step 1: Generate Certificate

```bash
aura cert myapp
```

This creates:
- `~/.aura/certs/domains/myapp/cert.pem`
- `~/.aura/certs/domains/myapp/key.pem`

The certificate includes:
- `myapp.aura` - Main domain
- `*.myapp.aura` - Wildcard for subdomains
- `localhost`, `127.0.0.1`, `127.0.0.2`, `::1` - Local access

### Step 2: Create docker-compose.yml

```yaml
version: '3.8'

services:
  myapp:
    image: nginx:alpine
    container_name: myapp
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

### Step 3: Start Service

```bash
docker compose up -d
```

### Step 4: Access Service

```bash
open https://myapp.aura
```

That's it! Your service is now available with automatic HTTPS.

---

## Label Reference

Caddy Docker Proxy uses Docker labels for configuration. Here's what each label does:

### Required Labels

```yaml
labels:
  # The domain(s) to serve - can be space-separated for multiple domains
  caddy: myapp.aura

  # Reverse proxy to the container
  # {{upstreams PORT}} automatically discovers container IP and routes to specified port
  caddy.reverse_proxy: "{{upstreams 80}}"

  # Path to TLS certificate and key
  # Format: /certs/domains/<domain-without-.aura>/cert.pem /certs/domains/<domain-without-.aura>/key.pem
  caddy.tls: "/certs/domains/myapp/cert.pem /certs/domains/myapp/key.pem"
```

### Common Optional Labels

```yaml
labels:
  # Enable gzip compression
  caddy.encode: gzip

  # Set custom headers
  caddy.header.X-Custom-Header: "value"
  caddy.header.Access-Control-Allow-Origin: "*"

  # Enable basic authentication
  caddy.basicauth: /*
  caddy.basicauth.admin: "$2a$14$YourHashedPasswordHere"

  # Rate limiting
  caddy.rate_limit: "remote_ip 10r/m"
```

---

## Example Configurations

### Simple Web Application

Basic nginx website:

```yaml
services:
  website:
    image: nginx:alpine
    container_name: my-website
    volumes:
      - ./public:/usr/share/nginx/html:ro
    networks:
      - aura-proxy
    labels:
      caddy: website.aura
      caddy.reverse_proxy: "{{upstreams 80}}"
      caddy.tls: "/certs/domains/website/cert.pem /certs/domains/website/key.pem"
      caddy.encode: gzip

networks:
  aura-proxy:
    external: true
```

**Usage:**
```bash
aura cert website
docker compose up -d
open https://website.aura
```

---

### API Service with CORS

Node.js API with CORS headers:

```yaml
services:
  api:
    image: node:alpine
    container_name: my-api
    command: ["node", "server.js"]
    working_dir: /app
    volumes:
      - ./api:/app
    networks:
      - aura-proxy
    labels:
      caddy: api.aura
      caddy.reverse_proxy: "{{upstreams 3000}}"
      caddy.tls: "/certs/domains/api/cert.pem /certs/domains/api/key.pem"
      # CORS headers
      caddy.header.Access-Control-Allow-Origin: "*"
      caddy.header.Access-Control-Allow-Methods: "GET, POST, PUT, DELETE, OPTIONS"
      caddy.header.Access-Control-Allow-Headers: "Content-Type, Authorization"
      # API versioning header
      caddy.header.X-API-Version: "v1"
      # Enable compression
      caddy.encode: gzip

networks:
  aura-proxy:
    external: true
```

---

### Multiple Domains for One Service

WordPress with multiple domain names:

```yaml
services:
  wordpress:
    image: wordpress:latest
    container_name: my-blog
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: wordpress
    networks:
      - aura-proxy
    labels:
      # Multiple domains (space-separated)
      caddy: "blog.aura www.blog.aura admin.blog.aura"
      caddy.reverse_proxy: "{{upstreams 80}}"
      caddy.tls: "/certs/domains/blog/cert.pem /certs/domains/blog/key.pem"
      caddy.encode: gzip

  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: rootpass
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: wordpress
    volumes:
      - db_data:/var/lib/mysql

volumes:
  db_data:

networks:
  aura-proxy:
    external: true
```

**Note:** All three domains (`blog.aura`, `www.blog.aura`, `admin.blog.aura`) need the same certificate since it includes `*.blog.aura` wildcard.

---

### WebSocket Application

Socket.io chat application:

```yaml
services:
  chat:
    image: socketio/chat-example
    container_name: chat-app
    networks:
      - aura-proxy
    labels:
      caddy: chat.aura
      caddy.reverse_proxy: "{{upstreams 3000}}"
      caddy.tls: "/certs/domains/chat/cert.pem /certs/domains/chat/key.pem"
      # WebSocket support is automatic in Caddy - no special configuration needed!

networks:
  aura-proxy:
    external: true
```

**How it works:** Caddy automatically detects WebSocket upgrade requests and handles them correctly.

---

### Protected Admin Panel

Basic authentication for sensitive services:

```yaml
services:
  admin:
    image: phpmyadmin/phpmyadmin
    container_name: database-admin
    environment:
      PMA_HOST: mysql
      PMA_USER: root
      PMA_PASSWORD: rootpass
    networks:
      - aura-proxy
    labels:
      caddy: dbadmin.aura
      # Protect all paths with basic auth
      caddy.basicauth: /*
      # Generate hash with: caddy hash-password
      caddy.basicauth.admin: "$2a$14$YourHashedPasswordHere"
      caddy.reverse_proxy: "{{upstreams 80}}"
      caddy.tls: "/certs/domains/dbadmin/cert.pem /certs/domains/dbadmin/key.pem"

networks:
  aura-proxy:
    external: true
```

**Generate password hash:**
```bash
# Run this in the aura-caddy container
docker exec -it aura-caddy caddy hash-password
# Enter your password when prompted
# Copy the hash to the basicauth label
```

---

### Rate-Limited Public API

Prevent API abuse with rate limiting:

```yaml
services:
  public-api:
    image: your-api-image
    container_name: public-api
    networks:
      - aura-proxy
    labels:
      caddy: api.public.aura
      caddy.reverse_proxy: "{{upstreams 8080}}"
      caddy.tls: "/certs/domains/public/cert.pem /certs/domains/public/key.pem"
      # Rate limit: 10 requests per minute per IP
      caddy.rate_limit: "remote_ip 10r/m"
      # Or 100 requests per hour
      # caddy.rate_limit: "remote_ip 100r/h"

networks:
  aura-proxy:
    external: true
```

**Rate limit formats:**
- `10r/s` - 10 requests per second
- `60r/m` - 60 requests per minute
- `1000r/h` - 1000 requests per hour

---

### Path-Based Routing

Serve multiple applications under different paths:

```yaml
services:
  # Main application at /
  main-app:
    image: your-main-app
    container_name: main
    networks:
      - aura-proxy
    labels:
      caddy: myapp.aura
      caddy.reverse_proxy: "{{upstreams 3000}}"
      caddy.tls: "/certs/domains/myapp/cert.pem /certs/domains/myapp/key.pem"

  # Documentation at /docs
  docs:
    image: docsify/docsify
    container_name: docs
    networks:
      - aura-proxy
    labels:
      caddy: myapp.aura
      caddy.route: /docs/*
      caddy.route.reverse_proxy: "{{upstreams 3000}}"
      caddy.route.strip_prefix: /docs

  # API at /api
  api:
    image: your-api
    container_name: api
    networks:
      - aura-proxy
    labels:
      caddy: myapp.aura
      caddy.route: /api/*
      caddy.route.reverse_proxy: "{{upstreams 8080}}"
      caddy.route.strip_prefix: /api

networks:
  aura-proxy:
    external: true
```

**Access:**
- `https://myapp.aura/` → main-app
- `https://myapp.aura/docs/` → docs
- `https://myapp.aura/api/` → api

---

### Static File Server

Serve static files with directory browsing:

```yaml
services:
  files:
    image: nginx:alpine
    container_name: file-server
    volumes:
      - ./shared:/usr/share/nginx/html:ro
    networks:
      - aura-proxy
    labels:
      caddy: files.aura
      caddy.file_server: ""
      caddy.file_server.root: "/usr/share/nginx/html"
      caddy.file_server.browse: ""
      caddy.tls: "/certs/domains/files/cert.pem /certs/domains/files/key.pem"

networks:
  aura-proxy:
    external: true
```

---

### Full-Stack Application

Complete example with frontend, backend, and database:

```yaml
version: '3.8'

services:
  frontend:
    image: your-frontend
    container_name: myapp-frontend
    networks:
      - aura-proxy
    labels:
      caddy: myapp.aura
      caddy.reverse_proxy: "{{upstreams 3000}}"
      caddy.tls: "/certs/domains/myapp/cert.pem /certs/domains/myapp/key.pem"
      caddy.encode: gzip

  backend:
    image: your-backend
    container_name: myapp-backend
    environment:
      DATABASE_URL: postgresql://postgres:password@db:5432/myapp
    networks:
      - aura-proxy
    labels:
      caddy: api.myapp.aura
      caddy.reverse_proxy: "{{upstreams 8080}}"
      caddy.tls: "/certs/domains/myapp/cert.pem /certs/domains/myapp/key.pem"
      caddy.header.Access-Control-Allow-Origin: "https://myapp.aura"

  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: password
      POSTGRES_DB: myapp
    volumes:
      - db_data:/var/lib/postgresql/data
    # Database doesn't need to be on aura-proxy network

volumes:
  db_data:

networks:
  aura-proxy:
    external: true
```

**Setup:**
```bash
# Generate certificates (only need one since *.myapp.aura wildcard covers both)
aura cert myapp

# Start all services
docker compose up -d

# Access
open https://myapp.aura          # Frontend
open https://api.myapp.aura      # Backend API
```

---

## Certificate Management

### Certificate Naming Convention

Certificates are stored based on the domain name without `.aura`:

| Domain | Certificate Directory | Certificate Paths |
|--------|----------------------|-------------------|
| `myapp.aura` | `~/.aura/certs/domains/myapp/` | `cert.pem`, `key.pem` |
| `api.aura` | `~/.aura/certs/domains/api/` | `cert.pem`, `key.pem` |
| `blog.example.aura` | `~/.aura/certs/domains/blog.example/` | `cert.pem`, `key.pem` |

### Wildcard Coverage

Each generated certificate includes a wildcard for subdomains:

```bash
aura cert myapp
# Creates certificate valid for:
# - myapp.aura
# - *.myapp.aura (api.myapp.aura, admin.myapp.aura, etc.)
# - localhost, 127.0.0.1, 127.0.0.2, ::1
```

This means one certificate can serve multiple subdomains:

```yaml
labels:
  # All these work with the same certificate
  caddy: "myapp.aura api.myapp.aura admin.myapp.aura"
  caddy.tls: "/certs/domains/myapp/cert.pem /certs/domains/myapp/key.pem"
```

### Regenerating Certificates

```bash
# Regenerate an existing certificate
aura cert myapp
# Prompts for confirmation before overwriting

# Auto-confirm (useful for scripts)
yes | aura cert myapp
```

---

## Complete Example: Microservices Architecture

```yaml
version: '3.8'

services:
  # Frontend React app
  web:
    build: ./frontend
    container_name: shop-web
    networks:
      - aura-proxy
    labels:
      caddy: shop.aura
      caddy.reverse_proxy: "{{upstreams 3000}}"
      caddy.tls: "/certs/domains/shop/cert.pem /certs/domains/shop/key.pem"
      caddy.encode: gzip

  # Product API
  product-api:
    build: ./services/products
    container_name: shop-products
    environment:
      DB_HOST: postgres
    networks:
      - aura-proxy
      - backend
    labels:
      caddy: api.shop.aura
      caddy.route: /products/*
      caddy.route.reverse_proxy: "{{upstreams 8080}}"
      caddy.tls: "/certs/domains/shop/cert.pem /certs/domains/shop/key.pem"

  # Order API
  order-api:
    build: ./services/orders
    container_name: shop-orders
    environment:
      DB_HOST: postgres
    networks:
      - aura-proxy
      - backend
    labels:
      caddy: api.shop.aura
      caddy.route: /orders/*
      caddy.route.reverse_proxy: "{{upstreams 8080}}"
      caddy.tls: "/certs/domains/shop/cert.pem /certs/domains/shop/key.pem"

  # Admin panel (protected)
  admin:
    build: ./admin
    container_name: shop-admin
    networks:
      - aura-proxy
    labels:
      caddy: admin.shop.aura
      caddy.basicauth: /*
      caddy.basicauth.admin: "$2a$14$YourHashedPasswordHere"
      caddy.reverse_proxy: "{{upstreams 3000}}"
      caddy.tls: "/certs/domains/shop/cert.pem /certs/domains/shop/key.pem"

  # Database (not on aura-proxy network)
  postgres:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: shoppass
      POSTGRES_DB: shop
    volumes:
      - pg_data:/var/lib/postgresql/data
    networks:
      - backend

volumes:
  pg_data:

networks:
  aura-proxy:
    external: true
  backend:
    driver: bridge
```

**Setup:**
```bash
aura cert shop
docker compose up -d
```

**Access:**
- Frontend: `https://shop.aura`
- Product API: `https://api.shop.aura/products/`
- Order API: `https://api.shop.aura/orders/`
- Admin: `https://admin.shop.aura` (requires authentication)

---

## Tips and Best Practices

### 1. Use Wildcards Efficiently

Generate one certificate per base domain, use wildcard for subdomains:

```bash
# Good: One cert for all subdomains
aura cert myapp
# Covers: myapp.aura, api.myapp.aura, admin.myapp.aura, etc.

# Avoid: Separate certs for each subdomain (unnecessary)
aura cert api.myapp
aura cert admin.myapp
```

### 2. Keep Secrets Out of Labels

Don't put real passwords in Docker Compose files:

```yaml
# Bad: Password in compose file
labels:
  caddy.basicauth.admin: "hardcoded-hash"

# Good: Use environment variable
labels:
  caddy.basicauth.admin: "${ADMIN_PASSWORD_HASH}"
```

```bash
# .env file
ADMIN_PASSWORD_HASH=$2a$14$YourHashedPasswordHere
```

### 3. Network Isolation

Only expose services that need to be publicly accessible:

```yaml
services:
  # Public-facing - on aura-proxy network
  web:
    networks:
      - aura-proxy
      - app

  # Internal only - NOT on aura-proxy network
  database:
    networks:
      - app

  redis:
    networks:
      - app
```

### 4. Compression for Better Performance

Always enable compression for text-based content:

```yaml
labels:
  caddy.encode: gzip  # HTML, CSS, JS, JSON, etc.
```

### 5. Debugging Labels

Check how Caddy interpreted your labels:

```bash
# View generated Caddyfile
docker logs aura-caddy 2>&1 | grep "New Caddyfile"

# View generated JSON config
docker logs aura-caddy 2>&1 | grep "New Config JSON"
```

---

## Next Steps

- [Troubleshooting guide →](TROUBLESHOOTING.md)
- [Advanced configuration →](ADVANCED.md)
- [Full docker-compose.example.yml](../docker-compose.example.yml)
