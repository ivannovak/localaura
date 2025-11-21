# Contributing to Aura

Thank you for your interest in contributing!

## Development Setup

1. **Fork and clone:**
   ```bash
   git clone https://github.com/ivannovak/aura.git
   cd aura
   ```

2. **Install dependencies:**
   ```bash
   go mod download
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
