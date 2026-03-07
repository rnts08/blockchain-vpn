BINARY_NAME=bcvpn
GO=go

.PHONY: all build build-gui test fmt tidy clean

all: build

build:
	$(GO) build -o $(BINARY_NAME) ./cmd/bcvpn/

build-gui:
	$(GO) build -o bcvpn-gui ./cmd/bcvpn-gui/

test:
	$(GO) test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

tidy:
	$(GO) mod tidy

clean:
	rm -f $(BINARY_NAME) bcvpn-gui
