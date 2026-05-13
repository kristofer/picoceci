.PHONY: build install clean test

BINARY_NAME=picoceci
INSTALL_DIR=$(HOME)/bin
CMD_PATH=./cmd/picoceci

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/

clean:
	rm -f $(BINARY_NAME)
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)

test:
	go test ./...
