# Contributor Onboarding

This document describes how to contribute to BlockchainVPN.

## Getting Started

### Prerequisites

- Go 1.25.x or later
- Git
- Make
- Basic understanding of VPN concepts
- Familiarity with Bitcoin/区块链

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
```bash
git clone https://github.com/YOUR_USERNAME/blockchain-vpn.git
cd blockchain-vpn
```

### Development Setup

1. Install dependencies:
```bash
make tidy
```

2. Run tests:
```bash
make test
```

3. Build the binary:
```bash
make build
```

## Code Structure

```
blockchain-vpn/
├── cmd/
│   └── bcvpn/           # CLI application
├── internal/
│   ├── auth/            # Authentication
│   ├── blockchain/     # Blockchain interactions
│   ├── config/         # Configuration
│   ├── crypto/         # Cryptography
│   ├── geoip/         # GeoIP lookups
│   ├── history/       # Payment history
│   ├── nat/           # NAT traversal
│   ├── obs/            # Observability
│   ├── protocol/       # VPN protocol
│   ├── tunnel/         # Tunnel management
│   └── util/           # Utilities
└── docs/               # Documentation
```

## Making Changes

### Coding Standards

- Follow Go best practices
- Use `gofmt` for formatting (`make fmt`)
- Add tests for new features
- Update documentation

### Commit Guidelines

1. Make small, focused commits
2. Write descriptive commit messages
3. Reference issues in commits

Example:
```
feat(tunnel): add multi-tunnel support

- Add MultiTunnelManager struct
- Implement concurrent tunnel management
- Add tests for tunnel lifecycle
Closes #123
```

### Pull Request Process

1. Create a feature branch:
```bash
git checkout -b feature/my-feature
```

2. Make your changes

3. Run tests and formatting:
```bash
make test
make fmt
```

4. Push and create PR:
```bash
git push origin feature/my-feature
```

5. Fill out PR template with:
   - Description of changes
   - Testing performed
   - Related issues

## Testing

### Unit Tests

Run unit tests:
```bash
make test
```

### Functional Tests

Run functional tests (requires network):
```bash
make test-functional
```

### Test Structure

Tests are placed alongside source files:
```
internal/tunnel/
├── tunnel.go
└── tunnel_test.go      # Unit tests
├── tunnel_functional_test.go  # Functional tests
```

## Documentation

### Code Comments

- Comment exported functions
- Explain complex logic
- Keep comments up-to-date

### Documentation Files

Update relevant docs:
- `docs/` - User and developer docs
- `README.md` - Main documentation
- `CHANGELOG.md` - Version history

## Getting Help

- Open an issue for bugs
- Use discussions for questions
- Check existing issues before creating new ones

## Recognition

Contributors are recognized in:
- GitHub contributors page
- Release notes
- Documentation

## Code of Conduct

Be respectful and inclusive. See `CODE_OF_CONDUCT.md` for details.
