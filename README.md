# picoceci

**A small, high-protein language** — a Smalltalk-syntax, Go-semantics interpreted language designed for microcontrollers.

## What is picoceci?

picoceci is a message-passing interpreted language that borrows Smalltalk's elegant syntax while embracing Go's composability and interface-based polymorphism. It targets resource-constrained microcontrollers (initially the ESP32-S3-N16R8) via [TinyGo](https://tinygo.org/) with a single-runtime design centered on picoceci `Task` objects.

| Feature | Choice |
|---|---|
| Syntax | Smalltalk-inspired (messages, blocks, cascades) |
| Typing | Typed declarations required; `Any` for explicit dynamic opt-in |
| Polymorphism | Composition over inheritance — no class hierarchy |
| Runtime host | TinyGo → bare-metal ESP32-S3 |
| Storage | SD card up to 32 GB (FAT32 / littlefs) |
| Concurrency | FreeRTOS tasks, queues, semaphores (via TinyGo) |
| Device services | Built-in singleton objects (`Wifi`, `SDCard`, `LED`) |

## Quick taste

```picoceci
"Hello, World"
Console println: 'Hello, picoceci!'.

"Fibonacci using a block"
let fib: Block.
fib := [ :n |
    (n <= 1)
        ifTrue:  [ n ]
        ifFalse: [ (fib value: n - 1) + (fib value: n - 2) ]
].
Console println: (fib value: 10) printString.

"Composing objects — v3 typed slots"
object Counter {
    let count: Int.
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}

object LoggedCounter {
    compose Counter.
    inc [ super inc. Console println: 'incremented'. ^self ]
}

let c: LoggedCounter.
c := LoggedCounter new.
c inc; inc; inc.
Console println: c value printString.   "=> 3"
```

## Repository layout

```
picoceci/
├── README.md               ← you are here
├── LANGUAGE_SPEC.md        ← full language specification (v3 draft)
├── IMPLEMENTATION_PLAN.md  ← phased implementation roadmap (agent-ready)
├── docs/
│   ├── grammar.ebnf            ← formal EBNF grammar (v3)
│   ├── TYPED_VARIABLES_PLAN.md ← v2 typed-variable design and implementation plan
│   ├── V3_VARIABLE_DECLARATIONS_PLAN.md ← v3 declaration-syntax recommendation and phased migration plan
│   ├── stdlib.md               ← standard library reference
│   ├── freertos-bridge.md      ← FreeRTOS / TinyGo runtime bridge
│   └── sdcard.md               ← SD-card / filesystem API
└── go.mod                  ← Go module skeleton for the interpreter
```

## Status

🚧 **Specification phase** — the documents above define everything an agent (or human) needs to implement the interpreter and runtime.

**v3** — declaration syntax uses statement-style `let`:
- typed declarations: `let x: Int.`
- inferred declarations: `let x := expr.`

Assignments (`x := expr`) require prior declaration in scope. See [`docs/V3_VARIABLE_DECLARATIONS_PLAN.md`](docs/V3_VARIABLE_DECLARATIONS_PLAN.md) for migration details.

## Current progress snapshot

- Host supports both engines:
  - `picoceci run` and `picoceci repl` (AST interpreter)
  - `picoceci run-vm` and `picoceci repl-vm` (bytecode VM)
- VM parity for object declarations and composed objects is implemented.
- `Console` and `Transcript` now support separate output sinks.
- TinyGo target now constructs VM with sink-aware globals:
  - `Console` -> serial console
  - `Transcript` -> pluggable sink (to be wired to a native WiFi TCP session writer)
- Test baseline is green with `go test ./...`.

## v3 runtime direction

picoceci v3 removes Canal as a runtime dependency. Concurrency and service boundaries now live inside one picoceci VM using lightweight `Task` objects, channels (backed by FreeRTOS queues), and built-in singleton objects for board services (`Wifi`, `SDCard`, `LED`).

## Notes

See <https://tinygo.org/docs/tutorials/serialmonitor/>
and <https://docs.espressif.com/projects/esp-idf/en/stable/esp32s3/get-started/establish-serial-connection.html> for serial console access to the ESP32-S3.

use

```bash
tinygo flash -target=esp32s3-generic -port=/dev/cu.usbmodem111201 ./target/esp32s3 && tinygo monitor
```

to load and run.

For a TCP endpoint on port 2323 (host bridge to USB serial), use:

```bash
make esp32-run-bridge PORT=/dev/cu.usbmodem111201 WIFI_SSID=atlasnet WIFI_PASS=atlasnet
```

Then connect from another terminal:

```bash
nc 127.0.0.1 2323
```

## Runtime modes

### Mode A: Default TinyGo runtime (stable)

Build and flash with the default target:

```bash
make esp32-build
make esp32-run PORT=/dev/cu.usbmodem111201 WIFI_SSID=atlasnet WIFI_PASS=atlasnet
```

### Mode B: Host TCP bridge (recommended today for TCP :2323)

This keeps the on-device runtime unchanged and bridges USB serial to host TCP.

```bash
make esp32-run-bridge PORT=/dev/cu.usbmodem111201 WIFI_SSID=atlasnet WIFI_PASS=atlasnet
nc 127.0.0.1 2323
```

### Mode C: Experimental native ESP-IDF/LwIP bridge

```bash
make esp32-run-idf PORT=/dev/cu.usbmodem111201 WIFI_SSID=atlasnet WIFI_PASS=atlasnet
```

If bridge mode fails with undefined symbols such as `nvs_flash_*`, `esp_netif_*`, `esp_wifi_*`, `lwip_*`, or `vTaskDelay`, your TinyGo link environment is missing the required ESP-IDF/LwIP exports for this path. Use Mode B while bridge runtime library wiring is completed.

Experimental native ESP-IDF/LwIP bridge attempt (on-device WiFi TCP :2323):

```bash
make esp32-run-idf PORT=/dev/cu.usbmodem111201 WIFI_SSID=atlasnet WIFI_PASS=atlasnet
```

If your TinyGo toolchain does not export/link ESP-IDF WiFi and LwIP symbols,
this experimental target may fail at link time. In that case, continue using
the host bridge workflow above.

## License

MIT — see [LICENSE](LICENSE).
