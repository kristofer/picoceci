SHELL := /bin/bash

.PHONY: build install clean test \
	esp32-build esp32-flash esp32-monitor esp32-run esp32-image-info \
	_check_tinygo _check_port

BINARY_NAME=picoceci
INSTALL_DIR=$(HOME)/bin
CMD_PATH=./cmd/picoceci

TINYGO ?= tinygo
PYTHON ?= python3
ESP_IDF_PATH ?= $(HOME)/esp/esp-idf
ESP_TARGET ?= ./targets/esp32s3-n16r8.json
ESP_APP_PATH ?= ./target/esp32s3
ESP_BUILD_DIR ?= ./build/esp32s3
ESP_ELF ?= $(ESP_BUILD_DIR)/picoceci.elf
ESP_BIN ?= $(ESP_BUILD_DIR)/picoceci.bin
ESP_BAUD ?= 115200
ESP_CHIP ?= esp32s3
ESP_FLASH_SIZE ?= 16MB
ESP_FLASH_FREQ ?= 80m
ESP_FLASH_MODE ?= dio
ESP_FLASH_BAUD ?= 921600
ESP_FLASH_OFFSET ?= 0x0

# Build-time WiFi credentials for target/esp32s3/main.go.
WIFI_SSID ?= picoceci-net
WIFI_PASS ?=

ifeq ($(shell uname),Darwin)
_DETECTED_PORT := $(firstword $(wildcard /dev/cu.usbmodem*))
else
_DETECTED_PORT := $(firstword $(wildcard /dev/ttyUSB* /dev/ttyACM*))
endif
PORT ?= $(_DETECTED_PORT)

IDF_EXPORT := source $(ESP_IDF_PATH)/export.sh >/dev/null 2>&1 &&

LDFLAGS_WIFI := -ldflags="-X main.wifiSSID=$(WIFI_SSID) -X main.wifiPass=$(WIFI_PASS)"

_check_tinygo:
	@command -v $(TINYGO) >/dev/null 2>&1 || \
		{ echo "error: tinygo not found"; exit 1; }

_check_port:
	@[ -n "$(PORT)" ] || \
		{ echo "error: no ESP32 USB port found; set PORT=/dev/cu.usbmodem..."; exit 1; }
	@[ -e "$(PORT)" ] || \
		{ echo "error: port $(PORT) does not exist"; exit 1; }

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)

esp32-build: _check_tinygo
	$(TINYGO) build -target=$(ESP_TARGET) $(LDFLAGS_WIFI) -o $(ESP_ELF) $(ESP_APP_PATH)
	@if [ ! -f "$(ESP_IDF_PATH)/export.sh" ]; then \
		echo "error: ESP-IDF export script not found at $(ESP_IDF_PATH)/export.sh"; \
		exit 1; \
	fi
	@$(IDF_EXPORT) command -v esptool >/dev/null 2>&1 || { echo "error: esptool not found after IDF export"; exit 1; }
	@$(IDF_EXPORT) esptool --chip $(ESP_CHIP) elf2image --flash-mode $(ESP_FLASH_MODE) --flash-freq $(ESP_FLASH_FREQ) --flash-size $(ESP_FLASH_SIZE) --dont-append-digest -o $(ESP_BIN) $(ESP_ELF)

esp32-flash: esp32-build _check_port
	@if [ ! -f "$(ESP_IDF_PATH)/export.sh" ]; then \
		echo "error: ESP-IDF export script not found at $(ESP_IDF_PATH)/export.sh"; \
		exit 1; \
	fi
	@$(IDF_EXPORT) command -v esptool >/dev/null 2>&1 || { echo "error: esptool not found after IDF export"; exit 1; }
	@$(IDF_EXPORT) esptool --chip $(ESP_CHIP) --port $(PORT) --baud $(ESP_FLASH_BAUD) write-flash --flash-mode keep --flash-freq keep --flash-size keep $(ESP_FLASH_OFFSET) $(ESP_BIN)

esp32-monitor: _check_tinygo _check_port
	@echo "Monitoring $(PORT) at $(ESP_BAUD) baud (Ctrl-] to quit)"
	@if [ -f "$(ESP_IDF_PATH)/export.sh" ]; then \
		$(IDF_EXPORT) $(PYTHON) -m serial.tools.miniterm --dtr 0 --rts 0 $(PORT) $(ESP_BAUD); \
	else \
		$(TINYGO) monitor -port=$(PORT) -baudrate=$(ESP_BAUD); \
	fi

esp32-run: esp32-flash
	@echo "Waiting for USB re-enumeration..."
	@sleep 2
	@NEW_PORT=$$(ls /dev/cu.usbmodem* /dev/ttyUSB* /dev/ttyACM* 2>/dev/null | head -1); \
	if [ -z "$$NEW_PORT" ]; then \
		echo "error: could not detect serial port after flash"; exit 1; \
	fi; \
	echo "Monitoring $$NEW_PORT at $(ESP_BAUD) baud"; \
	if [ -f "$(ESP_IDF_PATH)/export.sh" ]; then \
		$(IDF_EXPORT) $(PYTHON) -m serial.tools.miniterm --dtr 0 --rts 0 $$NEW_PORT $(ESP_BAUD); \
	else \
		$(TINYGO) monitor -port=$$NEW_PORT -baudrate=$(ESP_BAUD); \
	fi

esp32-image-info: esp32-build
	@if [ ! -f "$(ESP_IDF_PATH)/export.sh" ]; then \
		echo "error: ESP-IDF export script not found at $(ESP_IDF_PATH)/export.sh"; \
		exit 1; \
	fi
	@$(IDF_EXPORT) command -v esptool >/dev/null 2>&1 || { echo "error: esptool not found after IDF export"; exit 1; }
	@$(IDF_EXPORT) esptool --chip esp32s3 image-info $(ESP_BIN)

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/

clean:
	rm -f $(BINARY_NAME)
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)

test:
	go test ./...
