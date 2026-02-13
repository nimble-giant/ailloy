.PHONY: build clean install test lint help plugin-generate plugin-update plugin-validate check-act ci ci-build ci-lint ci-security setup check-deps

# Variables
BINARY_NAME=ailloy
BUILD_DIR=bin
PLUGIN_DIR=ailloy
PLUGIN_AUTO_DIR=ailloy-auto
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: build

# Build the CLI binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ailloy

# Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/ailloy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(PLUGIN_AUTO_DIR)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Plugin Management
plugin-generate: build
	@echo "Generating Claude Code plugin from templates..."
	@$(BUILD_DIR)/$(BINARY_NAME) plugin generate --output $(PLUGIN_AUTO_DIR) --force
	@echo "Plugin generated: $(PLUGIN_AUTO_DIR)"

plugin-update: build
	@echo "Updating Claude Code plugin..."
	@$(BUILD_DIR)/$(BINARY_NAME) plugin update $(PLUGIN_DIR)
	@echo "Plugin updated: $(PLUGIN_DIR)"

plugin-validate:
	@echo "Validating Claude Code plugin..."
	@$(BUILD_DIR)/$(BINARY_NAME) plugin validate $(PLUGIN_DIR)

plugin-rebuild: clean plugin-generate
	@echo "Plugin rebuilt from templates"

plugin-diff: plugin-generate
	@echo "Comparing manual and generated plugins..."
	@diff -r $(PLUGIN_DIR) $(PLUGIN_AUTO_DIR) || true

# Development Environment Setup
setup:
	@if command -v flox >/dev/null 2>&1; then \
		echo "Flox detected. Activating development environment..."; \
		echo "Run: flox activate"; \
	else \
		echo "Recommended: Install Flox for a reproducible dev environment"; \
		echo ""; \
		echo "  curl -fsSL https://flox.dev/install | bash"; \
		echo ""; \
		echo "Then run: flox activate"; \
		echo ""; \
		echo "This will install all required tools: go, golangci-lint, act, gh"; \
		echo ""; \
		echo "Alternatively, install dependencies manually:"; \
		echo "  go           - https://go.dev/dl/"; \
		echo "  golangci-lint - brew install golangci-lint"; \
		echo "  act          - brew install act"; \
		echo "  gh           - brew install gh"; \
	fi

check-deps:
	@echo "Checking development dependencies..."
	@missing=""; \
	command -v go >/dev/null 2>&1 || missing="$$missing go"; \
	command -v golangci-lint >/dev/null 2>&1 || missing="$$missing golangci-lint"; \
	command -v act >/dev/null 2>&1 || missing="$$missing act"; \
	command -v gh >/dev/null 2>&1 || missing="$$missing gh"; \
	if [ -n "$$missing" ]; then \
		echo "Missing:$$missing"; \
		echo ""; \
		echo "Run 'make setup' for installation instructions"; \
		echo "Or use Flox: flox activate"; \
		exit 1; \
	else \
		echo "All dependencies installed:"; \
		echo "  go            - $$(go version | cut -d' ' -f3)"; \
		echo "  golangci-lint - $$(golangci-lint --version 2>/dev/null | head -1 | cut -d' ' -f4)"; \
		echo "  act           - $$(act --version 2>/dev/null | cut -d' ' -f3)"; \
		echo "  gh            - $$(gh --version 2>/dev/null | head -1 | cut -d' ' -f3)"; \
	fi

# CI - Run GitHub Actions locally with act
check-act:
	@command -v act >/dev/null 2>&1 || { \
		echo "Error: 'act' is not installed."; \
		echo ""; \
		echo "Option 1 - Use Flox (recommended):"; \
		echo "  flox activate"; \
		echo ""; \
		echo "Option 2 - Install act directly:"; \
		echo "  macOS:   brew install act"; \
		echo "  Linux:   curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash"; \
		echo "  Windows: choco install act-cli"; \
		echo ""; \
		echo "Run 'make setup' for more options"; \
		exit 1; \
	}

ci: check-act
	@echo "Running full CI locally with act..."
	act push

ci-build: check-act
	@echo "Running build job locally with act..."
	act push -j build

ci-lint: check-act
	@echo "Running lint job locally with act..."
	act push -j lint

ci-security: check-act
	@echo "Running security workflow locally with act..."
	act push -W .github/workflows/security.yml

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  setup           - Show dev environment setup instructions"
	@echo "  check-deps      - Verify all dependencies are installed"
	@echo "  build           - Build the CLI binary"
	@echo "  install         - Install the binary to GOPATH/bin"
	@echo "  clean           - Clean build artifacts"
	@echo "  test            - Run tests"
	@echo "  lint            - Run linter"
	@echo ""
	@echo "Plugin Management:"
	@echo "  plugin-generate - Generate Claude plugin from templates"
	@echo "  plugin-update   - Update existing Claude plugin"
	@echo "  plugin-validate - Validate Claude plugin structure"
	@echo "  plugin-rebuild  - Clean and regenerate plugin"
	@echo "  plugin-diff     - Compare manual and generated plugins"
	@echo ""
	@echo "CI (requires act or flox activate):"
	@echo "  ci              - Run full CI locally"
	@echo "  ci-build        - Run build job locally"
	@echo "  ci-lint         - Run lint job locally"
	@echo "  ci-security     - Run security workflow locally"
	@echo ""
	@echo "Quick start: flox activate && make build"