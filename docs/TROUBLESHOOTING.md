# Troubleshooting Guide

Solutions to common issues when using Aura proxy.

## Quick Diagnostic Commands

Run these first to get system status:

```bash
# Check if Aura is running
aura status

# Check all containers
docker ps --filter name=aura-

# Test DNS resolution
dig whoami.aura

# Test HTTPS access
curl -I https://whoami.aura

# Check Caddy logs
aura logs --tail 50
```

---

## Installation Issues

### "Permission denied" during installation

**Symptom:**
```
Permission denied while trying to connect to the Docker daemon socket
```

**Cause:** Your user doesn't have Docker permissions.

**Solution:**
```bash
# Add your user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker

# Or use sudo
sudo docker ps
```

---

### "Homebrew not found" (macOS)

**Symptom:**
```
brew: command not found
```

**Cause:** Homebrew is required for mkcert installation on macOS.

**Solution:**
```bash
# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Then re-run
aura install
```

---

### "systemd-resolved is not active" (Linux)

**Symptom:**
```
⚠ Manual DNS configuration required
systemd-resolved is not active
```

**Cause:** Your Linux system doesn't use systemd-resolved.

**Solution:** See [Manual DNS Configuration](INSTALL.md#linux-without-systemd-resolved) in the installation guide.

---

### "Port already in use"

**Symptom:**
```
Error starting userland proxy: listen tcp4 127.0.0.2:443: bind: address already in use
```

**Cause:** Another service is using port 80 or 443 on 127.0.0.2.

**Solution:**
```bash
# Find what's using the port
sudo lsof -i :443 | grep 127.0.0.2

# If it's another instance of Aura:
docker compose down
aura start

# If it's another service, you'll need to stop it or use a different address
```

---

### Partial installation (setup failed halfway)

**Symptom:** Installation script failed, system is in unknown state.

**Solution:**
```bash
# Complete uninstall
aura uninstall

# Manual cleanup if uninstall fails
sudo rm -rf ~/.aura
sudo rm /etc/resolver/aura  # macOS
sudo rm /etc/systemd/resolved.conf.d/aura.conf  # Linux
sudo rm /Library/LaunchDaemons/com.aura.loopback.plist  # macOS
sudo systemctl disable --now aura-loopback  # Linux

# Reinstall
aura install
```

---

## DNS Resolution Issues

### Domain doesn't resolve

**Symptom:**
```bash
$ dig whoami.aura
# Returns no answer
```

**Diagnostic Steps:**

```bash
# 1. Test if CoreDNS is working
dig @127.0.0.2 whoami.aura
# Should return 127.0.0.2

# 2. Check if CoreDNS container is running
docker ps | grep coredns

# 3. Check CoreDNS logs
docker logs aura-coredns

# 4. macOS: Check resolver file
cat /etc/resolver/aura
# Should contain: nameserver 127.0.0.2

# 5. Linux: Check systemd-resolved
resolvectl status | grep -A 5 aura
```

**Solutions by Platform:**

**macOS:**
```bash
# Recreate resolver file
echo "nameserver 127.0.0.2" | sudo tee /etc/resolver/aura

# Flush DNS cache
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder

# Test again
dig whoami.aura
```

**Linux:**
```bash
# Restart systemd-resolved
sudo systemctl restart systemd-resolved

# Check configuration
cat /etc/systemd/resolved.conf.d/aura.conf

# Test again
resolvectl query whoami.aura
```

---

### DNS works but browser doesn't

**Symptom:** `dig whoami.aura` works, but browser shows "can't connect".

**Cause:** Browser is caching old DNS results or using different DNS.

**Solution:**
```bash
# Clear browser DNS cache
# Chrome: chrome://net-internals/#dns → "Clear host cache"
# Firefox: about:networking#dns → "Clear DNS Cache"
# Safari: Quit and reopen Safari

# Or restart browser completely
```

---

### Slow DNS resolution

**Symptom:** First request takes 5+ seconds, subsequent requests are fast.

**Cause:** CoreDNS cold start or system DNS timeout.

**Solution:**
```bash
# Check CoreDNS is running
docker ps | grep coredns

# Restart CoreDNS
docker restart aura-coredns

# Test resolution speed
time dig whoami.aura
# Should be < 100ms
```

---

## Certificate Issues

### "Certificate not trusted" warning in browser

**Symptom:** Browser shows security warning despite using HTTPS.

**Cause:** mkcert CA not installed in browser trust store.

**Solution:**
```bash
# Reinstall mkcert CA
mkcert -install

# Verify CA is installed
# macOS:
security find-certificate -c "mkcert" -a

# Linux:
ls -la ~/.local/share/mkcert

# Restart browser
```

---

### "Certificate expired" or invalid

**Symptom:** Browser shows cert error even though cert should be valid.

**Solution:**
```bash
# Check certificate validity
openssl x509 -in ~/.aura/certs/domains/whoami/cert.pem -noout -dates

# Regenerate certificate
aura cert whoami

# Restart service
docker compose restart
```

---

### Service can't find certificate files

**Symptom:**
```
Error loading certificate: no such file or directory
```

**Cause:** Certificate wasn't generated or is in wrong location.

**Solution:**
```bash
# Check certificate exists
ls -la ~/.aura/certs/domains/myapp/

# If missing, generate it
aura cert myapp

# Verify path in docker-compose.yml matches:
# /certs/domains/myapp/cert.pem

# Remember: no .aura in the path!
# Wrong: /certs/domains/myapp.aura/cert.pem
# Right: /certs/domains/myapp/cert.pem
```

---

## Service Access Issues

### Service shows "502 Bad Gateway"

**Symptom:** HTTPS works but shows Caddy 502 error.

**Cause:** Service container isn't running or isn't on the aura-proxy network.

**Diagnostic:**
```bash
# 1. Check if service container is running
docker ps | grep myapp

# 2. Check if it's on aura-proxy network
docker network inspect aura-proxy | grep myapp

# 3. Check Caddy can reach the service
docker exec aura-caddy wget -O- http://myapp:80
```

**Solution:**
```yaml
# Ensure service is on aura-proxy network
services:
  myapp:
    networks:
      - aura-proxy

networks:
  aura-proxy:
    external: true
```

---

### Service shows "404 Not Found"

**Symptom:** HTTPS works but shows 404 error.

**Cause:** Caddy isn't configured to proxy this domain.

**Diagnostic:**
```bash
# Check if Caddy knows about your domain
docker logs aura-caddy 2>&1 | grep "myapp.aura"

# Check what domains Caddy is serving
docker logs aura-caddy 2>&1 | grep "New Caddyfile" | tail -1
```

**Solution:**
```yaml
# Ensure labels are correct
labels:
  caddy: myapp.aura  # Domain to serve
  caddy.reverse_proxy: "{{upstreams 80}}"  # Port inside container
  caddy.tls: "/certs/domains/myapp/cert.pem /certs/domains/myapp/key.pem"
```

```bash
# Restart service after fixing labels
docker compose up -d
```

---

### "Connection refused" or "Can't connect"

**Symptom:** Browser shows connection refused before HTTPS handshake.

**Diagnostic:**
```bash
# 1. Check Caddy is listening on 127.0.0.2
lsof -nP -iTCP:443 | grep 127.0.0.2

# 2. Check DNS resolves correctly
dig myapp.aura
# Should return 127.0.0.2

# 3. Check Caddy container is running
docker ps | grep caddy

# 4. Try direct connection
curl -v https://127.0.0.2 -H "Host: myapp.aura"
```

**Solution:**
```bash
# Restart Aura
aura stop
aura start

# Check status
aura status
```

---

### WebSocket connection fails

**Symptom:** Initial HTTP connection works, but WebSocket upgrade fails.

**Cause:** Usually a service configuration issue, not Caddy.

**Diagnostic:**
```bash
# Check Caddy logs for WebSocket upgrade
docker logs aura-caddy | grep -i websocket

# Test WebSocket endpoint directly
wscat -c wss://chat.aura/socket.io/
```

**Solution:**
Caddy handles WebSocket automatically. Check your application:
```yaml
# No special config needed for WebSocket
labels:
  caddy: chat.aura
  caddy.reverse_proxy: "{{upstreams 3000}}"
  caddy.tls: "/certs/domains/chat/cert.pem /certs/domains/chat/key.pem"
  # That's it! WebSocket works automatically
```

---

## Container Issues

### "Aura proxy is not running"

**Symptom:**
```bash
$ aura status
❌ Aura proxy is not running
```

**Solution:**
```bash
# Start Aura
aura start

# If start fails, check Docker daemon
docker ps

# Check for errors
docker compose -f ~/.aura/docker-compose.yml up -d
```

---

### Containers keep restarting

**Symptom:** `docker ps` shows containers constantly restarting.

**Diagnostic:**
```bash
# Check container logs
docker logs aura-caddy
docker logs aura-coredns
docker logs aura-whoami

# Check for configuration errors
docker inspect aura-caddy
```

**Common causes:**
1. Port conflict (80/443 already in use)
2. Invalid Corefile syntax
3. Missing certificate files
4. Docker memory/CPU limits

---

### Can't access containers from dev directory

**Symptom:** Containers run but use old certificates or wrong configuration.

**Cause:** Running `docker compose` from development directory instead of `~/.aura`.

**Solution:**
```bash
# Always use aura commands
aura start
aura stop

# NOT:
# cd /path/to/repo && docker compose up

# If you must use docker compose directly:
cd ~/.aura
docker compose up -d
```

---

## Performance Issues

### Slow HTTPS responses

**Symptom:** Requests take several seconds.

**Diagnostic:**
```bash
# Test direct connection (bypass Caddy)
time curl http://$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' myapp):80

# Test through Caddy
time curl https://myapp.aura

# Check Caddy CPU usage
docker stats aura-caddy
```

**Solutions:**
```yaml
# Enable compression
labels:
  caddy.encode: gzip

# Increase Docker resources (Docker Desktop Settings)
# Memory: 4GB+
# CPUs: 2+
```

---

### High memory usage

**Symptom:** Docker using excessive memory.

**Solution:**
```bash
# Check which container is using memory
docker stats

# Restart containers
aura stop
aura start

# Clear Docker cache
docker system prune -a
```

---

## Platform-Specific Issues

### macOS: "Operation not permitted" on /etc/resolver

**Symptom:**
```
mkdir: /etc/resolver: Operation not permitted
```

**Cause:** macOS System Integrity Protection.

**Solution:**
```bash
# This is normal - the directory should already exist
ls -la /etc/resolver

# If it doesn't exist, create it with sudo
sudo mkdir -p /etc/resolver
sudo bash setup-resolver.sh
```

---

### macOS: Loopback disappears after reboot

**Symptom:** `ifconfig lo0` doesn't show 127.0.0.2 after restart.

**Solution:**
```bash
# Check LaunchDaemon is loaded
sudo launchctl list | grep aura

# If not loaded, reload it
sudo launchctl load /Library/LaunchDaemons/com.aura.loopback.plist

# Verify loopback exists
ifconfig lo0 | grep 127.0.0.2
```

---

### Linux: SELinux blocking access

**Symptom:** Permission denied errors on Fedora/RHEL with SELinux.

**Solution:**
```bash
# Temporarily disable SELinux
sudo setenforce 0

# Or configure SELinux policy (advanced)
# Check audit logs
sudo ausearch -m avc -ts recent
```

---

## Advanced Debugging

### Enable Caddy debug logging

```bash
# Stop Aura
aura stop

# Edit docker-compose.yml
vim ~/.aura/docker-compose.yml

# Add to caddy service:
environment:
  - CADDY_DEBUG=1

# Restart
aura start

# View debug logs
docker logs -f aura-caddy
```

---

### Test CoreDNS configuration

```bash
# Get current Corefile
docker exec aura-coredns cat /config/Corefile

# Test DNS query manually
docker exec aura-coredns nslookup test.aura localhost

# Check CoreDNS metrics
curl http://127.0.0.2:9153/metrics
```

---

### Inspect Caddy configuration

```bash
# See generated Caddyfile
docker logs aura-caddy 2>&1 | grep -A 50 "New Caddyfile" | tail -50

# Get current config via API
docker exec aura-caddy wget -qO- http://localhost:2019/config/

# Test config syntax
docker exec aura-caddy caddy validate --config /etc/caddy/Caddyfile
```

---

### Network debugging

```bash
# Check network exists
docker network ls | grep aura

# Inspect network
docker network inspect aura-proxy

# See all containers on network
docker network inspect aura-proxy -f '{{range .Containers}}{{.Name}} {{end}}'

# Test connectivity between containers
docker exec aura-caddy ping myapp
docker exec myapp ping aura-caddy
```

---

## Getting Help

If you're still stuck after trying these solutions:

1. **Check Logs:**
   ```bash
   aura logs > caddy-logs.txt
   docker logs aura-coredns > coredns-logs.txt
   docker logs aura-whoami > whoami-logs.txt
   ```

2. **Gather System Info:**
   ```bash
   # Create diagnostic report
   {
     echo "=== System Info ==="
     uname -a
     docker version
     docker compose version

     echo -e "\n=== Aura Status ==="
     aura status

     echo -e "\n=== DNS Test ==="
     dig whoami.aura

     echo -e "\n=== Network ==="
     docker network inspect aura-proxy

     echo -e "\n=== Containers ==="
     docker ps -a --filter name=aura-
   } > aura-diagnostic.txt
   ```

3. **Open an Issue:**
   - Go to: https://github.com/ivannovak/aura/issues
   - Include diagnostic output
   - Describe what you were trying to do
   - Include error messages

---

## Common Mistakes

### ❌ Forgot to generate certificate
```bash
# This will fail:
docker compose up -d myapp

# Do this first:
aura cert myapp
docker compose up -d myapp
```

### ❌ Wrong certificate path
```yaml
# Wrong - includes .aura in path
caddy.tls: "/certs/domains/myapp.aura/cert.pem /certs/domains/myapp.aura/key.pem"

# Correct - no .aura in path
caddy.tls: "/certs/domains/myapp/cert.pem /certs/domains/myapp/key.pem"
```

### ❌ Not on aura-proxy network
```yaml
# This won't work - no network specified
services:
  myapp:
    image: nginx

# This works
services:
  myapp:
    image: nginx
    networks:
      - aura-proxy

networks:
  aura-proxy:
    external: true
```

### ❌ Wrong port in reverse_proxy
```yaml
# Wrong - this is the host port
caddy.reverse_proxy: "{{upstreams 8080}}"  # If container exposes 80

# Correct - use the container's internal port
caddy.reverse_proxy: "{{upstreams 80}}"  # Port INSIDE the container
```

---

## Completely Reset Aura

If all else fails, start fresh:

```bash
# 1. Uninstall completely
aura uninstall

# 2. Remove CLI
sudo rm /usr/local/bin/aura

# 3. Clean Docker
docker system prune -a --volumes

# 4. Remove any leftover system files
# macOS:
sudo rm /etc/resolver/aura
sudo rm /Library/LaunchDaemons/com.aura.loopback.plist

# Linux:
sudo rm /etc/systemd/resolved.conf.d/aura.conf
sudo systemctl disable --now aura-loopback

# 5. Reinstall from scratch
cd /path/to/aura
make install
aura install
aura start
```

---

## Next Steps

- [Installation guide →](INSTALL.md)
- [Service examples →](EXAMPLES.md)
- [Advanced configuration →](ADVANCED.md)
