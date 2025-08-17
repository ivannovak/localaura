.PHONY: build install uninstall clean test

BINARY_NAME=aura
INSTALL_PATH=/usr/local/bin

build:
	@echo "Building Aura CLI..."
	@go build -o $(BINARY_NAME) ./cmd/aura

install: build
	@echo "Installing Aura CLI to $(INSTALL_PATH)..."
	@sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	@sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Aura CLI installed successfully!"
	@echo "Run 'aura install' to set up the proxy system"

uninstall:
	@echo "Uninstalling Aura CLI..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Aura CLI uninstalled"

clean:
	@rm -f $(BINARY_NAME)
	@go clean

test:
	@go test ./...

dev: build
	@./$(BINARY_NAME) $(ARGS)