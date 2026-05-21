SHELL := /bin/bash

.PHONY: build install clean test \
	esp32-build esp32-flash esp32-monitor esp32-run esp32-image-info \
	esp32-build-idf esp32-flash-idf esp32-run-idf \
	esp32-verify-target-isolation \
	esp32-tcp-bridge esp32-run-bridge esp32-clean \
	_check_tinygo _check_port

BINARY_NAME=picoceci
INSTALL_DIR=$(HOME)/bin
CMD_PATH=./cmd/picoceci

TINYGO ?= tinygo
PYTHON ?= python3
ESP_IDF_PATH ?= $(HOME)/esp/esp-idf
ESP_TARGET ?= ./targets/esp32s3-n16r8.json
ESP_IDF_BRIDGE_TARGET_TEMPLATE ?= ./targets/esp32s3-n16r8-idfbridge.json
ESP_IDF_BRIDGE_TARGET ?= $(ESP_BUILD_DIR)/esp32s3-n16r8-idfbridge.local.json
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

esp32-verify-target-isolation:
	@$(PYTHON) -c 'import json, pathlib, sys; d=pathlib.Path("$(ESP_TARGET)"); b=pathlib.Path("$(ESP_IDF_BRIDGE_TARGET_TEMPLATE)"); load=lambda p: json.loads(p.read_text()) if p.exists() else (_ for _ in ()).throw(SystemExit(f"error: missing target file: {p}")); default=load(d); bridge=load(b); dt=set(default.get("build-tags", [])); bt=set(bridge.get("build-tags", [])); extras=default.get("extra-files", []); [(_ for _ in ()).throw(SystemExit("error: default target must not include esp32s3_idf_bridge tag")) for _ in [0] if "esp32s3_idf_bridge" in dt]; [(_ for _ in ()).throw(SystemExit("error: bridge template must include esp32s3_idf_bridge tag")) for _ in [0] if "esp32s3_idf_bridge" not in bt]; [(_ for _ in ()).throw(SystemExit("error: default target must not include esp32s3_idf_bridge.c in extra-files")) for e in extras if str(e).endswith("esp32s3_idf_bridge.c")]; print("ok: target isolation checks passed")'

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)


$(ESP_IDF_BRIDGE_TARGET): $(ESP_IDF_BRIDGE_TARGET_TEMPLATE) ./targets/esp32s3_idf_bridge.c
	@mkdir -p $(ESP_BUILD_DIR)
	@$(PYTHON) -c 'import json, os, pathlib, subprocess; src=pathlib.Path("$(ESP_IDF_BRIDGE_TARGET_TEMPLATE)"); dst=pathlib.Path("$(ESP_IDF_BRIDGE_TARGET)"); data=json.loads(src.read_text()); tinygo_root=pathlib.Path(subprocess.check_output(["$(TINYGO)","env","TINYGOROOT"], text=True).strip()); tinygo_targets=tinygo_root/"targets"; bridge=pathlib.Path("./targets/esp32s3_idf_bridge.c").resolve(); rel=os.path.relpath(str(bridge), str(tinygo_targets)); extras=data.get("extra-files", []); extras=[x for x in extras if x != "targets/esp32s3_idf_bridge.c"]; extras.append(rel); data["extra-files"]=extras; dst.write_text(json.dumps(data, indent=2) + "\n")'

esp32-build: _check_tinygo esp32-verify-target-isolation
	$(TINYGO) build -target=$(ESP_TARGET) $(LDFLAGS_WIFI) -o $(ESP_ELF) $(ESP_APP_PATH)
	@if [ ! -f "$(ESP_IDF_PATH)/export.sh" ]; then \
		echo "error: ESP-IDF export script not found at $(ESP_IDF_PATH)/export.sh"; \
		exit 1; \
	fi
	@$(IDF_EXPORT) command -v esptool >/dev/null 2>&1 || { echo "error: esptool not found after IDF export"; exit 1; }
	@$(IDF_EXPORT) esptool --chip $(ESP_CHIP) elf2image --flash-mode $(ESP_FLASH_MODE) --flash-freq $(ESP_FLASH_FREQ) --flash-size $(ESP_FLASH_SIZE) --dont-append-digest -o $(ESP_BIN) $(ESP_ELF)

esp32-build-idf: _check_tinygo esp32-verify-target-isolation $(ESP_IDF_BRIDGE_TARGET)
	@set -o pipefail; \
	BUILD_LOG=$$(mktemp); \
	if ! $(TINYGO) build -target=$(ESP_IDF_BRIDGE_TARGET) $(LDFLAGS_WIFI) -o $(ESP_ELF) $(ESP_APP_PATH) 2>&1 | tee $$BUILD_LOG; then \
		if grep -Eqi 'undefined symbol: (nvs_flash_|esp_netif_|esp_event_|esp_wifi_|lwip_|vTaskDelay)' $$BUILD_LOG; then \
			echo ""; \
			echo "bridge diagnostic: missing ESP-IDF/LwIP symbols in current TinyGo link environment."; \
			echo "bridge diagnostic: this is expected until prelinked runtime library wiring (Option B) is in place."; \
			echo "bridge diagnostic: use 'make esp32-run-bridge' as working fallback for TCP :2323."; \
		fi; \
		rm -f $$BUILD_LOG; \
		exit 1; \
	fi; \
	rm -f $$BUILD_LOG
	@$(PYTHON) -c 'import json, pathlib; p=pathlib.Path("$(ESP_IDF_BRIDGE_TARGET)"); data=json.loads(p.read_text()) if p.exists() else (_ for _ in ()).throw(SystemExit("error: expected generated bridge target file is missing")); tags=set(data.get("build-tags", [])); extras=data.get("extra-files", []); [(_ for _ in ()).throw(SystemExit("error: generated bridge target is missing esp32s3_idf_bridge tag")) for _ in [0] if "esp32s3_idf_bridge" not in tags]; [(_ for _ in ()).throw(SystemExit("error: generated bridge target is missing esp32s3_idf_bridge.c extra-files entry")) for _ in [0] if not any(str(item).endswith("esp32s3_idf_bridge.c") for item in extras)]; print("ok: generated bridge target includes bridge tag and extra-file")'
	@if [ ! -f "$(ESP_IDF_PATH)/export.sh" ]; then \
		echo "error: ESP-IDF export script not found at $(ESP_IDF_PATH)/export.sh"; \
		exit 1; \
	fi
	@$(IDF_EXPORT) command -v esptool >/dev/null 2>&1 || { echo "error: esptool not found after IDF export"; exit 1; }
	@$(IDF_EXPORT) esptool --chip $(ESP_CHIP) elf2image --flash-mode $(ESP_FLASH_MODE) --flash-freq $(ESP_FLASH_FREQ) --flash-size $(ESP_FLASH_SIZE) --dont-append-digest -o $(ESP_BIN) $(ESP_ELF)

esp32-flash-idf: esp32-build-idf _check_port
	@if [ ! -f "$(ESP_IDF_PATH)/export.sh" ]; then \
		echo "error: ESP-IDF export script not found at $(ESP_IDF_PATH)/export.sh"; \
		exit 1; \
	fi
	@$(IDF_EXPORT) command -v esptool >/dev/null 2>&1 || { echo "error: esptool not found after IDF export"; exit 1; }
	@$(IDF_EXPORT) esptool --chip $(ESP_CHIP) --port $(PORT) --baud $(ESP_FLASH_BAUD) write-flash --flash-mode keep --flash-freq keep --flash-size keep $(ESP_FLASH_OFFSET) $(ESP_BIN)

esp32-run-idf: esp32-flash-idf
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

esp32-tcp-bridge: _check_port
	@echo "Starting serial->TCP bridge on 0.0.0.0:2323 via $(PORT)"
	@$(PYTHON) ./tools/serial_tcp_bridge.py --port $(PORT) --baud $(ESP_BAUD) --host 0.0.0.0 --tcp-port 2323

esp32-run-bridge: esp32-flash
	@echo "Waiting for USB re-enumeration..."
	@sleep 2
	@NEW_PORT=$$(ls /dev/cu.usbmodem* /dev/ttyUSB* /dev/ttyACM* 2>/dev/null | head -1); \
	if [ -z "$$NEW_PORT" ]; then \
		echo "error: could not detect serial port after flash"; exit 1; \
	fi; \
	echo "Starting serial->TCP bridge on 0.0.0.0:2323 via $$NEW_PORT"; \
	$(PYTHON) ./tools/serial_tcp_bridge.py --port $$NEW_PORT --baud $(ESP_BAUD) --host 0.0.0.0 --tcp-port 2323

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

clean: esp32-clean
	rm -f $(BINARY_NAME)
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)

esp32-clean:
	rm -rf $(ESP_BUILD_DIR)
	rm -f $(ESP_IDF_BRIDGE_TARGET)
	$(TINYGO) clean

test:
	go test ./...
