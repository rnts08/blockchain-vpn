# Agent Guidelines for BlockchainVPN

This document provides coding standards and commands for agentic tools operating in this repository.

---

## Build, Lint, Test Commands

### Quick Reference (Makefile)
- `make build` - Build CLI binary for current platform
- `make build-gui` - Build GUI binary for current platform  
- `make build-cli-all` - Cross-compile CLI for Linux, macOS, Windows
- `make test` - Run unit tests with verbose output
- `make test-functional` - Run functional tests (requires -tags)
- `make fmt` - Format all Go source files with gofmt
- `make tidy` - Sync go.mod/go.sum (go mod tidy)
- `make clean` - Remove build artifacts

### Direct Go Commands
- Build: `go build -o bcvpn ./cmd/bcvpn`
- Single package test: `go test -v ./internal/blockchain`
- Single test: `go test -v ./internal/blockchain -run TestFunctionName`
- All tests: `go test -v ./...`
- Format: `gofmt -w $(go list -f '{{.Dir}}' ./... | tr '\n' ' ')`
- Lint/Type check: `go vet ./...`
- Tidy deps: `go mod tidy`

---

## Go Code Style Guidelines

### Import Organization
Group imports in this order, each group separated by a blank line:
1. Standard library packages (alphabetized)
2. Third-party libraries (alphabetized)
3. Internal project packages (alphabetized)

Example:
```go
import (
    "context"
    "fmt"
    "net"
    "strings"
    "time"
    
    "github.com/btcsuite/btcd/btcec/v2"
    "github.com/btcsuite/btcd/rpcclient"
    
    "blockchain-vpn/internal/auth"
    "blockchain-vpn/internal/protocol"
)
```

### Formatting
- Use `gofmt` exclusively; do not manually format whitespace
- Indentation: tabs (gofmt default)
- Maximum line length: gofmt's default (no hard limit)
- Always run `make fmt` before committing

### Naming Conventions
- **Exported identifiers**: CamelCase (e.g., `VPNEndpoint`, `AnnounceService`)
- **Unexported identifiers**: lowerCamelCase or snake_case for private constants
- **Package names**: short, all lowercase, no underscores (e.g., `blockchain`, `tunnel`, `config`)
- **Interface names**: end with `-er` when describing behavior (e.g., `Validator`, `Provider`) or use nouns (e.g., `Store`, `Manager`)
- **Constants**: exported = CamelCase (`MaxConsumers`); unexported = snake_case with prefix (`defaultTimeout`)
- **Error variables**: prefix with `Err` (`ErrInvalidConfig`, `ErrPoolExhausted`)

### Types and Structs
- Prefer concrete types; use interfaces for abstraction and testing
- Embed interfaces sparingly; only when behavior is truly intrinsic
- Use pointer receivers for methods that modify the receiver or need to avoid copying
- Use value receivers for small, immutable types
- Group related fields together; add blank lines between logical groups
- Use JSON tags in backticks for config structs (`json:"field_name"`)

### Error Handling
- **Wrap errors**: `fmt.Errorf("context: %w", err)`
- **Create error variables**: for sentinel errors (`var ErrNotFound = errors.New("not found")`)
- **Return errors**: never panic for expected failure modes; panic only for unrecoverable programmer errors
- **Logging**: use `log.Printf` for informational messages; errors are returned, not logged
- **Validation**: return descriptive errors that help users fix configuration issues

### Comments
- **File comments**: use C-style `/* */` for package documentation, complete sentences, period at end
- **Line comments**: use `//` for explanations, sentence case, period at end
- **Exported declarations**: must have a comment explaining purpose, usage, and any invariants
- **Unexported declarations**: comment if non-obvious
- **TODO/FIXME**: include context and ideally an issue reference

---

## Project Structure

```
.
├── Makefile              # Build/test targets (use these!)
├── go.mod                # Go dependencies (Go 1.25.x)
├── cmd/                  # Main applications
│   ├── bcvpn/           # CLI entrypoint
│   └── bcvpn-gui/       # GUI entrypoint  
├── internal/            # Internal packages (not importable externally)
│   ├── auth/            # Authorization & cert management
│   ├── blockchain/      # RPC, provider, scanner, payment
│   ├── config/          # Configuration loading and validation
│   ├── crypto/          # Keystore & encryption
│   ├── geoip/           # GeoIP lookups
│   ├── history/         # Payment history
│   ├── nat/             # UPnP & NAT-PMP
│   ├── protocol/        # VPN protocol encoding/decoding
│   ├── tunnel/          # Core VPN logic (TUN, TLS, networking)
│   ├── util/            # Utilities
│   └── obs/             # Observability (logging, metrics)
└── docs/                # User and developer documentation
```

---

## Testing Guidelines

### Test File Naming
- Unit tests: `*_test.go` alongside the code
- Integration tests: `*_integration_test.go` (may use build tags)
- Always include example tests for primary exported functions

### Test Structure
```go
func TestFunctionName(t *testing.T) {
    t.Parallel() // if safe to run in parallel
    // ... test logic
}
```

### Functional Tests
- Use build tag `//go:build functional` at top of file
- Run with: `make test-functional` or `go test -v -tags functional ./...`

### Table-Driven Tests
- Prefer table-driven tests using `t.Run` subtests
- Provide clear failure messages with `t.Errorf` or `t.Fatalf`

---

## Additional Notes

### No Linter Configured
The project uses `gofmt` only. Consider enabling `staticcheck` for deeper analysis:
```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

### Go Version
- Requires Go 1.25.x (specified in go.mod)
- CI runs on Ubuntu with Go 1.25.x

### CI/CD
- GitHub Actions workflow at `.github/workflows/ci.yml`
- Runs: `make test` and builds all targets

### No Cursor/Copilot Rules
- No `.cursor/rules/`, `.cursorrules`, or `.github/copilot-instructions.md` exist
- Follow this AGENTS.md exclusively

### Dependencies
- Run `make tidy` after adding/removing imports to keep go.mod clean
- Do not edit go.sum manually; it's auto-generated

### Security
- Never commit secrets, private keys, or config files with passwords
- Rotate provider keys using `bcvpn rotate-provider-key` command (not in code)
- Use environment variables for sensitive data in automation

---

## Version Management

### Version Bumping Rules
- **Patch version** (e.g., 0.5.21 → 0.5.22): Use for bug fixes and small changes - bump automatically after each feature or fix
- **Minor version** (e.g., 0.5.21 → 0.6.0): Use only when explicitly told to do so
- **Major version** (e.g., 0.5.21 → 1.0.0): Use only when explicitly told to do so

### Version Update Procedure
When completing a feature or fix that should be committed:

1. **Update version in `VERSION`** (simple version file)
2. **Update version in `internal/version/version.go`**
3. **Update README.md** - find and update the version number (search for `Current version:`)
4. **Update CHANGELOG.md** - add a new section at the top with the new version and date
5. **Run `make fmt` and `make test`** to ensure code is formatted and tests pass
6. **Commit the changes** with a descriptive commit message

### Changelog Format
```markdown
## [0.5.22] - 2026-03-17

### Feature Name
- Description of the change

### Bug Fix
- Description of the fix

### Version Bump
- Bumped patch version to 0.5.22.
```

### Commit Message Format
- Use imperative mood: "Add feature" not "Added feature"
- Start with subsystem: "config:", "fix:", "docs:", "test:"
- Example: `config: add new option to config`

---

## Common Tasks

**Add a new package:**
1. Create directory under `internal/` or `cmd/`
2. Include `doc.go` with package comment if appropriate
3. Write tests alongside code
4. Run `make fmt && make test`

**Modify configuration:**
1. Update structs in `internal/config/`
2. Add validation in `internal/config/validate.go`
3. Update example config in README if needed

**Expose new CLI command:**
1. Add method to `cmd/bcvpn/main.go` command tree
2. Wire up subcommands with proper flags
3. Add unit tests for argument parsing and execution

**Add build tag:**
- Use sparingly for platform-specific or optional features
- Document the tag and its purpose at top of file
- Provide fallback implementation for the opposite case

---

## Questions?
Refer to `README.md` and `docs/` for detailed documentation. When in doubt, follow existing patterns in the codebase.
