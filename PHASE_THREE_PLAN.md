# Phase 3 — Bytecode Compiler and VM: Detailed Implementation Plan

Version: 0.1-draft
Parent: `IMPLEMENTATION_PLAN.md` Phase 3
Goal: Reduce memory footprint and improve execution speed for MCU targets

---

## Overview

This document breaks Phase 3 into discrete, independently-deliverable tasks. Each task has clear inputs, outputs, and acceptance criteria.

**Dependencies:** Phase 1 (lexer/parser) and Phase 2 (tree-walking interpreter) must be complete.

---

## Task Summary

| Task | Deliverable | Dependencies |
|------|-------------|--------------|
| 3.1 | `pkg/memory/allocator.go` | None |
| 3.2 | Refactor `pkg/eval/builtins.go` | None |
| 3.3 | `pkg/bytecode/opcode.go` | None |
| 3.4 | `pkg/bytecode/chunk.go` | 3.3 |
| 3.5 | `pkg/bytecode/compiler.go` | 3.3, 3.4 |
| 3.6 | `pkg/bytecode/vm.go` | 3.2, 3.3, 3.4 |
| 3.7 | `pkg/bytecode/vm_test.go` | 3.5, 3.6 |
| 3.8 | Update `target/esp32s3/main.go` | 3.6 |
| 3.9 | Final validation | All above |

---

## Task 3.1 — Reference-Counted Allocator

**Goal:** Implement a simple reference-counted memory allocator for picoceci objects.

### Deliverable

`pkg/memory/allocator.go`

### Design

```go
// pkg/memory/allocator.go
package memory

import (
    "sync/atomic"
    "github.com/kristofer/picoceci/pkg/object"
)

// Retain increments the reference count of an object.
func Retain(o *object.Object)

// Release decrements the reference count. If it reaches zero,
// the object's resources are released (slots cleared, array items released).
func Release(o *object.Object)

// RefCount returns the current reference count.
func RefCount(o *object.Object) int32

// AllocObject creates a new object with refcount=1.
func AllocObject(kind object.Kind) *object.Object
```

### Implementation Notes

- Use `atomic.AddInt32` for thread-safe reference counting
- `object.Object` already has `RefCount int32` field
- When refcount hits zero:
  - For arrays: call Release on each item
  - For objects with slots: call Release on each slot value
  - For blocks with captured env: release captured variables
- Do NOT implement cycle collection yet (defer to later iteration)

### Acceptance Criteria

- [ ] `go build ./pkg/memory/...` passes
- [ ] Unit tests verify retain/release increment/decrement correctly
- [ ] Test that releasing an object with refcount=1 triggers cleanup
- [ ] Test that nested objects (array of arrays) release recursively

### Files to Create/Modify

- Create: `pkg/memory/allocator.go`
- Create: `pkg/memory/allocator_test.go`
- Modify: Remove `pkg/memory/doc.go` placeholder content

---

## Task 3.2 — Refactor builtins.go (Export Functions)

**Goal:** Export key functions from `pkg/eval/builtins.go` so the bytecode VM can reuse primitive dispatch logic.

### Current State

The following are unexported:
- `builtinDispatch(...)` — dispatches messages to primitive types
- `registerBuiltins(env *Env)` — populates global environment

### Required Exports

| Current | New Export | Purpose |
|---------|------------|---------|
| `builtinDispatch` | `BuiltinDispatch` | VM calls this for primitive type dispatch |
| `registerBuiltins` | `InitialGlobals` | Returns map of global name → object |
| (new) | `BlockCaller` | Interface for VM to call blocks |

### Design

```go
// pkg/eval/builtins.go

// BuiltinDispatch handles message sends to primitive types (int, float, bool, string, etc.)
// Returns (result, true) if handled, or (nil, false) if the receiver doesn't understand the selector.
func BuiltinDispatch(caller BlockCaller, recv *object.Object, sel string, args []*object.Object, pos ast.Pos) (*object.Object, bool, error)

// InitialGlobals returns a map of global names to their initial values.
// This includes: nil, true, false, Console, Transcript, Array
func InitialGlobals() map[string]*object.Object

// BlockCaller is an interface that allows builtins to invoke blocks.
// Both the tree-walking interpreter and bytecode VM implement this.
type BlockCaller interface {
    CallBlock(blk *object.Object, args []*object.Object) (*object.Object, error)
}
```

### Implementation Notes

- Rename `builtinDispatch` → `BuiltinDispatch`
- Change signature to accept `BlockCaller` interface instead of `*Interpreter`
- Make `*Interpreter` implement `BlockCaller` (it already has `CallBlock`)
- Extract global object creation from `registerBuiltins` into `InitialGlobals()`
- Keep type-specific dispatch functions (`intDispatch`, etc.) unexported

### Acceptance Criteria

- [ ] `BuiltinDispatch` is exported and callable from outside eval package
- [ ] `InitialGlobals` returns complete map of global objects
- [ ] `BlockCaller` interface is defined and `*Interpreter` implements it
- [ ] All existing eval tests pass unchanged
- [ ] `go build ./pkg/eval/...` passes

### Files to Modify

- Modify: `pkg/eval/builtins.go`
- Modify: `pkg/eval/eval.go` (implement BlockCaller, use InitialGlobals)
- Possibly create: `pkg/eval/interfaces.go` (for BlockCaller interface)

---

## Task 3.3 — Instruction Set Constants

**Goal:** Define the bytecode instruction set as constants.

### Deliverable

`pkg/bytecode/opcode.go`

### Design

```go
// pkg/bytecode/opcode.go
package bytecode

// OpCode represents a single bytecode instruction.
type OpCode uint8

const (
    // Stack manipulation
    OpPop         OpCode = iota // discard TOS
    OpDup                       // duplicate TOS

    // Push constants
    OpPushNil                   // push nil
    OpPushTrue                  // push true
    OpPushFalse                 // push false
    OpPushSelf                  // push self
    OpPushInt                   // push int32 immediate (next 4 bytes)
    OpPushConst                 // push constant from pool (next 2 bytes = index)

    // Variables
    OpPushLocal                 // push local[arg] (next 1 byte = slot)
    OpStoreLocal                // TOS → local[arg], pop
    OpPushUpvalue               // push upvalue[arg] (next 1 byte = index)
    OpStoreUpvalue              // TOS → upvalue[arg], pop
    OpPushInst                  // push self.slots[arg] (next 2 bytes = name index)
    OpStoreInst                 // TOS → self.slots[arg], pop
    OpPushGlobal                // push global[arg] (next 2 bytes = name index)
    OpStoreGlobal               // TOS → global[arg], pop

    // Message sends
    OpSend                      // send message (next: 2 bytes selector idx, 1 byte argc)
    OpSuperSend                 // super send (next: 2 bytes selector idx, 1 byte argc)

    // Blocks and closures
    OpClosure                   // create closure from CompiledBlock (next 2 bytes = block index)

    // Control flow
    OpJump                      // unconditional jump (next 2 bytes = signed offset)
    OpJumpIfFalse               // pop, jump if false (next 2 bytes = signed offset)
    OpJumpIfTrue                // pop, jump if true (next 2 bytes = signed offset)

    // Return
    OpReturn                    // return TOS
    OpReturnSelf                // return self
    OpBlockReturn               // non-local return from block
)

// String returns the human-readable name of an opcode.
func (op OpCode) String() string

// OperandWidths returns the byte widths of operands for this opcode.
// E.g., OpPushInt returns [4] (one 4-byte operand).
func (op OpCode) OperandWidths() []int
```

### Acceptance Criteria

- [ ] All opcodes from IMPLEMENTATION_PLAN.md are defined
- [ ] `String()` returns readable names for disassembly
- [ ] `OperandWidths()` correctly describes each opcode's operands
- [ ] `go build ./pkg/bytecode/...` passes

### Files to Create

- Create: `pkg/bytecode/opcode.go`
- Create: `pkg/bytecode/opcode_test.go`

---

## Task 3.4 — Chunk and Compiled Block Types

**Goal:** Define data structures for compiled bytecode.

### Deliverable

`pkg/bytecode/chunk.go`

### Design

```go
// pkg/bytecode/chunk.go
package bytecode

import "github.com/kristofer/picoceci/pkg/object"

// Chunk holds compiled bytecode and associated constant pool.
type Chunk struct {
    Code      []byte           // bytecode instructions
    Constants []*object.Object // constant pool (strings, floats, symbols)
    Lines     []int            // source line for each instruction (for errors)
}

// Write appends a byte to the chunk.
func (c *Chunk) Write(b byte, line int)

// WriteOp appends an opcode to the chunk.
func (c *Chunk) WriteOp(op OpCode, line int)

// AddConstant adds a constant to the pool, returns its index.
func (c *Chunk) AddConstant(val *object.Object) int

// Disassemble returns human-readable bytecode listing.
func (c *Chunk) Disassemble(name string) string

// CompiledBlock represents a compiled block/closure template.
type CompiledBlock struct {
    Arity      int       // number of parameters
    LocalCount int       // number of local variables (including params)
    Upvalues   []Upvalue // captured variable descriptors
    Chunk      *Chunk    // the bytecode
    Name       string    // for debugging (e.g., "block in Counter>>increment")
}

// Upvalue describes a captured variable from an enclosing scope.
type Upvalue struct {
    Index   uint8 // slot index in parent's locals or upvalues
    IsLocal bool  // true if captured from immediate parent's locals
}

// CompiledMethod represents a compiled method.
type CompiledMethod struct {
    Selector   string         // e.g., "increment" or "at:put:"
    Block      *CompiledBlock // the method body
}

// CompiledObject represents a compiled object template.
type CompiledObject struct {
    Name       string                      // object name
    SlotNames  []string                    // instance variable names
    Methods    map[string]*CompiledMethod  // compiled methods
    Composes   []string                    // composed object names
}
```

### Acceptance Criteria

- [ ] Chunk can store bytecode and constants
- [ ] CompiledBlock captures closure metadata
- [ ] Disassemble produces readable output
- [ ] `go build ./pkg/bytecode/...` passes

### Files to Create

- Create: `pkg/bytecode/chunk.go`
- Create: `pkg/bytecode/chunk_test.go`

---

## Task 3.5 — AST to Bytecode Compiler

**Goal:** Compile parsed AST into bytecode.

### Deliverable

`pkg/bytecode/compiler.go`

### Design

```go
// pkg/bytecode/compiler.go
package bytecode

import (
    "github.com/kristofer/picoceci/pkg/ast"
    "github.com/kristofer/picoceci/pkg/object"
)

// Compiler compiles AST nodes to bytecode.
type Compiler struct {
    // internal state: current chunk, scope stack, etc.
}

// NewCompiler creates a new compiler.
func NewCompiler() *Compiler

// Compile compiles a program (list of statements) into a Chunk.
func (c *Compiler) Compile(nodes []ast.Node) (*Chunk, error)

// CompileBlock compiles a block literal into a CompiledBlock.
func (c *Compiler) CompileBlock(blk *ast.Block) (*CompiledBlock, error)

// CompileObject compiles an object declaration into a CompiledObject.
func (c *Compiler) CompileObject(decl *ast.ObjectDecl) (*CompiledObject, error)
```

### Compilation Strategy

**Expressions:**
- `IntLit` → `OpPushInt` (if small) or `OpPushConst`
- `FloatLit`, `StringLit`, `SymbolLit` → `OpPushConst`
- `NilLit` → `OpPushNil`
- `BoolLit` → `OpPushTrue` / `OpPushFalse`
- `Ident` → resolve to local/upvalue/inst/global, emit appropriate push

**Messages:**
- `UnaryMsg` → compile receiver, `OpSend` with selector, argc=0
- `BinaryMsg` → compile receiver, compile arg, `OpSend` with selector, argc=1
- `KeywordMsg` → compile receiver, compile args, `OpSend` with selector, argc=N

**Blocks:**
- Create nested Compiler for block body
- Track upvalues (variables captured from enclosing scope)
- Emit `OpClosure` with index to CompiledBlock

**Control Flow:**
- `ifTrue:` → compile receiver, `OpJumpIfFalse`, compile block call, patch jump
- `whileTrue:` → loop target, compile condition, `OpJumpIfFalse` to end, compile body, `OpJump` to target

**Assignment:**
- `Assign` → compile value, emit store to local/upvalue/inst/global

**Return:**
- `Return` in method → `OpReturn`
- `^` in block → `OpBlockReturn` (non-local return)

### Scope Management

```go
type scope struct {
    locals    []string      // local variable names
    upvalues  []Upvalue     // captured variables
    enclosing *scope        // parent scope
}

func (s *scope) resolveLocal(name string) (int, bool)
func (s *scope) resolveUpvalue(name string) (int, bool)
```

### Acceptance Criteria

- [ ] Literals compile to correct push opcodes
- [ ] Variable references resolve correctly (local/upvalue/global)
- [ ] Message sends compile to OpSend with correct selector and argc
- [ ] Blocks compile to CompiledBlock with correct upvalue capture
- [ ] Object declarations compile to CompiledObject
- [ ] All examples from LANGUAGE_SPEC.md compile without error
- [ ] `go build ./pkg/bytecode/...` passes

### Files to Create

- Create: `pkg/bytecode/compiler.go`
- Create: `pkg/bytecode/scope.go` (scope management)
- Create: `pkg/bytecode/compiler_test.go`

---

## Task 3.6 — Bytecode Virtual Machine

**Goal:** Execute compiled bytecode.

### Deliverable

`pkg/bytecode/vm.go`

### Design

```go
// pkg/bytecode/vm.go
package bytecode

import (
    "github.com/kristofer/picoceci/pkg/eval"
    "github.com/kristofer/picoceci/pkg/object"
)

// VM executes compiled bytecode.
type VM struct {
    globals    map[string]*object.Object // global namespace
    stack      []*object.Object          // operand stack
    sp         int                       // stack pointer
    frames     []CallFrame               // call stack
    frameCount int                       // current frame count
}

// CallFrame represents a single activation record.
type CallFrame struct {
    closure *Closure          // the closure being executed
    ip      int               // instruction pointer within chunk
    bp      int               // base pointer (stack frame start)
}

// Closure is a runtime closure (CompiledBlock + captured upvalues).
type Closure struct {
    Block    *CompiledBlock
    Upvalues []*UpvalueRef
}

// UpvalueRef points to a captured variable (may be on stack or heap).
type UpvalueRef struct {
    Value  *object.Object // heap-allocated (closed-over) value
    Slot   int            // stack slot (if still on stack)
    Closed bool           // true if value has been moved to heap
}

// NewVM creates a new VM with default globals.
func NewVM() *VM

// Run executes the given chunk and returns the result.
func (vm *VM) Run(chunk *Chunk) (*object.Object, error)

// CallBlock invokes a block with arguments (implements eval.BlockCaller).
func (vm *VM) CallBlock(blk *object.Object, args []*object.Object) (*object.Object, error)
```

### Execution Loop

```go
func (vm *VM) run() (*object.Object, error) {
    for {
        frame := &vm.frames[vm.frameCount-1]
        op := OpCode(frame.closure.Block.Chunk.Code[frame.ip])
        frame.ip++

        switch op {
        case OpPushNil:
            vm.push(object.Nil)
        case OpPushTrue:
            vm.push(object.True)
        // ... other opcodes
        case OpSend:
            selector := vm.readConstantString()
            argc := vm.readByte()
            vm.send(selector, int(argc))
        case OpReturn:
            result := vm.pop()
            vm.frameCount--
            if vm.frameCount == 0 {
                return result, nil
            }
            vm.sp = frame.bp
            vm.push(result)
        }
    }
}
```

### Message Dispatch

1. Pop arguments and receiver from stack
2. Check if receiver is primitive → call `eval.BuiltinDispatch`
3. If not handled, look up method in receiver's object
4. If method has CompiledBlock → create CallFrame, execute
5. If method has Native func → call directly
6. If not found → MessageNotUnderstood error

### Upvalue Handling

- When creating closure: capture references to parent's stack slots
- When parent frame returns: "close" upvalues by copying to heap
- When accessing upvalue: check if closed, read from appropriate location

### Error Handling

- Track source line via Chunk.Lines
- On error, build stack trace from CallFrames
- Return `*eval.Error` with position info

### Acceptance Criteria

- [ ] VM executes simple expressions (arithmetic, comparisons)
- [ ] Local variables work correctly
- [ ] Message sends dispatch to methods
- [ ] Blocks capture upvalues correctly
- [ ] Non-local return (`^`) exits to correct frame
- [ ] All Phase 2 test programs produce identical output
- [ ] VM implements `eval.BlockCaller` interface
- [ ] `go build ./pkg/bytecode/...` passes

### Files to Create

- Create: `pkg/bytecode/vm.go`
- Create: `pkg/bytecode/value_stack.go` (stack operations)

---

## Task 3.7 — VM Tests

**Goal:** Comprehensive test suite ensuring bytecode execution matches tree-walking.

### Deliverable

`pkg/bytecode/vm_test.go`

### Test Categories

**1. Literal Tests**
```go
func TestVMLiterals(t *testing.T)
// Test: integers, floats, strings, symbols, nil, true, false
```

**2. Arithmetic Tests**
```go
func TestVMArithmetic(t *testing.T)
// Test: +, -, *, /, //, \\, comparisons
```

**3. Variable Tests**
```go
func TestVMVariables(t *testing.T)
// Test: local declaration, assignment, access
```

**4. Control Flow Tests**
```go
func TestVMControlFlow(t *testing.T)
// Test: ifTrue:, ifFalse:, ifTrue:ifFalse:, whileTrue:, timesRepeat:
```

**5. Block Tests**
```go
func TestVMBlocks(t *testing.T)
// Test: block creation, value, value:, upvalue capture
```

**6. Object Tests**
```go
func TestVMObjects(t *testing.T)
// Test: object declaration, method dispatch, instance variables, compose
```

**7. Non-Local Return Tests**
```go
func TestVMNonLocalReturn(t *testing.T)
// Test: ^ in blocks returns from enclosing method
```

**8. Equivalence Tests**
```go
func TestVMEquivalenceWithTreeWalk(t *testing.T)
// Run each .pc file in testdata/ with both interpreters, compare output
```

### Acceptance Criteria

- [ ] All test categories pass
- [ ] Equivalence tests confirm VM matches tree-walking output
- [ ] No memory leaks detected (verify with reference counting)
- [ ] `go test ./pkg/bytecode/...` passes

### Files to Create

- Create: `pkg/bytecode/vm_test.go`
- Possibly create: `pkg/bytecode/testutil_test.go` (test helpers)

---

## Task 3.8 — Update TinyGo Entry Point

**Goal:** Update ESP32-S3 entry point to use bytecode VM.

### Deliverable

Updated `target/esp32s3/main.go`

### Design

```go
//go:build tinygo

package main

import (
    "machine"
    "github.com/kristofer/picoceci/pkg/bytecode"
    "github.com/kristofer/picoceci/pkg/lexer"
    "github.com/kristofer/picoceci/pkg/parser"
)

func main() {
    uart := machine.UART0
    uart.Configure(machine.UARTConfig{BaudRate: 115200})

    // Initialize VM
    vm := bytecode.NewVM()
    compiler := bytecode.NewCompiler()

    // Simple REPL loop
    for {
        // Read line from UART
        line := readLine(uart)

        // Parse
        l := lexer.New(line)
        p := parser.New(l)
        program, err := p.Parse()
        if err != nil {
            writeLine(uart, "Parse error: "+err.Error())
            continue
        }

        // Compile
        chunk, err := compiler.Compile(program.Statements)
        if err != nil {
            writeLine(uart, "Compile error: "+err.Error())
            continue
        }

        // Execute
        result, err := vm.Run(chunk)
        if err != nil {
            writeLine(uart, "Runtime error: "+err.Error())
            continue
        }

        // Print result
        writeLine(uart, "=> "+result.PrintString())
    }
}

func readLine(uart *machine.UART) string { /* ... */ }
func writeLine(uart *machine.UART, s string) { /* ... */ }
```

### Implementation Notes

- Keep implementation minimal for now (no SD card yet — that's Phase 5)
- UART I/O should be synchronous and simple
- Focus on demonstrating bytecode VM runs on TinyGo target

### Acceptance Criteria

- [ ] `tinygo build -target=esp32-coreboard-v2 ./target/esp32s3` succeeds
- [ ] Basic REPL works over UART (tested on hardware or emulator)
- [ ] Simple expressions evaluate correctly

### Files to Modify

- Modify: `target/esp32s3/main.go`

---

## Task 3.9 — Final Validation

**Goal:** Verify all Phase 3 deliverables work together.

### Validation Steps

1. **Build Check**
   ```bash
   go build ./...
   ```

2. **Unit Tests**
   ```bash
   go test ./...
   ```

3. **Bytecode Equivalence**
   - Run all `testdata/*.pc` files with both tree-walking and bytecode VM
   - Compare output for exact match

4. **TinyGo Build**
   ```bash
   tinygo build -target=esp32-coreboard-v2 ./target/esp32s3
   ```

5. **Memory Verification**
   - Run test programs that create and discard many objects
   - Verify reference counts reach zero when expected

### Acceptance Criteria

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes with no failures
- [ ] All testdata programs produce identical output under VM
- [ ] TinyGo build succeeds
- [ ] No obvious memory leaks in reference counting

---

## Implementation Order (Recommended)

Tasks can be partially parallelized:

```
    3.1 (memory)     3.2 (builtins refactor)     3.3 (opcode)
         |                    |                       |
         v                    v                       v
                                                    3.4 (chunk)
                                                      |
                                                      v
                             +----------------------3.5 (compiler)
                             |                        |
                             v                        v
                          3.6 (vm) <-----------------+
                             |
                             v
                          3.7 (tests)
                             |
                             v
                          3.8 (tinygo)
                             |
                             v
                          3.9 (validation)
```

**Suggested order for sequential implementation:**

1. **3.3** (opcode) — no dependencies, foundational
2. **3.4** (chunk) — needs opcodes
3. **3.1** (memory) — independent, can be done in parallel
4. **3.2** (builtins refactor) — independent, can be done in parallel
5. **3.5** (compiler) — needs opcode + chunk
6. **3.6** (vm) — needs all above
7. **3.7** (tests) — needs vm working
8. **3.8** (tinygo) — needs vm working
9. **3.9** (validation) — final integration check

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Upvalue capture complexity | Start with simple cases, add tests incrementally |
| Non-local return semantics | Study Smalltalk/Self implementations |
| TinyGo limitations | Test TinyGo build early and often |
| Performance regression | Benchmark against tree-walking interpreter |
| Reference counting cycles | Accept limitation for now; cycle collector in future phase |

---

## Notes for Implementers

- Keep the VM simple initially — optimize later
- Add disassembly output for debugging
- Maintain strict equivalence with tree-walking interpreter
- Use existing test programs from `testdata/` as regression tests
- The Phase 2 interpreter should remain working (dual execution paths)

---

*End of Phase 3 Detailed Implementation Plan*
