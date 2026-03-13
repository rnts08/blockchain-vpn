BINARY_NAME=bcvpn
GUI_BINARY_NAME=bcvpn-gui
GO=go
VERSION_FILE=VERSION
VERSION=$(shell cat $(VERSION_FILE))

# Color output for help
BLUE:=\033[0;34m
GREEN:=\033[0;32m
YELLOW:=\033[0;33m
RESET:=\033[0m

.PHONY: all build build-gui build-cli-all build-linux build-darwin build-windows test test-functional fmt tidy clean bump-version release help

all: build

## Build binaries
build:
	$(GO) build -o $(BINARY_NAME) ./cmd/bcvpn

build-gui:
	$(GO) build -o $(GUI_BINARY_NAME) ./cmd/bcvpn-gui

build-rpc-test:
	$(GO) build -o rpc-test ./cmd/rpc-test

build-mock-rpc:
	$(GO) build -o mock-rpc ./cmd/mock-rpc

build-tui:
	$(GO) build -o bcvpn-tui ./cmd/bcvpn_tui

build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-linux-amd64 ./cmd/bcvpn

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-darwin-amd64 ./cmd/bcvpn

build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-windows-amd64.exe ./cmd/bcvpn

build-cli-all: build-linux build-darwin build-windows

## Testing and code quality
test-unit:
	$(GO) test -v ./...

test-functional:
	$(GO) test -v -tags functional ./...

test: test-unit

fmt:
	gofmt -w $$(rg --files -g'*.go')

tidy:
	$(GO) mod tidy

## Cleanup
clean:
	rm -f $(BINARY_NAME) $(GUI_BINARY_NAME) \
		$(BINARY_NAME)-linux-amd64 \
		$(BINARY_NAME)-darwin-amd64 \
		$(BINARY_NAME)-windows-amd64.exe

## Version and release management
# Bump version in source files. Use VERSION_OVERRIDE=x.y.z to set explicitly, otherwise patch is incremented.
bump-version:
	@if [ ! -f "$(VERSION_FILE)" ]; then echo "VERSION file not found"; exit 1; fi
	@if [ -n "$(VERSION_OVERRIDE)" ]; then \
		NEW_VERSION="$(VERSION_OVERRIDE)"; \
	else \
		CURRENT=$$(cat $(VERSION_FILE)); \
		major=$$(echo $$CURRENT | cut -d. -f1); \
		minor=$$(echo $$CURRENT | cut -d. -f2); \
		patch=$$(echo $$CURRENT | cut -d. -f3); \
		NEW_VERSION="$$major.$$minor.$$((patch+1))"; \
	fi
	@echo "Bumping version: $$(cat $(VERSION_FILE)) -> $$NEW_VERSION"
	@echo "$$NEW_VERSION" > $(VERSION_FILE)
	@sed -i.bak "s/Version = \".*\"/Version = \"$$NEW_VERSION\"/" internal/version/version.go
	@if grep -q "Current version:" README.md; then \
		sed -i.bak "s/Current version: \`[0-9.]*\`/Current version: \`$$NEW_VERSION\`/" README.md; \
	fi
	@rm -f internal/version/version.go.bak README.md.bak
	@git add $(VERSION_FILE) internal/version/version.go README.md
	@echo "Version files updated and staged for commit."

# Release a new version: bumps version (if not already set), runs tests, tags, and pushes.
# Usage: make release [VERSION_OVERRIDE=x.y.z] - specify version to use, or patch will be incremented from VERSION file.
release:
	@echo "=== Starting release process ==="
	# Check if working tree is clean (except staged version changes)
	@status=$$(git status --porcelain); \
	dirty=$$(echo "$$status" | grep -v '^??' | grep -v ' $(VERSION_FILE)$$' | grep -v ' internal/version/version.go$$' | grep -v ' README.md$$'); \
	if [ -n "$$dirty" ]; then \
		echo "Working tree has uncommitted changes (excluding version files). Please commit or stash them first."; \
		echo "$$dirty"; \
		exit 1; \
	fi
	# Bump version to desired version
	@if [ -n "$(VERSION_OVERRIDE)" ]; then \
		$(MAKE) bump-version VERSION_OVERRIDE=$(VERSION_OVERRIDE); \
	else \
		$(MAKE) bump-version; \
	fi
	@echo "Running tests..."
	@$(MAKE) test
	@echo "Formatting code..."
	@$(MAKE) fmt
	@echo "Committing version bump..."
	@NEW_VER=$$(cat $(VERSION_FILE)); \
		git commit -m "Release v$$NEW_VER" || true
	@echo "Creating and pushing tag v$$NEW_VER..."
	@git tag -a "v$$NEW_VER" -m "Release v$$NEW_VER"
	@git push origin HEAD
	@git push origin v$$NEW_VER
	@echo "Tag v$$NEW_VER pushed. CI will build and create a GitHub release automatically."
	@echo "Release process complete!"

## Help
help:
	@echo "$(BLUE)BlockchainVPN Makefile$(RESET)"
	@echo ""
	@echo "$(GREEN)Available targets:$(RESET)"
	@echo ""
	@echo "  $(YELLOW)all$(RESET)            - Build CLI (default)"
	@echo "  $(YELLOW)build$(RESET)          - Build CLI binary (bcvpn)"
	@echo "  $(YELLOW)build-gui$(RESET)      - Build GUI binary (bcvpn-gui)"
	@echo "  $(YELLOW)build-cli-all$(RESET)  - Build CLI for Linux, macOS, Windows"
	@echo "  $(YELLOW)test$(RESET)           - Run unit tests"
	@echo "  $(YELLOW)test-unit$(RESET)      - Run unit tests (explicit)"
	@echo "  $(YELLOW)test-functional$(RESET)- Run functional tests (requires -tags)"
	@echo "  $(YELLOW)fmt$(RESET)            - Format source code with gofmt"
	@echo "  $(YELLOW)tidy$(RESET)           - Tidy Go dependencies"
	@echo "  $(YELLOW)clean$(RESET)          - Remove build artifacts"
	@echo "  $(YELLOW)bump-version$(RESET)   - Bump version in source files (increments patch)"
	@echo "                       Use VERSION_OVERRIDE=x.y.z to set specific version"
	@echo "  $(YELLOW)release$(RESET)        - Create a new release (bump, test, tag, push)"
	@echo "                       Use VERSION_OVERRIDE=x.y.z to override version"
	@echo "  $(YELLOW)help$(RESET)           - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make release"
	@echo "  make release VERSION_OVERRIDE=1.2.3"
	@echo ""
