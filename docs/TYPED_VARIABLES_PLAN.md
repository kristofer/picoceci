# picoceci v2 — Typed Variables: Feasibility and Scope Plan

Version: 0.2-draft  
Status: **Planning only — no implementation has been started**  
Author: picoceci contributors  
Target audience: contributors, reviewers, and evaluators

---

> *This document is a planning artifact. Its purpose is to define the scope and implementation plan for mandatory typed variable declarations in picoceci. No production code is changed by this document.*
>
> **v0.2 revision note:** Based on project direction, typed declarations are now **required** — there is no untyped fallback form. Every variable must carry an explicit type annotation; `Any` is the opt-in dynamic type. This is a breaking change to the v1 language. Since picoceci has not been publicly released, this is the right moment to enforce strict type discipline. Existing programs in this repository and in [github.com/kristofer/Canal](https://github.com/kristofer/Canal) must be updated as part of this work.

---

## 1. Problem Statement

picoceci v1 uses untyped variables. Every variable is declared with `| name |` and can hold any value at any moment:

```picoceci
| x |
x := 42.
x := 'now a string'.    "perfectly legal in v1"
x := SomeSensor new.   "also legal"
```

This flexibility is convenient for quick scripts, but it becomes a liability as picoceci programs grow:

- **Sensors** must receive raw integers or floats from hardware; storing a string by mistake silently corrupts a reading.
- **Channels** (typed message queues) are designed to carry one kind of payload; an untyped channel can queue mixed objects, causing hard-to-diagnose runtime crashes.
- **Tasks** communicating via channels have no compile-time or declaration-time guarantee that the producer and consumer agree on what flows between them.
- **Newcomers** learning to build picoceci domains (IoT nodes, spacecraft watchmen) have no declaration-level documentation of intent.

Typed variables address all of these by making the *kind of thing* a variable can hold part of the program text rather than an implicit runtime convention.

---

## 2. Goals

| Goal | Description |
|------|-------------|
| **Clarity** | Declarations communicate intent to both the runtime and the reader |
| **Safety** | Type mismatches are detected at assignment (runtime check) or at parse/compile time |
| **Defaults** | Typed declarations are automatically initialised to their zero value, eliminating a whole class of nil-reference bugs |
| **Required declarations** | Every variable must carry an explicit type annotation; bare `| x |` is a parse error. Use `| x: Any |` to opt into dynamic typing |
| **Typed channels and sensors** | Core IoT objects gain first-class type parameters |
| **Composability** | Typed slots in objects and typed method parameters enable better tooling |

---

## 3. Proposed Syntax

### 3.1 Basic typed declaration

A typed variable declaration annotates each name with its type using a colon-separated form inside the existing pipe delimiters:

```picoceci
| x: Int  y: Float  running: Bool  name: String |
```

Every variable **must** carry a type annotation — the bare `| x |` form (without `: TypeName`) is a parse error in v2. To retain full dynamism for a variable, declare it explicitly as `Any`:

```picoceci
| x: Any |      "explicitly dynamic — nil by default, accepts any value"
| x: Int |      "typed as Int, 0 by default"
```

`Any` is the escape hatch for code that genuinely needs dynamic behaviour; it must be stated explicitly rather than implied by omission.

### 3.2 Default zero values

When a typed variable is declared but not yet assigned, its value is its type's *zero value*:

| Type keyword | Zero value | Notes |
|---|---|---|
| `Int` | `0` | 63-bit signed integer |
| `Float` | `0.0` | IEEE-754 double |
| `Bool` | `false` | |
| `String` | `''` | empty string (not nil) |
| `Char` | `$\0` | NUL character; in picoceci a character literal is a `$` followed by a character or escape sequence (`$A` = 'A', `$\n` = newline, `$\0` = NUL — see LANGUAGE_SPEC §2.5) |
| `Symbol` | `#''` | empty interned symbol; picoceci allows symbols to be written as `#'<text>'`, so `#''` is the symbol formed from an empty string (see LANGUAGE_SPEC §2.5) |
| `ByteArray` | `#[]` | empty byte array |
| `Array` | `#()` | empty array |
| `Nil` | `nil` | explicit nil type, for compatibility |
| `Any` | `nil` | explicitly dynamic; accepts any value, no type check on assignment |
| `<ObjectName>` | `nil` | user-defined object type; nil until assigned |
| `<InterfaceName>` | `nil` | interface type; nil until assigned |

### 3.3 Typed object slots

Slots in an `object` declaration gain the same annotation syntax:

```picoceci
object TempSensor {
    | bus: I2C  address: Int  lastC: Float  active: Bool |

    init: aBus address: anAddr [
        bus     := aBus.
        address := anAddr.
        "lastC is already 0.0, active is already false"
    ]

    poll [
        | raw: ByteArray |
        raw   := bus readFrom: address count: 2.
        lastC := ((raw at: 1) * 256 + (raw at: 2)) / 16.0.
        ^self
    ]

    celsius    [ ^lastC ]
    fahrenheit [ ^lastC * 1.8 + 32 ]
    activate   [ active := true. ^self ]
    deactivate [ active := false. ^self ]
    isActive   [ ^active ]
}
```

Typed slots give every instance of `TempSensor` a guaranteed layout: readers of the object declaration immediately know that `lastC` is always a number, never accidentally a string.

### 3.4 Typed method parameters *(planned for v2.1)*

> **v2.0 scope note:** Typed method parameters are a v2.1 feature. The domain examples in §4 deliberately use untyped method selectors (v1 form) so they reflect only the v2.0 scope — typed local variables and typed object slots. This section describes the *design space* being considered for v2.1.

Method parameters can carry type annotations alongside the selector keyword. This is a v2.1 design concern — two syntax options are under consideration (see §7.1 for the full discussion):

**Option A — inline annotation (parameter name distinct from keyword):**

```picoceci
"Selector: kp:ki:kd:  Parameters: newKp (Float), newKi (Float), newKd (Float)"
kp: newKp: Float ki: newKi: Float kd: newKd: Float [
    kp := newKp. ki := newKi. kd := newKd.
]
```

**Option B — annotate via typed var-decl inside the method body:**

```picoceci
"Same selector, untyped in signature; annotated internally"
kp: newKp ki: newKi kd: newKd [| newKp: Float  newKi: Float  newKd: Float |
    kp := newKp. ki := newKi. kd := newKd.
]
```

The recommendation (§7.1) is to defer this to v2.1 and focus v2.0 on typed local variables and typed object slots only.

### 3.5 Typed channels and queues

Channels and queues gain a type parameter that restricts what may be sent. The recommended syntax uses double angle brackets to avoid conflict with block-literal `[` (see §7.2):

```picoceci
| tempChan:   Channel<<Float>>
  alertChan:  Channel<<String>>
  cmdQueue:   Queue<<Symbol>>
|
```

Sending the wrong type raises a `TypeError` at the point of send, not silently at the consumer.

---

## 4. Domain Examples

> **Scope note:** The examples below reflect the **v2.0 feature set** — typed local variables and typed object slots. Method selectors intentionally use the v1 untyped form (no parameter type annotations) because typed method parameters are deferred to v2.1 (§7.1). The examples still illustrate the most important typing benefits: slot layout clarity, zero-value defaults, and typed channel/queue contracts.

### 4.1 Typed spacecraft atmosphere monitor

```picoceci
"AtmosMonitor.pc — v2 typed"
import 'sensors/CO2Sensor'.
import 'sensors/HumiditySensor'.
import 'sensors/TempSensor'.

object AtmosNode {
    | co2:      CO2Sensor
      humidity: HumiditySensor
      temp:     TempSensor
      reportCh: Channel<<Array>>
      running:  Bool
    |

    init: i2c reportTo: ch [
        co2      := CO2Sensor new init: i2c address: 16r61.
        humidity := HumiditySensor new init: i2c address: 16r44.
        temp     := TempSensor new init: i2c address: 16r48.
        reportCh := ch.
        running  := false.
    ]

    start [
        running := true.
        Task spawn: [ self runLoop ] name: 'atmos-monitor'.
        ^self
    ]

    stop [ running := false. ^self ]

    runLoop [
        [ running ] whileTrue: [
            co2 poll. humidity poll. temp poll.
            reportCh <- { #co2.      co2 ppm.
                          #humidity. humidity percent.
                          #tempC.    temp celsius }.
            Task delay: 10000.
        ]
    ]
}
```

Notice what typed slots buy us here:
- `co2`, `humidity`, `temp` are statically declared to be specific sensor objects — a reader knows exactly what protocols they support.
- `reportCh` is `Channel<<Array>>` — the compiler and runtime verify every `<-` send is an `Array`.
- `running` is `Bool` — `whileTrue:` no longer needs to guess whether the loop condition could accidentally become nil.

### 4.2 Typed PID controller domain

```picoceci
"PIDLoop.pc — v2.0 typed (slots and locals; method params are v1 untyped)"
object PIDLoop {
    | kp: Float  ki: Float  kd: Float
      integral: Float  lastError: Float
      output: Float  setpoint: Float
    |

    gains: newKp ki: newKi kd: newKd [
        kp := newKp. ki := newKi. kd := newKd.
    ]

    target: sp [ setpoint := sp ]

    step: measured dt: deltaT [
        | error: Float  derivative: Float |
        error      := setpoint - measured.
        integral   := integral + error * deltaT.
        derivative := (error - lastError) / deltaT.
        output     := kp * error + ki * integral + kd * derivative.
        lastError  := error.
    ]

    output [ ^output ]
}

"Usage"
| pid: PIDLoop  cmdChan: Channel<<Float>>  sensorChan: Channel<<Float>> |
pid := PIDLoop new.
pid gains: 1.2 ki: 0.05 kd: 0.01.
pid target: 21.0.

Task spawn: [
    [ true ] whileTrue: [
        | measured: Float |
        measured := <-sensorChan.
        pid step: measured dt: 0.1.
        cmdChan <- pid output.
    ]
] name: 'pid-loop'.
```

With typed variables, `measured` is declared `Float` — if the sensor channel accidentally carries a `Symbol` due to a wiring error in the system, the runtime catches it on `<-sensorChan` before the subtraction causes a silent wrong-answer.

### 4.3 Typed LED blinker (simple MCU domain)

```picoceci
"Blinker.pc — typed"
object Blinker {
    | pin: Int  onMs: Int  offMs: Int  running: Bool |

    init: aPin on: msOn off: msOff [
        pin    := aPin.
        onMs   := msOn.
        offMs  := msOff.
    ]

    start [
        running := true.
        Task spawn: [ self blink ] name: 'blinker'.
        ^self
    ]

    blink [
        | led: GPIO |
        led := GPIO pin: pin direction: #output.
        [ running ] whileTrue: [
            led high. Task delay: onMs.
            led low.  Task delay: offMs.
        ].
        led low.
    ]

    stop [ running := false ]
}

| blinker: Blinker |
blinker := Blinker new init: 2 on: 500 off: 1500.
blinker start.
```

The `onMs` and `offMs` slots are declared `Int`, so `Task delay: onMs` is guaranteed to pass an integer — eliminating the type mismatch that would otherwise manifest as a runtime crash deep in the FreeRTOS bridge when someone accidentally assigns a float millisecond value. Even though `init:on:off:` takes untyped parameters, the assignment `onMs := msOn` is protected by the slot type: if `msOn` is not an integer, the runtime raises `TypeError` at that assignment, catching the error at the earliest possible point.

### 4.4 Typed command dispatcher

```picoceci
"Dispatcher.pc — v2.0 typed (slots and locals)"
object CommandDispatcher {
    | handlers: Dictionary  cmdQueue: Queue<<Symbol>> |

    init: q [ cmdQueue := q. handlers := Dictionary new ]

    on: cmd do: blk [
        handlers at: cmd put: blk.
    ]

    run [
        [ true ] whileTrue: [
            | cmd: Symbol  handler: Block |
            cmd     := cmdQueue receive.
            handler := handlers at: cmd ifAbsent: [ nil ].
            handler notNil ifTrue: [ handler value ].
        ]
    ]
}

| q: Queue<<Symbol>>  disp: CommandDispatcher |
q    := Queue new: 20.
disp := CommandDispatcher new init: q.
disp on: #start do: [ Console println: 'starting...' ].
disp on: #stop  do: [ Console println: 'stopping...' ].
Task spawn: [ disp run ] name: 'dispatcher'.
```

`Queue<<Symbol>>` makes the contract explicit: only symbols flow through this queue.  The dispatcher no longer needs defensive checks for unexpected payload types.

---

## 5. Impact on the Programmer's Mental Model

### 5.1 From "variables are buckets" to "variables are named contracts"

In v1, a variable is an anonymous bucket that can hold anything. The programmer must remember (or read the code carefully) to know what is actually in any given bucket. In v2, the declaration is a *contract*: `| temp: Float |` says "temp is always a Float, starting at 0.0, and the runtime enforces this."

This shifts picoceci from the mental model of Smalltalk-style dynamism toward Go's combination of static type safety with interface-based flexibility. The result is:

- **Faster comprehension** — a reader scanning a domain object immediately knows the shape of its state, without tracing all assignment paths.
- **Explicit channels** — typed channels make the data flow between tasks part of the program's declaration, not a convention buried in comments.
- **Safer composition** — when composing objects, typed slots prevent accidental slot-name collisions between incompatible types being silently accepted.

### 5.2 The two-tier type system: `Any` and typed names

v2 introduces a deliberate two-tier approach:
- **Explicitly dynamic (`Any`)** — must be written as `| x: Any |`. Retains full dynamism for exploratory code and scripts, but the intent must be stated explicitly. `Any` variables still start as `nil` and accept any value without a type check.
- **Typed** — the normal case. All variables carry a concrete type unless the programmer actively opts into `Any`.

This mirrors the Go philosophy: use interfaces (`Any` ≈ `interface{}`) when you need to, and use concrete types when you can afford to be specific. Unlike Go, picoceci makes the dynamic opt-in *explicit* — there is no implicit `interface{}` escape.

### 5.3 Typed declarations as living documentation

In embedded IoT code, human readers matter as much as compilers. When a student opens `TempSensor.pc` and sees:

```picoceci
| bus: I2C  address: Int  lastC: Float  active: Bool |
```

they immediately understand the object's state without reading any method bodies. This is the *specification as code* principle — declarations double as documentation that is always up to date.

### 5.4 Zero-value discipline

The zero-value rule ("every typed declaration is initialised to its zero value automatically") reduces an entire class of bugs:

| v1 pattern | v2 equivalent |
|---|---|
| `lastC := 0.0.` in `init` | automatic — `lastC: Float` starts at `0.0` |
| `running := false.` in `init` | automatic — `running: Bool` starts at `false` |
| `count := 0.` in `init` | automatic — `count: Int` starts at `0` |

`init` methods become shorter and more focused on configuration rather than tedious zero-filling.

---

## 6. Breaking Change and Migration

**This is a breaking change.** The untyped `| x |` form is no longer valid; the parser will reject it. Since picoceci has not been publicly released, there is no existing user base to protect — this is the right moment to enforce strict type discipline before any external commitments are made.

### 6.1 Scope of migration

Two repositories contain picoceci source files that must be updated:

| Repository | Action |
|---|---|
| `github.com/kristofer/picoceci` | Update all `testdata/` programs; update `LANGUAGE_SPEC.md` and grammar |
| `github.com/kristofer/Canal` | Update all picoceci code examples and generated glue files |

### 6.2 Migration rule

The migration from v1 to v2 is mechanical:

- Any bare `| x |` declaration becomes `| x: Any |`.
- Wherever the intended type is known, replace `Any` with the concrete type (e.g. `| count: Int |`).

A one-pass `sed`/`awk` script can handle the mechanical `Any` substitution; typed annotations are then added incrementally by the developer.

### 6.3 No incremental compatibility path

Unlike the original v0.1 design, there is no "mix typed and untyped in the same block" allowance. Every name in every `| ... |` block must have an explicit `: TypeName`. This keeps the parser and type-checker simple and prevents the gradual degradation of typed codebases through accidental untyped additions.

---

## 7. Open Design Decisions

### 7.1 Typed method parameter syntax

Two candidate syntaxes exist:

**Option A — Inline annotation after parameter name:**

```picoceci
setpoint: sp: Float timestep: deltaT: Float [ ... ]
```

This keeps the selector keyword and the parameter name together but requires the parser to distinguish `sp:` (parameter name colon) from `Float` (type annotation) from the next keyword `timestep:`. The parameter name must always differ from the keyword token that precedes it — e.g., `sp` ≠ `setpoint`, `deltaT` ≠ `timestep`.

**Option B — Separate annotation block:**

```picoceci
setpoint: sp timestep: deltaT [| sp: Float  deltaT: Float | ... ]
```

The method body opens with a typed var-decl that annotates the parameter names introduced by the selector. Those names already exist as local bindings; the typed var-decl simply adds a type constraint and zero-value initialisation rule. Simpler to parse; more verbose.

**Recommendation:** Defer typed method parameters to a follow-on v2.1 iteration. Focus v2.0 on typed local variables and typed object slots.

### 7.2 Generic (parameterised) channel and queue syntax

An initial design of `Channel[Float]` (using square brackets as a type parameter) conflicts with the lexer: `[` currently opens block literals. Two disambiguation approaches were considered:

- **Context-sensitive lexing** — `[` after a type name in a variable declaration is a type parameter, not a block.
- **Alternative delimiter** — use `<<Float>>` or `(Float)` as the type parameter syntax.

**Recommendation:** Use `<<T>>` angle-bracket style throughout (`Channel<<Float>>`, `Queue<<Symbol>>`), or the keyword form `Channel of: Float`. This document uses `<<T>>` in all examples. The final choice is deferred to the design-decision phase (§12 step 2).

### 7.3 Compile-time vs. runtime checking

Full static type checking requires a type-inference pass (significant effort). For v2.0:

- **Type checking is at runtime** — assignment type mismatch raises `TypeError` at the point of assignment.
- **Compile-time** checking (where types are fully inferrable) is deferred to v3.0.

---

## 8. Scope of Changes: Interpreter, Compiler, VM, Documentation

### 8.1 Summary table

| Component | Change category | Estimated complexity |
|---|---|---|
| `pkg/lexer/` | New token for type annotation (`:` in var-decl context) | Low |
| `pkg/ast/` | `VarDecl.Types []string`, `ObjectDecl.SlotTypes []string` | Low |
| `pkg/parser/` | Parse typed var-decls, typed slots | Medium |
| `pkg/object/` | `Object.DeclaredKind Kind` field for runtime type tag | Low |
| `pkg/eval/` | Default-value init; assignment type guard; `TypeError` | Medium |
| `pkg/bytecode/compiler.go` | Emit type-check opcodes; encode slot types | Medium |
| `pkg/bytecode/vm.go` | `CHECK_TYPE` opcode; typed-slot init on object creation | Medium |
| `pkg/runtime/` | `TypeError` object; typed Channel/Queue constructors | Medium |
| `docs/grammar.ebnf` | Updated `var_decl` and `object_decl` productions | Low |
| `LANGUAGE_SPEC.md` | §3 expanded; §5 typed slots; §10 typed channels | Medium |
| `docs/IMPLEMENTATION_PLAN.md` | New section for v2 typing work | Low |
| `docs/picoceci-whitepaper.md` | Examples updated; §4 mental model section | Medium |
| `testdata/` | New typed-variable test programs | Low |

### 8.2 Lexer (`pkg/lexer/`)

The lexer currently emits `PIPE`, `IDENTIFIER`, and `KEYWORD` (identifier + `:`).  In a typed var-decl the colon follows the *variable name*, not a message keyword. Two approaches:

1. **Reuse `KEYWORD` token** — parse `x:` as a keyword token inside `| ... |` and treat the following identifier as the type name. (Minimal lexer change; disambiguation in the parser.)
2. **New `TYPED_NAME` token** — emit a dedicated token `x:Type` pair when a `|`-delimited context is active. (Requires stateful lexer mode.)

**Recommended: approach 1** (reuse `KEYWORD`; disambiguate in parser). No new lexer tokens.

### 8.3 AST (`pkg/ast/`)

**Current `VarDecl`:**

```go
type VarDecl struct {
    Pos   token.Pos
    Names []string
}
```

**v2 `VarDecl`:**

```go
type VarDecl struct {
    Pos   token.Pos
    Names []string   // parallel slices
    Types []string   // always populated; "Any" means explicitly dynamic
}
```

**Current `ObjectDecl`:**

```go
type ObjectDecl struct {
    Pos      token.Pos
    Name     string
    Composes []string
    Slots    []string
    Methods  []*MethodDef
}
```

**v2 `ObjectDecl`:**

```go
type ObjectDecl struct {
    Pos       token.Pos
    Name      string
    Composes  []string
    Slots     []string   // parallel slices
    SlotTypes []string   // always populated; "Any" means explicitly dynamic
    Methods   []*MethodDef
}
```

Parallel slices keep the AST representation straightforward — `SlotTypes[i]` is always a non-empty string. An entry of `"Any"` means the programmer explicitly opted into dynamic typing.

### 8.4 Parser (`pkg/parser/`)

The `parseVarDecl()` function currently reads:

```
'|' identifier* '|'
```

It must be extended to read:

```
'|' ( identifier ':' typeName )+ '|'
```

where `typeName` is an `IDENTIFIER` (primitive keyword like `Int`, `Float`) or a user-defined object/interface name. The `:` and `typeName` are **required** — a bare identifier without a type annotation is a parse error that reports "missing type annotation; use `: Any` for a dynamic variable".

The `parseObjectDecl()` slot parsing path calls `parseVarDecl()` and stores only names; it must additionally store types.

**Estimated parser changes:** approximately 40–60 lines.

### 8.5 Object representation (`pkg/object/`)

A new field on `Object` records the declared type at instance creation:

```go
type Object struct {
    // ... existing fields ...
    DeclaredKind string   // "" = Any; "Int", "Float", "Bool", ... or user type name
}
```

For object slots, the object factory stores a parallel `SlotTypes map[string]string` in the template. When `new` creates an instance, it initialises each typed slot to its zero value.

### 8.6 Interpreter (`pkg/eval/`)

**VarDecl evaluation (current):**

```go
case *ast.VarDecl:
    for _, name := range node.Names {
        env.Define(name)   // sets to nil
    }
```

**v2:**

```go
case *ast.VarDecl:
    for i, name := range node.Names {
        env.DefineTyped(name, node.Types[i])  // sets to zero value for type
    }
```

Because the parser guarantees `node.Types[i]` is always a non-empty string, there is no legacy untyped code path in the evaluator. `"Any"` maps to `nil` as its zero value and imposes no type check on assignment.

**Assignment evaluation** gains a type guard:

```go
case *ast.Assign:
    val, err := interp.evalNode(node.Value, env)
    if err != nil { return nil, err }
    if err := env.CheckType(node.Name, val); err != nil { return nil, err }
    env.Set(node.Name, val)
```

**`Env.CheckType`** looks up the declared type of the named variable and verifies the value's `Kind` matches, raising `TypeError` on mismatch.

**New `TypeError` error kind** joins the existing error family in `pkg/eval/errors.go`.

### 8.7 Bytecode compiler (`pkg/bytecode/compiler.go`)

Two new opcodes:

| Opcode | Operand | Effect |
|---|---|---|
| `INIT_TYPED_LOCAL` | slot-idx, type-tag | initialise local slot to its zero value |
| `CHECK_TYPE` | type-tag | peek stack top; raise TypeError if kind mismatch |

The compiler emits `INIT_TYPED_LOCAL` for each typed variable declaration and `CHECK_TYPE` before each `STORE_LOCAL` / `STORE_INST` that targets a typed slot.

For `Any` variables (`type-tag = ANY_TAG`), `INIT_TYPED_LOCAL` still runs (setting the slot to `nil`) but `CHECK_TYPE` is a no-op that the optimizer can elide — so the runtime overhead for dynamic variables is minimal while still going through the unified typed path.

### 8.8 VM (`pkg/bytecode/vm.go`)

The VM must handle two new opcodes plus update `newObject` to initialise typed slots:

```go
case INIT_TYPED_LOCAL:
    slot     := readUint16(frame)
    typeTag  := readUint8(frame)
    frame.locals[slot] = zeroValueFor(typeTag)

case CHECK_TYPE:
    typeTag := readUint8(frame)
    top     := vm.peek()
    if !kindMatches(top, typeTag) {
        return vm.raiseTypeError(top, typeTag)
    }
```

`zeroValueFor` maps a type tag byte to a pre-allocated zero object (same as interpreter's `DefineTyped`).

### 8.9 Runtime (`pkg/runtime/`)

**TypeError object:**

```picoceci
"Built-in error"
Error TypeError [ "subclass of Error for type mismatch" ]
```

Go-side:

```go
func newTypeError(varName, expected, got string) *object.Object {
    msg := fmt.Sprintf("TypeError: %s expects %s, got %s", varName, expected, got)
    return newError("TypeError", msg)
}
```

**Typed Channel and Queue constructors:**

`Channel new: capacity type: Float` (or the `<<Float>>` syntax from §7.2) creates a channel that rejects non-Float sends.

---

## 9. Documentation Changes

### 9.1 Grammar (`docs/grammar.ebnf`)

Update `var_decl` production:

```ebnf
var_decl
    = "|" , { typed_name } , "|"
    ;

typed_name
    = IDENTIFIER , ":" , type_name
    ;

type_name
    = "Int" | "Float" | "Bool" | "String" | "Char"
    | "Symbol" | "ByteArray" | "Array" | "Any" | "Nil"
    | IDENTIFIER          (* user-defined object or interface name *)
    | IDENTIFIER , "<<" , type_name , ">>"   (* generic: Channel<<Float>> *)
    ;
```

Note: the `[ ":" , type_name ]` optional form from v0.1 is replaced by `":" , type_name` (required). A bare identifier inside `| ... |` is a syntax error.

Update `object_decl` to use the new `var_decl`.

Update `method_def` if Option A for typed parameters is chosen (§7.1).

### 9.2 Language Specification (`LANGUAGE_SPEC.md`)

Sections to update:

| Section | Change |
|---|---|
| §2.3 | Add type keywords (`Int`, `Float`, `Bool`, …) to reserved words |
| §3 (Types and Values) | Expand with zero-value table; describe `Any` as explicit opt-in dynamic type |
| §4.4 (Assignment) | Add type-guard description and TypeError; update `varDecl` example to use typed form |
| §5.1 (Object declaration) | Typed slot syntax and zero-value init; remove untyped slot example |
| §6 (Interfaces) | Typed interface variables |
| §10 (Concurrency) | Typed `Channel<<T>>` and `Queue<<T>>` |
| §14 (Grammar summary) | Updated `varDecl` production (`:` required, not optional) |
| New §3.x | "Typed declarations and zero values" |
| New §9.x | "TypeError" in built-in error kinds |

### 9.3 Implementation Plan (`docs/IMPLEMENTATION_PLAN.md`)

A new **Phase 9 — Typed Variables (v2)** section documents the tasks described in §8 above, following the same structure as existing phases (inputs, deliverables, acceptance criteria).

### 9.4 Whitepaper (`docs/picoceci-whitepaper.md`)

| Section | Change |
|---|---|
| §3 (Why Smalltalk syntax?) | Add note: "v2 adds opt-in typed declarations for reliability" |
| §5 (Domains and Composition) | Update TempSensor example to typed form |
| §6.2 (TempSensor object) | Rewrite with `| bus: I2C address: Int lastC: Float |` |
| §6.3 (Environmental Monitor) | Rewrite with typed channels |
| New §4.x | "The Two-Tier Type Philosophy: Any and Typed" |
| New §7 or Appendix | "Typed picoceci: the v2 mental model" |
| Appendix A (Quick Reference) | Add typed var-decl syntax |
| Appendix B (Sensor patterns) | Update patterns 1 and 2 with typed channels |

---

## 10. Work Breakdown and Estimates

The following tasks are sized relative to each other. All are conditional on design decisions in §7 being finalised first.

| # | Task | Deliverable | Estimated effort |
|---|------|-------------|-----------------|
| T1 | Finalise syntax design (§7 decisions) | ADR document | 0.5 days |
| T2 | Lexer: KEYWORD-reuse in var-decl context | `pkg/lexer/` | 0.5 days |
| T3 | AST: typed VarDecl and ObjectDecl (required types) | `pkg/ast/ast.go` | 0.5 days |
| T4 | Parser: required typed var-decl, typed slots, error on bare `| x |` | `pkg/parser/parser.go` | 1 day |
| T5 | Object: DeclaredKind field, zero values | `pkg/object/object.go` | 0.5 days |
| T6 | Eval: DefineTyped, CheckType, TypeError | `pkg/eval/eval.go` + `errors.go` | 1 day |
| T7 | Runtime: TypeError object, typed Channel | `pkg/runtime/` | 1 day |
| T8 | Bytecode compiler: INIT_TYPED_LOCAL, CHECK_TYPE | `pkg/bytecode/compiler.go` | 1.5 days |
| T9 | VM: new opcodes, typed slot init | `pkg/bytecode/vm.go` | 1 day |
| T10 | Test data: new typed programs, error tests for bare `| x |` | `testdata/typed/` | 1 day |
| T10a | Migrate existing `testdata/` programs to typed form | `testdata/` | 0.5 days |
| T10b | Update `github.com/kristofer/Canal` picoceci references | Canal repo | 1 day |
| T11 | Docs: grammar (`:` required), LANGUAGE_SPEC, IMPL_PLAN | markdown edits | 1 day |
| T12 | Docs: whitepaper update | `docs/picoceci-whitepaper.md` | 1 day |
| T13 | Integration test and bug-fix pass | CI green | 1 day |
| **Total** | | | **~12.5 days** |

---

## 11. Feasibility Assessment

### 11.1 Technical feasibility

**High.** The change is technically straightforward; the main difference from v0.1 is that there is no backward-compat code path to maintain. Removing the optional `[ ':' typeName ]` grammar form actually simplifies the parser. The existing code in `pkg/eval/eval.go` already distinguishes value kinds via `object.Kind`; `CheckType` is a straightforward lookup-and-compare operation.

This is a breaking change in language semantics. However, since picoceci has not been publicly released, the migration cost is confined to programs within this repository and `github.com/kristofer/Canal`. The migration is mechanical (§6.2) and can be completed before any public release.

The most complex part is still the bytecode compiler path (T8), because it must emit the right initialisation sequence for every variable. The `Any` type requires `INIT_TYPED_LOCAL` (to set the slot to `nil`) but skips `CHECK_TYPE`, preserving near-zero overhead for dynamic variables.

### 11.2 Impact on existing tests

Existing v1 test programs in `testdata/` will **fail to parse** after this change because they contain bare `| x |` declarations. They must be migrated:

1. Any `| x |` that genuinely needs dynamic behaviour becomes `| x: Any |`.
2. Any `| x |` where the type is known should be given its concrete type.

This migration must be completed as part of the v2 implementation work (added as task T10a — see §10). All migrated tests must pass before the feature is considered complete. New test cases for parse errors on bare `| x |` forms should also be added.

### 11.3 Risk areas

| Risk | Mitigation |
|---|---|
| Lexer ambiguity: `x:` in var-decl vs message keyword | Parser-level disambiguation (check position inside `| ... |`) |
| Generic channel syntax conflicts with block `[` | Use `<<T>>` instead of `[T]`; revisit in v2.1 |
| Typed parameters (§7.1) are complicated | Defer to v2.1; focus v2.0 on local vars and slots |
| Runtime type check overhead on hot paths | `Any` variables skip `CHECK_TYPE`; typed-only paths pay the check cost once per assignment |
| Existing `testdata/` programs break | Mechanical migration (§6.2); migration is small and contained |
| Canal repository has picoceci references | Coordinate update of `github.com/kristofer/Canal` alongside this work (§6.1) |
| Whitepaper tone consistency | Maintain the whitepaper's accessible, first-person narrative style while adding technical precision |

### 11.4 Scope conclusion

This is a **medium-scope** feature: approximately 11 developer-days of implementation work plus 2 additional days for review and iteration. The removal of backward compatibility simplifies the interpreter and parser slightly (no dual code paths), while adding a one-time migration cost for `testdata/` and Canal.

The feature is independently deliverable in a new `v2-types` branch without blocking other in-progress work (Phase 3 bytecode VM, Phase 4 module system). However, the Canal update should be coordinated closely so that both repositories move to v2 semantics together.

The impact on the programmer's mental model is **positive and significant**: mandatory typed declarations transform picoceci from a quick-scripting language into a language suitable for building *reliable* IoT domains from the very first program — which is exactly the spacecraft-watchman vision the project was designed to realize.

---

## 12. Recommended Next Steps

1. Review this document with the project team.
2. Make design decisions on the open items in §7 (parameter syntax and channel generic syntax).
3. Write an Architecture Decision Record (ADR) capturing those decisions and confirming that mandatory typed declarations are the chosen direction.
4. Audit `testdata/` in this repository and all picoceci code in `github.com/kristofer/Canal` for bare `| x |` declarations (T10a / T10b).
5. Once ADR is approved, open a `v2-types` branch and begin with T1–T6 (interpreter path), keeping the bytecode path (T7–T9) in a subsequent PR.
6. Coordinate the Canal update to land at the same time as, or immediately after, the interpreter PR.
7. Update this plan document with any scope changes discovered during implementation.

---

*End of Typed Variables Plan v0.2-draft*
