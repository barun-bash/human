BINARY_NAME = human
BUILD_DIR = build
INSTALL_DIR = /usr/local/bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%d)
LDFLAGS = -X github.com/barun-bash/human/internal/version.Version=$(VERSION) \
          -X github.com/barun-bash/human/internal/version.CommitSHA=$(COMMIT) \
          -X github.com/barun-bash/human/internal/version.BuildDate=$(DATE)

.PHONY: build test install uninstall clean lint mcp mcp-embed

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/human/

mcp-embed:
	@mkdir -p cmd/human-mcp/embedded/examples
	cp LANGUAGE_SPEC.md cmd/human-mcp/embedded/LANGUAGE_SPEC.md
	@for f in examples/*/app.human; do \
		name=$$(basename $$(dirname "$$f")); \
		cp "$$f" "cmd/human-mcp/embedded/examples/$${name}.human"; \
	done

mcp: mcp-embed
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/human-mcp ./cmd/human-mcp/

test:
	go test ./...

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR)
	rm -rf cmd/human-mcp/embedded/

lint:
	go vet ./...
