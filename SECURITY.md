# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to: ivan@ivannovak.com

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
