BINARY = sigcli
INSTALL_DIR = $(HOME)/.local/bin

.PHONY: build install clean

build:
	go build -o bin/$(BINARY) ./cmd/sigcli/

install: build
	@mkdir -p $(INSTALL_DIR)
	@cp bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(INSTALL_DIR)"
	@if echo "$$PATH" | tr ':' '\n' | grep -qx "$(INSTALL_DIR)"; then \
		echo "  $(INSTALL_DIR) is in your PATH."; \
	else \
		echo "  WARNING: $(INSTALL_DIR) is not in your PATH."; \
		echo "  Add: export PATH=\"\$$PATH:$(INSTALL_DIR)\""; \
	fi

clean:
	rm -rf bin/
