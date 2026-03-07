BINARY_NAME=bcvpn
GUI_BINARY_NAME=bcvpn-gui
GO=go
VERSION=$(shell cat VERSION)

.PHONY: all build build-gui build-cli-all build-linux build-darwin build-windows test test-functional fmt tidy clean release

all: build

build:
	$(GO) build -o $(BINARY_NAME) ./cmd/bcvpn

build-gui:
	$(GO) build -o $(GUI_BINARY_NAME) ./cmd/bcvpn-gui

build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-linux-amd64 ./cmd/bcvpn

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-darwin-amd64 ./cmd/bcvpn

build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-windows-amd64.exe ./cmd/bcvpn

build-cli-all: build-linux build-darwin build-windows

test:
	$(GO) test -short ./...

test-functional:
	$(GO) test -v -tags functional ./...

fmt:
	gofmt -w $$(rg --files -g'*.go')

tidy:
	$(GO) mod tidy

clean:
	rm -f $(BINARY_NAME) $(GUI_BINARY_NAME) \
		$(BINARY_NAME)-linux-amd64 \
		$(BINARY_NAME)-darwin-amd64 \
		$(BINARY_NAME)-windows-amd64.exe

release:
	@echo "Creating release v$(VERSION)..."
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)

