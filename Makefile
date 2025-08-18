.PHONY: build clean install test fmt vet deps

BINARY_NAME=bashtrack
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Build the application
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)

# Build for multiple platforms
build-all:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe

# Install to system
install: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Install to user bin
install-user: build
	mkdir -p ~/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/bin/

# Format code
fmt:
	go fmt ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Run with sample data for testing
run-sample:
	./$(BUILD_DIR)/$(BINARY_NAME) record "ls -la"
	./$(BUILD_DIR)/$(BINARY_NAME) record "cd /tmp"
	./$(BUILD_DIR)/$(BINARY_NAME) record "git status"
	./$(BUILD_DIR)/$(BINARY_NAME) list

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build for current platform"
	@echo "  build-all    - Build for all platforms"
	@echo "  install      - Install to /usr/local/bin (requires sudo)"
	@echo "  install-user - Install to ~/bin"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format code"
	@echo "  deps         - Download dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  run-sample   - Test with sample data"