# Picoceci Architecture

This document traces three critical paths through the picoceci runtime, providing a map for understanding how source code flows from text to execution.

## Overview

The picoceci interpreter/VM architecture can be understood through three interconnected paths:

1. **Lexer → Parser**: Source text to Abstract Syntax Tree
2. **Compiler → VM**: AST to bytecode execution
3. **Globals Factory**: Runtime object initialization

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        PICOCECI ARCHITECTURE                            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   Source Code                                                           │
│       │                                                                 │
│       ▼                                                                 │
│   ┌────────┐    ┌────────┐    ┌──────────┐    ┌─────┐                  │
│   │ Lexer  │───▶│ Parser │───▶│ Compiler │───▶│ VM  │                  │
│   └────────┘    └────────┘    └──────────┘    └─────┘                  │
│       │              │              │             │                     │
│       │              │              │             │                     │
│       ▼              ▼              ▼             ▼                     │
│    Tokens          AST          Bytecode      Result                    │
│                                                   ▲                     │
│                                                   │                     │
│                              InitialGlobalsWithSinks()                  │
│                                       │                                 │
│                    ┌──────────────────┼──────────────────┐             │
│                    ▼                  ▼                  ▼             │
│               Primitives         Collections         Hardware          │
│            (nil,true,false)    (Array,Queue,...)   (LED,Wifi,...)     │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Path 1: Lexer → Parser (Source to AST)

The first path transforms source text into a structured Abstract Syntax Tree.

### Data Flow

```
Source String
     │
     ▼
┌─────────────────────────────┐
│    lexer.NewString(src)     │
│    pkg/lexer/lexer.go:43    │
└─────────────────────────────┘
     │
     │  Lexer.NextToken()
     │  (called repeatedly)
     ▼
┌─────────────────────────────┐
│      token.Token            │
│   {Type, Literal, Pos}      │
│   pkg/token/token.go        │
└─────────────────────────────┘
     │
     ▼
┌─────────────────────────────┐
│      parser.New(lexer)      │
│   pkg/parser/parser.go:55   │
└─────────────────────────────┘
     │
     │  parser.ParseProgram()
     ▼
┌─────────────────────────────┐
│         ast.Program         │
│   {Statements: []Statement} │
│   pkg/ast/ast.go            │
└─────────────────────────────┘
```

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| `Lexer` | `pkg/lexer/lexer.go` | Tokenizes source into tokens |
| `Token` | `pkg/token/token.go` | Represents lexical units (keywords, operators, literals) |
| `Parser` | `pkg/parser/parser.go` | Pratt parser building AST from token stream |
| `ast.Program` | `pkg/ast/ast.go` | Root AST node containing statements |

### Token Types

The lexer recognizes Smalltalk-inspired tokens:
- **Keywords**: Message selectors ending in `:` (e.g., `ifTrue:`, `at:put:`)
- **Literals**: Integers, floats, strings, symbols (`#symbol`)
- **Operators**: Binary operators, assignment (`:=`)
- **Delimiters**: `[ ]` for blocks, `( )` for grouping, `.` statement separator

### Parser Strategy

The parser uses a **Pratt parser** (top-down operator precedence) with:
- Prefix parse functions for literals, identifiers, blocks, arrays
- Infix parse functions for binary operators and keyword messages
- Special handling for Smalltalk-style cascades (`;`)

---

## Path 2: Compiler → VM (AST to Execution)

The second path compiles the AST to bytecode and executes it on a stack-based virtual machine.

### Data Flow

```
ast.Program
     │
     ▼
┌─────────────────────────────────┐
│   bytecode.NewCompiler()        │
│   pkg/bytecode/compiler.go:78   │
└─────────────────────────────────┘
     │
     │  compiler.Compile(statements)
     ▼
┌─────────────────────────────────┐
│        bytecode.Chunk           │
│   {Code, Constants, Lines}      │
│   pkg/bytecode/chunk.go         │
└─────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────┐
│   bytecode.NewVMWithGlobals()   │
│   pkg/bytecode/vm.go:61         │
└─────────────────────────────────┘
     │
     │  vm.Run(chunk)
     ▼
┌─────────────────────────────────┐
│       *object.Object            │
│   (evaluation result)           │
└─────────────────────────────────┘
```

### Bytecode Instructions

The VM uses a compact instruction set (defined in `pkg/bytecode/opcodes.go`):

| Category | Examples | Purpose |
|----------|----------|---------|
| Stack | `OpPush`, `OpPop`, `OpDup` | Stack manipulation |
| Constants | `OpConstant` | Load constants from pool |
| Variables | `OpGetGlobal`, `OpSetGlobal`, `OpGetLocal`, `OpSetLocal` | Variable access |
| Control | `OpJump`, `OpJumpIfFalse`, `OpReturn` | Control flow |
| Objects | `OpArray`, `OpSend` | Object construction and messaging |
| Blocks | `OpClosure`, `OpCall` | Block/closure operations |

### VM Architecture

```
┌─────────────────────────────────────────────────────┐
│                        VM                           │
├─────────────────────────────────────────────────────┤
│  stack: []*object.Object     (operand stack)        │
│  sp: int                     (stack pointer)        │
│  globals: map[string]*Object (global namespace)     │
│  blocks: []*CompiledBlock    (closure templates)    │
│  frames: []Frame             (call stack)           │
│  fp: int                     (frame pointer)        │
└─────────────────────────────────────────────────────┘
```

### Compilation Pipeline

1. **AST Walk**: Compiler visits each AST node recursively
2. **Emit Bytecode**: Generates opcodes and operands into chunk
3. **Constant Pool**: Literals stored in constants array, referenced by index
4. **Block Compilation**: Blocks compiled to `CompiledBlock`, stored separately
5. **Local Resolution**: Compiler tracks local variable scopes

---

## Path 3: Globals Factory (Runtime Initialization)

The third path shows how runtime globals are created and wired into both the interpreter and VM.

### Data Flow

```
                        GlobalSinks (dependency injection)
                                │
                ┌───────────────┼───────────────┐
                ▼               ▼               ▼
          ConsoleWriter    WifiManager      LEDDriver
          TranscriptWriter REPLRunner
                │               │               │
                └───────────────┼───────────────┘
                                ▼
                  ┌─────────────────────────────┐
                  │   InitialGlobalsWithSinks() │
                  │   pkg/eval/builtins.go:90   │
                  └─────────────────────────────┘
                                │
      ┌─────────────────────────┼─────────────────────────┐
      │                         │                         │
      ▼                         ▼                         ▼
┌──────────────┐          ┌──────────────┐          ┌──────────────┐
│ Primitives   │          │ Collections  │          │ Hardware     │
│ nil, true,   │          │ Array, Queue │          │ LED, Wifi,   │
│ false        │          │ Channel      │          │ SDCard       │
└──────────────┘          └──────────────┘          └──────────────┘
      │                         │                         │
      └─────────────────────────┼─────────────────────────┘
                                │
                  ┌─────────────┴─────────────┐
                  ▼                           ▼
           NewVMWithSinks()              New() (interpreter)
           vm.go:56                      eval.go:283
                  │                           │
                  ▼                           ▼
           NewVMWithGlobals()        NewWithGlobals()
           vm.go:61                  eval.go:287
```

### GlobalSinks Structure

The `GlobalSinks` struct (`pkg/eval/builtins.go:62-78`) enables dependency injection:

```go
type GlobalSinks struct {
    ConsoleWriter    io.Writer  // Output destination for Console
    TranscriptWriter io.Writer  // Output destination for Transcript
    WifiManager      *picnet.Manager  // WiFi network backend
    REPLRunner       REPLRunner       // Remote REPL session handler
    LEDDriver        LEDDriver        // Hardware LED control
}
```

### Built-in Globals

`InitialGlobalsWithSinks()` creates all runtime globals:

| Global | Type | Purpose |
|--------|------|---------|
| `nil`, `true`, `false` | Primitive | Language constants |
| `Console` | Object | Standard output |
| `Transcript` | Object | Logging output |
| `Array` | Class | Array constructor |
| `Queue` | Class | FIFO queue constructor |
| `Channel` | Class | Go-style channel constructor |
| `Task` | Singleton | Goroutine spawning |
| `TaskSupervisor` | Singleton | Task lifecycle management |
| `Wifi` | Singleton | WiFi station management |
| `PicoceciREPL` | Singleton | Remote REPL server |
| `LED` | Singleton | Board LED control |
| `SDCard` | Singleton | SD card filesystem |
| `File` | Class | File operations |
| `Directory` | Class | Directory operations |
| `Path` | Class | Path manipulation |
| `Timestamp` | Class | Time since boot |
| `Duration` | Class | Time span values |

### ESP32 Optimization

On embedded targets (`target/esp32s3/main.go`), globals are created **once** per REPL session via `newVMState()`, then reused across evaluations. This avoids re-allocating WiFi managers and LED drivers on every expression.

```go
// vmState holds persistent REPL state across evaluations
type vmState struct {
    globals map[string]*object.Object
    blocks  []*bytecode.CompiledBlock
    loader  *module.Loader
}
```

---

## Cross-Cutting Concerns

### Object System

All values in picoceci are `*object.Object` (`pkg/object/object.go`):

```go
type Object struct {
    Kind   Kind                    // Type discriminator
    IVal   int64                   // Integer value
    FVal   float64                 // Float value
    SVal   string                  // String/symbol value
    Slots  map[string]*Object      // Instance variables
    Env    interface{}             // Internal state (closures, etc.)
}
```

### Module System

The module loader (`pkg/module/loader.go`) supports:
- File-based imports: `import "path/to/module"`
- Circular dependency detection
- Caching of loaded modules

### Error Handling

Errors propagate through:
- Parser errors (syntax): Collected in `parser.Errors()`
- Compiler errors (semantic): Returned from `compiler.Compile()`
- Runtime errors (execution): Returned from `vm.Run()`

---

## Entry Points

### Desktop CLI

`cmd/picoceci/main.go` provides:
- REPL mode (tree-walker or VM)
- File execution
- Module loading

### ESP32 Firmware

`target/esp32s3/main.go` provides:
- Serial REPL
- WiFi REPL server
- Persistent VM state across sessions

---

## Key Design Principles

1. **Single Source of Truth**: `InitialGlobalsWithSinks()` is the sole factory for all runtime globals

2. **Dependency Injection**: Hardware dependencies (LED, WiFi, I/O) are injected via `GlobalSinks`

3. **Dual Execution Modes**: Both tree-walking interpreter and bytecode VM share the same:
   - Lexer/parser
   - Object system
   - Global initialization

4. **Embedded-First**: Design choices (compact bytecode, minimal allocations, reusable state) target ESP32 constraints

5. **Smalltalk Heritage**: Message-passing semantics, blocks as closures, cascade syntax
