# picoceci

**A small, high-protein language** — a Smalltalk-syntax, Go-semantics interpreted language designed for microcontrollers.

## What is picoceci?

picoceci is a message-passing interpreted language that borrows Smalltalk's elegant syntax while embracing Go's composability and interface-based polymorphism.  It targets resource-constrained microcontrollers (initially the ESP32-S3-N16R8) via [TinyGo](https://tinygo.org/) and the [Canal](https://github.com/kristofer/Canal) capability-based microkernel.

| Feature | Choice |
|---|---|
| Syntax | Smalltalk-inspired (messages, blocks, cascades) |
| Typing | Typed declarations required; `Any` for explicit dynamic opt-in |
| Polymorphism | Composition over inheritance — no class hierarchy |
| Runtime host | TinyGo → bare-metal ESP32-S3 |
| Storage | SD card up to 32 GB (FAT32 / littlefs) |
| Concurrency | FreeRTOS tasks, queues, semaphores (via TinyGo) |
| Kernel services | Canal capability-kernel IPC |

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
  - `Transcript` -> placeholder sink (ready to swap for Canal TCP writer)
- Test baseline is green with `go test ./...`.

## Relation to Canal

picoceci is intended to run as a user-space scripting layer on top of the Canal capability microkernel.  Canal provides isolated capability objects; picoceci objects map naturally onto Canal capabilities, letting scripts safely compose and invoke kernel services without raw pointer access.

## Notes

See <https://tinygo.org/docs/tutorials/serialmonitor/>
and <https://docs.espressif.com/projects/esp-idf/en/stable/esp32s3/get-started/establish-serial-connection.html> for serial console access to the ESP32-S3.

use

```bash
tinygo flash -target=esp32s3-generic -port=/dev/cu.usbmodem11201 ./target/esp32s3 && tinygo monitor
```

to load and run.

## Sat May 2: Work notes

Summary of Today's Work
Created picoceci as a [Canal](https://github.com/kristofer/Canal.git) domain:

File Purpose
Canal/canal/domains/picoceci/main.go Domain entry, REPL loop
Canal/canal/domains/picoceci/console.go Serial I/O with proper echo/newline handling
Canal/canal/domains/picoceci/filesystem.go FS stub for future capability wiring
Canal/canal/go.mod Added picoceci dependency
Canal/canal/Makefile Added picoceci, picoceci-flash, picoceci-run targets
Build commands:

cd /Users/kryounger/LocalProjects/Canal/canal
make picoceci-run   # Build, flash, and monitor
What works:

Arithmetic: 3 + 4. → 7
Console output: Console println: 'hello'.
Echo and backspace handling
Ctrl-D exits cleanly
Next steps (when you're ready):

- Wire filesystem to Canal capabilities for module loading
- Test memory-intensive expressions (now has access to PSRAM via Canal/ESP-IDF)
- Add more picoceci builtins that use Canal services

## License

MIT — see [LICENSE](LICENSE).
