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
| fib: Block |
fib := [ :n |
    (n <= 1)
        ifTrue:  [ n ]
        ifFalse: [ (fib value: n - 1) + (fib value: n - 2) ]
].
Console println: (fib value: 10) printString.

"Composing objects — v2 typed slots"
object Counter {
    | count: Int |
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}

object LoggedCounter {
    compose Counter.
    inc [ super inc. Console println: 'incremented'. ^self ]
}

| c: LoggedCounter |
c := LoggedCounter new.
c inc; inc; inc.
Console println: c value printString.   "=> 3"
```

## Repository layout

```
picoceci/
├── README.md               ← you are here
├── LANGUAGE_SPEC.md        ← full language specification (v2)
├── IMPLEMENTATION_PLAN.md  ← phased implementation roadmap (agent-ready)
├── docs/
│   ├── grammar.ebnf            ← formal EBNF grammar (v2)
│   ├── TYPED_VARIABLES_PLAN.md ← v2 typed-variable design and implementation plan
│   ├── stdlib.md               ← standard library reference
│   ├── freertos-bridge.md      ← FreeRTOS / TinyGo runtime bridge
│   └── sdcard.md               ← SD-card / filesystem API
└── go.mod                  ← Go module skeleton for the interpreter
```

## Status

🚧 **Specification phase** — the documents above define everything an agent (or human) needs to implement the interpreter and runtime.

**v2** — typed variable declarations are now required.  Every variable must carry an explicit type annotation (`| x: Int |`); bare `| x |` is a parse error.  Use `| x: Any |` to opt into dynamic typing.  See [`docs/TYPED_VARIABLES_PLAN.md`](docs/TYPED_VARIABLES_PLAN.md) for the full design rationale and implementation plan.

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
tinygo flash -target=esp32s3-generic -port=/dev/cu.usbmodem11201 ./target/esp32s3 && tinygo monitor
```

to load and run.

## License

MIT — see [LICENSE](LICENSE).
