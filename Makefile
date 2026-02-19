BINARY_NAME = human
BUILD_DIR = build
INSTALL_DIR = /usr/local/bin

.PHONY: build test install clean lint

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/human/

test:
	go test ./...

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR)

lint:
	go vet ./...
