# Variables
BINARY_DIR := bin
EMULATOR_BINARY := $(BINARY_DIR)/emulator
CLIENT_BINARY := $(BINARY_DIR)/client
GOFILES := $(shell find . -name "*.go" -type f)

# Build flags
LDFLAGS := -w -s
BUILD_FLAGS := -ldflags="$(LDFLAGS)"

# Default target
.DEFAULT_GOAL := all

# Main targets
.PHONY: all build emulator client clean help

all: build

build: emulator client

# Create binary directory
$(BINARY_DIR):
	@mkdir -p $(BINARY_DIR)

# Build emulator
emulator: $(EMULATOR_BINARY)

$(EMULATOR_BINARY): $(BINARY_DIR) $(GOFILES)
	@echo "Building emulator..."
	@go build $(BUILD_FLAGS) -o $(EMULATOR_BINARY) ./cmd/emulator/emulator.go

# Build client
client: $(CLIENT_BINARY)

$(CLIENT_BINARY): $(BINARY_DIR) $(GOFILES)
	@echo "Building client..."
	@go build $(BUILD_FLAGS) -o $(CLIENT_BINARY) ./cmd/client/client.go

# Clean up
clean:
	@echo "Cleaning up..."
	@rm -rf $(BINARY_DIR)

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Build all binaries (default)"
	@echo "  build        - Build all binaries"
	@echo "  emulator     - Build emulator emulator"
	@echo "  client       - Build client"
	@echo "  clean        - Remove built binaries"
	@echo "  help         - Show this help message"
