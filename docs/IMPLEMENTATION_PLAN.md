# picoceci Implementation Plan

Version: 0.3-draft  
Audience: AI agents and human contributors implementing the picoceci interpreter  
Target: TinyGo 0.32+ · ESP32-S3-N16R8 · standalone picoceci runtime

Version note: v0.2 remained an internal working draft; v0.3 marks the published Canal→standalone runtime shift.

---

## Overview

This document breaks the picoceci implementation into discrete, independently-deliverable phases.  Each phase has clearly defined inputs, outputs, acceptance criteria, and suggested implementation notes so that an agent can pick up any phase and deliver it without needing context from other phases (except where noted).

## v3 shift assumptions and design implications

- **No Canal dependency:** picoceci now owns the MCU runtime end-to-end. No phase may require Canal services to boot, run, or accept user input.
- **Single VM, many tasks:** concurrency boundaries that were previously represented as external domains are now `Task` objects inside one VM.
- **Channels as queue-backed mailboxes:** `Channel` should map directly to FreeRTOS queues so task communication is explicit and testable.
- **Singleton device services:** `Wifi`, `SDCard`, and `LED` are global singleton objects, available to all tasks.
- **Network-first interaction path:** the ESP32-S3 target must expose a WiFi-backed TCP listener so users can interact with a live interpreter session (`nc` workflow).
- **Boot reliability first:** each MCU-facing phase must include startup stability checks and crash-loop mitigation, before adding new features.

## Recent progress (May 2026)

The following milestones have been implemented and verified with `go test ./...`:

- Host runtime now supports both execution engines side-by-side:
  - `picoceci run` / `picoceci repl` (AST tree-walking interpreter)
  - `picoceci run-vm` / `picoceci repl-vm` (bytecode VM)
- VM parity work for object semantics landed:
  - `ObjectDecl` registration now creates factory globals in the bytecode path.
  - Object methods are compiled to bytecode blocks and executed directly by the VM.
  - `new` triggers `init` in VM mode, matching tree-walker behavior.
  - Composition cases such as `LoggedCounter compose Counter` run correctly in VM mode.
- Console/Transcript output routing was split:
  - `eval.InitialGlobalsWithSinks(...)` supports separate output sinks.
  - New constructors allow explicit globals/sinks in both interpreter and VM.
  - Default behavior remains backward-compatible when sinks are not provided.
- Phase 5 complete:
  - `target/esp32s3/main.go` entry point with UART REPL and paste mode.
  - `pkg/tinygo/` — platform-independent Console I/O with TinyGo + desktop stubs.
  - `pkg/freertos/` — Task, Queue, Semaphore, Timer bridge objects with stubs.
  - `pkg/sdcard/` — SD card / filesystem objects with local-filesystem stubs.
- Phase 6 complete — WiFi ingress + remote REPL:
  - `pkg/net/` — TinyGo WiFi + TCP listener wrapper (`Session`, `Listener`, `Manager`).
  - `Wifi` singleton exposed in globals (`connectSSID:password:`, `status`, `ipAddress`,
    `listenOn:do:`, `disconnect`).
  - `Task` singleton exposed in globals (`spawn:name:`) — BlockCaller auto-wired via
    `eval.SetTaskCaller` after interpreter/VM creation.
  - `PicoceciREPL` singleton exposed in globals (`serve:`) — uses `GlobalSinks.REPLRunner`
    callback to run a REPL on a session's reader/writer.
  - `target/esp32s3/main.go` updated: placeholder Transcript replaced by WiFi TCP session
    writer; serial Console remains available for local recovery; accept loop spawns per-
    session REPL goroutines.
  - Desktop stubs fully tested; integration tests in `pkg/net/net_test.go`.
- Phase 7 partial (standard library expansion):
  - `LED` singleton added to globals — `on`, `off`, `toggle`, `blinkEvery:`, `stopBlink`.
    Platform driver injected via `GlobalSinks.LEDDriver`; no-op stub used on desktop.
    TinyGo implementation in `pkg/tinygo/led_tinygo.go` drives `machine.LED`.
  - `Timestamp` class singleton — `Timestamp now` returns a Timestamp instance capturing
    milliseconds since boot via `freertos.GetTickCount()`. Instances support `-` (yields
    Duration), `<`, `>`, `=`, `asMilliseconds`, `printString`.
  - `Duration` class singleton — `Duration ms: n` creates a Duration instance. Instances
    support `+`, `-`, `<`, `>`, `=`, `asMilliseconds`, `asSeconds`, `printString`.
  - `TaskSupervisor` singleton — `supervise:name:` restarts a block on error (stops on
    clean exit); `supervise:name:maxRestarts:` caps restarts. Shares `taskObjectData`
    with `Task`; both are wired by `eval.SetTaskCaller`.
  - All new singletons tested in `pkg/eval/builtins_sinks_test.go`.

## Bridge Runtime Phase 1 Verification Gates (BR1)

These gates define done-criteria for the Bridge Runtime MVP Phase 1 foundations.

### Mandatory gates (must pass)

1. Build isolation gates

- `make esp32-build` succeeds.
- `make esp32-build` prints target isolation success checks.
- Default target remains free of `esp32s3_idf_bridge` tag and bridge `extra-files`.

1. Host regression gates

- `go test ./pkg/net ./pkg/eval ./pkg/tinygo` succeeds.
- `go test ./...` succeeds.

1. Bridge contract gates

- `docs/freertos-bridge.md` contains Bridge ABI v1 function table.
- `docs/freertos-bridge.md` contains bridge return-code mapping table.
- `targets/esp32s3_idf_bridge.c` exported bridge symbols have explicit contract comments.

1. Runtime guardrail gates

- `target/esp32s3/main.go` logs selected runtime network mode at boot.
- WiFi/TCP startup failures explicitly preserve serial REPL recovery path.
- Accept loop retries transient errors and exits only on closed-listener conditions.

### Experimental gates (allowed to fail with known reason)

1. Native bridge link gate

- `make esp32-build-idf` may fail on unresolved ESP-IDF/LwIP symbols in current TinyGo link environment.
- Failure output must include actionable diagnostics and fallback command (`make esp32-run-bridge`).

1. Native runtime activation gate

- `make esp32-run-idf` is informational during Phase 1 and not a release blocker.

### Evidence checklist

Record command output snippets for each gate in PR notes:

1. `make esp32-build`
2. `go test ./pkg/net ./pkg/eval ./pkg/tinygo`
3. `go test ./...`
4. `make esp32-build-idf`

---

## Repository layout (target)

```
picoceci/
├── cmd/
│   └── picoceci/          Phase 1 — CLI host (desktop)
│       └── main.go
├── pkg/
│   ├── lexer/             Phase 1 — tokenizer
│   ├── ast/               Phase 1 — AST node types
│   ├── parser/            Phase 1 — recursive-descent parser
│   ├── eval/              Phase 2 — tree-walking interpreter
│   ├── object/            Phase 2 — value / object representation
│   ├── runtime/           Phase 2 — standard library (desktop)
│   ├── bytecode/          Phase 3 — bytecode compiler + VM
│   ├── memory/            Phase 3 — reference-counted allocator
│   ├── module/            Phase 4 — module loader
│   ├── tinygo/            Phase 5 — TinyGo-specific runtime glue
│   ├── freertos/          Phase 5 — FreeRTOS bridge objects
│   ├── sdcard/            Phase 5 — SD card / filesystem objects
│   └── net/               Phase 6 — WiFi/TCP listener bridge
├── target/
│   └── esp32s3/           Phase 5 — board-specific entry point
│       └── main.go
├── testdata/              All phases — .pc test programs
├── docs/
│   ├── grammar.ebnf
│   ├── stdlib.md
│   ├── freertos-bridge.md
│   └── sdcard.md
├── LANGUAGE_SPEC.md
├── IMPLEMENTATION_PLAN.md
├── go.mod
└── README.md
```

---

## Phase 0 — Repository Bootstrap ✅

**Goal:** Establish Go module, directory skeleton, and CI scaffolding.

### Tasks

- [x] Create `README.md`, `LANGUAGE_SPEC.md`, `IMPLEMENTATION_PLAN.md`
- [x] Create `docs/grammar.ebnf`, `docs/stdlib.md`, `docs/freertos-bridge.md`, `docs/sdcard.md`
- [x] Create `go.mod` with module path `github.com/kristofer/picoceci`
- [x] Create directory skeleton with placeholder `doc.go` files
- [x] Add `.github/workflows/ci.yml` — `go build ./...` and `go test ./...`
- [x] Add `.gitignore` for Go build artifacts

### Acceptance criteria

- `go build ./...` succeeds (even if no real code yet)
- `go test ./...` reports 0 failures

---

## Phase 1 — Lexer and Parser ✅

**Goal:** Produce a correct AST from picoceci source text.

### Inputs

- `LANGUAGE_SPEC.md` §2 (lexical), §4 (expressions), §14 (grammar)
- `docs/grammar.ebnf`

### Deliverables

- `pkg/lexer/` — tokenizer
- `pkg/ast/` — AST node definitions
- `pkg/parser/` — recursive-descent parser

### Lexer requirements

| Token kind | Examples |
|---|---|
| `INTEGER` | `42`, `16rFF`, `2r1010` |
| `FLOAT` | `3.14`, `1.5e-3` |
| `STRING` | `'hello'`, `'it''s'` |
| `SYMBOL` | `#hello`, `#at:put:` |
| `CHARACTER` | `$A`, `$\n` |
| `BYTEARRAY_OPEN` | `#[` |
| `ARRAY_OPEN` | `#(` |
| `IDENTIFIER` | `counter`, `Counter` |
| `KEYWORD` | `at:`, `put:`, `ifTrue:` |
| `BINARY_OP` | `+`, `<=`, `~=` |
| `SPECIAL` | `.` `;` `^` `:=` `\|` `[` `]` `(` `)` `{` `}` |
| `RESERVED` | `nil` `true` `false` `self` `super` `object` `interface` `compose` `import` |
| `EOF` | — |

### Parser requirements

Implement a recursive-descent parser following the grammar in `docs/grammar.ebnf`.

AST nodes (minimum set):

| Node | Fields |
|---|---|
| `Program` | Statements []Node |
| `VarDecl` | Names []string |
| `Assign` | Name string, Value Node |
| `Return` | Value Node |
| `Cascade` | Receiver Node, Messages []Node |
| `UnaryMsg` | Receiver Node, Selector string |
| `BinaryMsg` | Receiver Node, Op string, Arg Node |
| `KeywordMsg` | Receiver Node, Keywords []string, Args []Node |
| `Block` | Params []string, Locals []string, Body []Node |
| `IntLit` | Value int64 |
| `FloatLit` | Value float64 |
| `StringLit` | Value string |
| `SymbolLit` | Value string |
| `CharLit` | Value rune |
| `BoolLit` | Value bool |
| `NilLit` | — |
| `ArrayLit` | Elements []Node |
| `ByteArrayLit` | Bytes []byte |
| `Ident` | Name string |
| `ObjectDecl` | Name string, Composes []string, Slots []string, Methods []*MethodDecl |
| `MethodDecl` | Selector string, Params []string, Locals []string, Body []Node |
| `InterfaceDecl` | Name string, Sigs []string |
| `ImportDecl` | Path string |

### Test requirements

- Round-trip parse all examples in `LANGUAGE_SPEC.md`
- Error recovery: parser should return a meaningful error for common mistakes (missing `.`, unbalanced `[]`)
- Fuzz test the lexer with random input — must not panic

### Suggested implementation notes

- Use `text/scanner` or a hand-written lexer (preferred for control)
- Store source positions on every AST node for error messages
- Parser state: current token, peek token, error list

---

## Phase 2 — Tree-walking Interpreter (Desktop)

**Goal:** Execute picoceci programs on a desktop Go host.

### Inputs

- Phase 1 deliverables (AST)
- `LANGUAGE_SPEC.md` §3–§9

### Deliverables

- `pkg/object/` — value representation
- `pkg/eval/` — interpreter
- `pkg/runtime/` — built-in objects (desktop-flavoured)
- `cmd/picoceci/` — CLI: `picoceci run file.pc` and `picoceci repl`

### Object representation

```go
// pkg/object/object.go
type Kind uint8

const (
    KindNil Kind = iota
    KindBool
    KindSmallInt
    KindFloat
    KindChar
    KindString
    KindSymbol
    KindByteArray
    KindArray
    KindBlock
    KindObject
    KindNativeFunc
)

type Object struct {
    Kind   Kind
    // union-like fields; only the relevant ones are populated
    IVal   int64
    FVal   float64
    SVal   string
    BVal   []byte
    AVal   []*Object
    Slots  map[string]*Object
    Methods map[string]*MethodDef
    // for blocks:
    Params  []string
    Body    []ast.Node
    Env     *Env
}
```

### Environment (scope)

```go
type Env struct {
    vars   map[string]*Object
    outer  *Env
}
```

### Interpreter loop

1. `Eval(node ast.Node, env *Env) (*Object, error)`
2. Dispatch on node type.
3. For message sends, look up the method in the receiver's `Methods` map, then in composed objects.
4. Invoke the method in a new `Env` with `self` bound.

### Built-in objects for desktop

| Object | Methods |
|---|---|
| `Console` | `print:`, `println:`, `nl` |
| `Transcript` | same as Console (alias) |
| `String` | all from §3.4 |
| `Array` | all from §3.5 |
| `Integer` | all from §3.3 |
| `Float` | all from §3.3 |
| `Boolean` | `ifTrue:`, `ifFalse:`, etc. |
| `Error` | `signal:`, `messageText` |

### CLI

```
$ picoceci run hello.pc
$ picoceci repl
picoceci> 3 + 4.
=> 7
picoceci> 'hello' reversed.
=> 'olleh'
```

### Test requirements

- All examples from `LANGUAGE_SPEC.md` must produce expected output
- Error cases: `MessageNotUnderstood`, `IndexOutOfBounds` produce readable errors
- REPL: single-line expressions evaluated and printed

---

## Phase 3 — Bytecode Compiler and VM

**Goal:** Reduce memory footprint and improve execution speed for MCU targets.

### Motivation

Tree-walking requires keeping the full AST in RAM.  On an ESP32-S3-N16R8 with 8 MB PSRAM this is feasible but wasteful.  A bytecode representation is more compact and faster.

### Inputs

- Phase 1 + Phase 2 deliverables
- `LANGUAGE_SPEC.md` §11 (memory model)

### Deliverables

- `pkg/bytecode/compiler.go` — AST → bytecode
- `pkg/bytecode/vm.go` — bytecode virtual machine
- `pkg/memory/` — reference-counted object allocator

### Instruction set (suggested)

| Opcode | Operand | Effect |
|---|---|---|
| `PUSH_SELF` | — | push self onto stack |
| `PUSH_NIL` | — | push nil |
| `PUSH_TRUE` | — | push true |
| `PUSH_FALSE` | — | push false |
| `PUSH_INT` | imm32 | push tagged int |
| `PUSH_FLOAT` | const-idx | push float constant |
| `PUSH_STRING` | const-idx | push string constant |
| `PUSH_SYMBOL` | const-idx | push symbol constant |
| `PUSH_LOCAL` | slot | push local variable |
| `STORE_LOCAL` | slot | pop → local variable |
| `PUSH_INST` | slot | push instance slot |
| `STORE_INST` | slot | pop → instance slot |
| `SEND` | selector-idx, argc | message send |
| `SUPER_SEND` | selector-idx, argc | super send |
| `BLOCK` | const-idx | push block object |
| `RETURN` | — | return top of stack |
| `RETURN_SELF` | — | return self |
| `JUMP` | offset | unconditional jump |
| `JUMP_IF_FALSE` | offset | pop and jump if false |
| `POP` | — | discard top of stack |
| `DUP` | — | duplicate top of stack |

### Memory allocator

- Object header: `uint32` (ref count in high 28 bits, kind in low 4 bits)
- `Alloc(kind, slotCount)` → `*Object`
- `Retain(*Object)` / `Release(*Object)` — ref count management
- Simple cycle collector: mark-and-sweep over live object graph, triggered when free list is exhausted

### Test requirements

- All Phase 2 test programs must produce identical output when run under the VM
- Memory: no leak on programs with cycles (verify with leak checker)

### Incremental REPL VM compile note

When embedding a VM-backed REPL that compiles each input independently (for example,
the ESP32 WiFi session REPL path), keep VM block/global state stable across inputs.

Required call order per input:

1. Compile with a fresh compiler (`c := bytecode.NewCompilerWithLoader(...)`).
2. Append compiler blocks into the persistent VM and rewrite closure indices in the
  just-compiled chunk: `vm.AddBlocksAndAdjustChunk(chunk, c.GetBlocks())`.
3. Merge prior REPL globals and newly declared globals:
  `vm.AddGlobals(replGlobals)` then `vm.AddGlobals(c.GetGlobals())`.
4. Run chunk.
5. Persist globals for next input: `replGlobals = vm.Globals()`.

Do not replace VM blocks with only the latest compiler block slice unless the compiler
was seeded with prior blocks. Replacing can invalidate closure block indices captured
by earlier inputs.

---

## Phase 4 — Module System

**Goal:** Support `import` and multi-file programs.

### Inputs

- Phase 3 deliverables
- `LANGUAGE_SPEC.md` §12

### Deliverables

- `pkg/module/loader.go` — module loader and cache
- `pkg/module/resolver.go` — path resolution (built-in → SD card → absolute)

### Module lifecycle

1. `import 'Counter'` triggers loader.
2. Resolver searches: built-in library → `/sdcard/picoceci/libs/Counter.pc` → absolute path.
3. If found, source is lexed + parsed + compiled to bytecode.
4. Top-level object and interface declarations are registered in the global namespace.
5. The module's compiled bytecode is cached keyed by resolved path.

### Built-in module list

Modules always available regardless of filesystem:

- `core` (auto-imported) — `Boolean`, `Integer`, `Float`, `String`, `Symbol`, `Array`, `ByteArray`, `Error`
- `io` — `Console`, `Transcript`, `ReadStream`, `WriteStream`
- `collections` — `OrderedCollection`, `Dictionary`, `Set`, `Bag`
- `task` — `Task`, `Queue`, `Semaphore`, `Timer`, `Channel`
- `sdcard` — `File`, `Directory`, `Path`
- `wifi` — `Wifi`
- `led` — `LED`
- `gpio` — `GPIO`
- `uart` — `UART`
- `i2c` — `I2C`
- `spi` — `SPI`

### Test requirements

- `import` of a missing module raises `IOError` with descriptive message
- Circular import detected at compile time
- Module loaded only once even if imported from multiple files

---

## Phase 5 — TinyGo / MCU Target

**Goal:** Run picoceci programs on ESP32-S3-N16R8 via TinyGo.

### Inputs

- Phase 3 + Phase 4 deliverables
- `docs/freertos-bridge.md`
- `docs/sdcard.md`

### Deliverables

- `pkg/tinygo/` — TinyGo-compatible runtime shims (no `os.Stdout`, no `net`, etc.)
- `pkg/freertos/` — FreeRTOS bridge objects (`Task`, `Queue`, `Semaphore`, `Timer`)
- `pkg/sdcard/` — SD card / filesystem objects (`File`, `Directory`)
- `target/esp32s3/main.go` — entry point, mounts SD card, starts picoceci REPL on UART0

### Build constraint pattern

```go
//go:build tinygo

package tinygo

// TinyGo-specific implementations
```

```go
//go:build !tinygo

package tinygo

// Desktop stubs / test doubles
```

### FreeRTOS bridge implementation notes

- FreeRTOS C functions are called via `//export` CGo-like TinyGo declarations.
- Task spawn creates a FreeRTOS task that calls the picoceci VM with the block's bytecode.
- Task stack is allocated from a separate arena to avoid heap fragmentation.
- Queues pass serialized picoceci object references (pointer-sized tokens).

### SD card implementation notes

- Use `machine/sdcard` package from TinyGo.
- Mount FAT32 filesystem at `/sdcard/`.
- `File open: path mode: #read` → TinyGo `os.Open` equivalent.
- `File open: path mode: #write` → create or truncate.
- `File open: path mode: #append`.
- Streams: `ReadStream`, `WriteStream` wrap file handles.

### ESP32-S3 entry point

```go
//go:build tinygo

package main

import (
    "machine"
    "github.com/kristofer/picoceci/pkg/tinygo"
    "github.com/kristofer/picoceci/pkg/sdcard"
    "github.com/kristofer/picoceci/pkg/eval"
)

func main() {
    uart := machine.UART0
    uart.Configure(machine.UARTConfig{BaudRate: 115200})

    sdcard.Mount("/sdcard/")

    interp := eval.New()
    interp.RunREPL(uart)
}
```

### Memory budget (ESP32-S3-N16R8)

| Region | Size | Use |
|---|---|---|
| Internal SRAM | 512 KB | TinyGo runtime, FreeRTOS, VM stack |
| PSRAM (OCTAL) | 8 MB | picoceci heap, bytecode cache, SD buffers |
| Flash | 16 MB | TinyGo binary + built-in module bytecode |
| SD card | ≤ 32 GB | User scripts, data, large modules |

### Test requirements

- FreeRTOS bridge objects have desktop stubs that pass all unit tests
- SD card objects have a local-filesystem stub for desktop testing
- `target/esp32s3` builds with `tinygo build -target=esp32-coreboard-v2 ./target/esp32s3`

---

## Phase 6 — WiFi ingress + remote REPL

**Goal:** Make one picoceci VM reachable over WiFi TCP on ESP32-S3 without Canal.

### Inputs

- Phase 5 deliverables
- `LANGUAGE_SPEC.md` §13.2

### Deliverables

- `pkg/net/` (or equivalent target package) — TinyGo WiFi + TCP listener wrapper
- `Wifi` singleton object exposed in globals
- Session bridge that wires TCP stream to REPL input and Transcript output
- Startup config path for SSID/password/port (build-time constants first; SD-backed config later)

### Design

```picoceci
Wifi connectSSID: ssid password: pass.
Wifi listenOn: 2323 do: [ :session |
    Task spawn: [ PicoceciREPL serve: session ] name: 'tcp-session'
].
```

Implementation notes:

1. WiFi connect/retry loop runs before listener startup; failures surface as `IOError`.
2. Listener accepts one session at a time in MVP; each accepted session runs in a spawned `Task`.
3. Transcript sink is rebound per active session so `nc` users see command output.
4. Serial Console remains available for local recovery/debug.

### Test requirements

- Desktop stubs for `Wifi` object and listener API
- Integration test that starts listener, accepts a session, and echoes REPL output
- Manual board test: `nc <ip> <port>` reaches interpreter prompt after boot

---

## Phase 7 — Standard Library Expansion

**Goal:** Fill out the standard library to a level suitable for practical scripting.

### Collections

- `OrderedCollection` — resizable array
- `Dictionary` — hash map (string or symbol keys)
- `Set` — hash set
- `Bag` — multiset

### I/O Streams

- `ReadStream on: collection`
- `WriteStream on: collection`
- `ReadWriteStream`
- `TranscriptStream` — serial console with line buffering

### Date / Time ✅

- `Timestamp now` — milliseconds since boot (`xTaskGetTickCount`) — **implemented**
- `Duration` — milliseconds-based duration — **implemented**

### Math

- `Math sin:`, `cos:`, `sqrt:`, `pow:exp:`
- Fixed-point arithmetic (`FixedPoint` object) for MCUs without FPU

### v3 singleton services (required)

- `Wifi` singleton — connect, status, listen, disconnect — **implemented (Phase 6)**
- `SDCard` singleton — mount state + card health + filesystem root helpers — **implemented (Phase 5)**
- `LED` singleton — status and heartbeat control — **implemented**

### Reliability/runtime lifecycle

- Task supervision strategy (`TaskSupervisor`) for crash/restart behavior — **implemented**
- Task naming conventions for diagnostics (all system tasks must have stable names)
- Boot-phase health checks (`Wifi`, `SDCard`, interpreter) with explicit failure states

---

## Phase 8 — Developer Tooling

**Goal:** Make picoceci pleasant to develop with.

### Tools

- `picoceci repl` — interactive REPL on desktop
- `picoceci run file.pc` — run a script
- `picoceci compile file.pc -o file.pcbc` — compile to bytecode
- `picoceci disasm file.pcbc` — disassemble bytecode
- `picoceci fmt file.pc` — auto-format source
- `picoceci doc` — extract inline documentation
- VS Code extension — syntax highlighting (TextMate grammar)

---

## Cross-cutting concerns

### Logging

All interpreter-internal errors use structured logging:

```go
log.Printf("picoceci: MessageNotUnderstood: %s does not understand %s", receiver, selector)
```

On TinyGo, `log` writes to UART0.

### Testing strategy

| Layer | Tool |
|---|---|
| Lexer / parser | Go `testing` package, table-driven tests |
| Interpreter | `.pc` test programs in `testdata/`, driven by `go test` |
| MCU targets | Board-in-the-loop (BIL) tests via JTAG/serial — optional |

### CI pipeline (`.github/workflows/ci.yml`)

```yaml
- go vet ./...
- go test ./...
- tinygo build -target=esp32-coreboard-v2 ./target/esp32s3   # smoke build
```

### Versioning

picoceci uses semantic versioning.  The language version is embedded in bytecode files so older VMs refuse to load newer bytecode.

---

## Milestone summary

| Milestone | Phases | Estimated effort |
|---|---|---|
| **M1 — Parse** | 0 + 1 | 2–3 days |
| **M2 — Interpret (desktop)** | 2 | 3–5 days |
| **M3 — Bytecode VM** | 3 | 5–7 days |
| **M4 — Modules + SD** | 4 + partial 5 | 3–4 days |
| **M5 — MCU target** | 5 | 4–6 days |
| **M6 — WiFi ingress** | 6 | 3–4 days |
| **M7 — StdLib + Tooling** | 7 + 8 | ongoing |

---

*End of picoceci Implementation Plan v0.3-draft*
