BINARY_NAME=bcvpn
GUI_BINARY_NAME=bcvpn-gui
GO=go

.PHONY: all build build-gui build-cli-all build-linux build-darwin build-windows test fmt tidy clean

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
	$(GO) test ./...

fmt:
	gofmt -w $$(rg --files -g'*.go')

tidy:
	$(GO) mod tidy

clean:
	rm -f $(BINARY_NAME) $(GUI_BINARY_NAME) \
		$(BINARY_NAME)-linux-amd64 \
		$(BINARY_NAME)-darwin-amd64 \
		$(BINARY_NAME)-windows-amd64.exe
