# picoceci

**A small, high-protein language** — a Smalltalk-syntax, Go-semantics interpreted language designed for microcontrollers.

## What is picoceci?

picoceci is a message-passing interpreted language that borrows Smalltalk's elegant syntax while embracing Go's composability and interface-based polymorphism.  It targets resource-constrained microcontrollers (initially the ESP32-S3-N16R8) via [TinyGo](https://tinygo.org/) and the [Canal](https://github.com/kristofer/Canal) capability-based microkernel.

| Feature | Choice |
|---|---|
| Syntax | Smalltalk-inspired (messages, blocks, cascades) |
| Typing | Structural / interface-based (like Go) |
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
| fib |
fib := [ :n |
    (n <= 1)
        ifTrue:  [ n ]
        ifFalse: [ (fib value: n - 1) + (fib value: n - 2) ]
].
Console println: (fib value: 10) printString.

"Composing objects"
object Counter {
    | count |
    init  [ count := 0 ]
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}

object LoggedCounter {
    compose Counter.
    inc [ super inc. Console println: 'incremented'. ^self ]
}

| c |
c := LoggedCounter new.
c inc; inc; inc.
Console println: c value printString.   "=> 3"
```

## Repository layout

```
picoceci/
├── README.md               ← you are here
├── LANGUAGE_SPEC.md        ← full language specification
├── IMPLEMENTATION_PLAN.md  ← phased implementation roadmap (agent-ready)
├── docs/
│   ├── grammar.ebnf        ← formal EBNF grammar
│   ├── stdlib.md           ← standard library reference
│   ├── freertos-bridge.md  ← FreeRTOS / TinyGo runtime bridge
│   └── sdcard.md           ← SD-card / filesystem API
└── go.mod                  ← Go module skeleton for the interpreter
```

## Status

🚧 **Specification phase** — the documents above define everything an agent (or human) needs to implement the interpreter and runtime.

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

## License

MIT — see [LICENSE](LICENSE).
