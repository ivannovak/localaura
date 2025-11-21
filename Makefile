.PHONY: build install uninstall clean test version

BINARY_NAME=aura
INSTALL_PATH=/usr/local/bin

# Build variables
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(shell go version | awk '{print $$3}')
PLATFORM=$(shell go env GOOS)/$(shell go env GOARCH)

# Linker flags
LDFLAGS=-ldflags "\
	-X 'github.com/ivannovak/aura/pkg/version.Version=$(VERSION)' \
	-X 'github.com/ivannovak/aura/pkg/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/ivannovak/aura/pkg/version.BuildDate=$(BUILD_DATE)' \
	-X 'github.com/ivannovak/aura/pkg/version.GoVersion=$(GO_VERSION)' \
	-X 'github.com/ivannovak/aura/pkg/version.Platform=$(PLATFORM)' \
	-s -w"

build:
	@echo "Building Aura CLI..."
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(GIT_COMMIT)"
	@echo "  Build Date: $(BUILD_DATE)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  Platform:   $(PLATFORM)"
	@go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/aura

install: build
	@echo "Installing Aura CLI to $(INSTALL_PATH)..."
	@sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	@sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Aura CLI installed successfully!"
	@echo "Run 'aura version' to verify installation"

uninstall:
	@echo "Uninstalling Aura CLI..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Aura CLI uninstalled"

clean:
	@rm -f $(BINARY_NAME)
	@go clean

test:
	@go test ./...

version:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Platform:   $(PLATFORM)"

dev: build
	@./$(BINARY_NAME) $(ARGS)