# picoceci Language Specification

Version: 2.0-draft  
Status: Specification complete; implementation in progress  
Target runtime: TinyGo 0.32+ on ESP32-S3-N16R8 (and compatible MCUs)

---

## Table of Contents

1. [Design Goals](#1-design-goals)
2. [Lexical Structure](#2-lexical-structure)
3. [Types and Values](#3-types-and-values)
4. [Expressions](#4-expressions)
5. [Objects and Composition](#5-objects-and-composition)
6. [Interfaces](#6-interfaces)
7. [Control Flow](#7-control-flow)
8. [Blocks (Closures)](#8-blocks-closures)
9. [Error Handling](#9-error-handling)
10. [Concurrency](#10-concurrency)
11. [Memory Model](#11-memory-model)
12. [Module System](#12-module-system)
13. [Interop with TinyGo / Canal](#13-interop-with-tinygo--canal)
14. [Grammar Summary](#14-grammar-summary)

---

## 1. Design Goals

| Goal | Detail |
|---|---|
| **Small footprint** | Interpreter + runtime < 128 KB RAM on ESP32-S3-N16R8 |
| **Familiar syntax** | Smalltalk message-sending syntax (unary / binary / keyword) |
| **Go semantics** | Interfaces, structural typing, composition — no inheritance |
| **Deterministic** | No stop-the-world GC; reference-counted or arena-based allocation |
| **Safe** | No raw pointers exposed to picoceci code |
| **Composable** | Objects compose other objects; capabilities compose at the kernel level |
| **Concurrent** | First-class access to FreeRTOS tasks, queues, semaphores via TinyGo |
| **Persistent** | SD card filesystem access as a first-class citizen |

---

## 2. Lexical Structure

### 2.1 Character set

Source files are UTF-8.  Identifiers may contain Unicode letters.

### 2.2 Comments

```
"This is a comment — double-quoted, exactly as in Smalltalk."
```

Comments do not nest.

### 2.3 Identifiers

```
identifier ::= letter (letter | digit | '_')*
letter     ::= [A-Za-z] | unicode-letter
```

Identifiers starting with a capital letter are conventionally used for object (type) names.  All others are variable or message names.

**Reserved words** (cannot be used as identifiers):

```
nil  true  false  self  super  thisContext
object  interface  compose  import  ^
```

**Type keywords** (used in typed variable declarations — also reserved):

```
Int  Float  Bool  String  Char  Symbol  ByteArray  Array  Any  Nil
```

### 2.4 Keywords

A keyword is an identifier immediately followed by `:` with no space:

```
at:   put:   ifTrue:   value:   spawn:
```

### 2.5 Literals

#### Integer

```
42        "decimal"
16rFF     "hex — base#value notation like Smalltalk"
2r1010    "binary"
```

#### Float

```
3.14   1.5e-3
```

#### Character

```
$A   $\n   $\t
```

#### String

Single-quoted, with `''` as escape for a literal single quote:

```
'Hello'   'it''s fine'
```

#### Symbol

```
#hello   #at:put:   #'with spaces'
```

Symbols are interned; two symbols with the same characters are identical objects.

#### Byte Array

```
#[ 1 2 3 255 ]
```

#### Array literal

```
#( 1 'two' #three )
```

Elements must be literals.

#### Boolean

```
true   false
```

#### Nil

```
nil
```

### 2.6 Special tokens

```
.   statement terminator (optional before ']' or at EOF)
|   variable declaration delimiter
:=  assignment
^   return
;   cascade
( ) block of sub-expressions (parentheses)
[ ] block literal
{ } object body
:   part of keyword
#   symbol / array prefix
```

---

## 3. Types and Values

picoceci v2 is **statically typed by declaration**: every variable must carry an explicit type annotation.  The runtime tags every value with its kind and enforces the declared type at the point of assignment.  Use `Any` to opt into dynamic typing where genuinely needed.

The following primitive value types exist:

| Type tag | Description | Size (bytes) |
|---|---|---|
| `SmallInt` | 63-bit (or 31-bit) tagged integer | 0 (tagged pointer) |
| `Float` | IEEE-754 double | 8 |
| `Bool` | true / false | 0 (tagged pointer) |
| `Nil` | the nil object | 0 (tagged pointer) |
| `Char` | Unicode code point | 4 |
| `String` | immutable UTF-8 byte array | variable |
| `Symbol` | interned string | variable |
| `ByteArray` | mutable byte buffer | variable |
| `Array` | heterogeneous array | variable |
| `Block` | closure | variable |
| `Object` | user-defined object | variable |

All values are **heap-allocated objects** internally; SmallInt / Bool / Nil / Char are **immediate values** encoded in the pointer word to avoid heap pressure.

### 3.1 Nil

`nil` is the zero value.  Sending any message to nil other than `isNil`, `notNil`, `printString`, and `==` raises a `MessageNotUnderstood` error.

### 3.2 Booleans

`true` and `false` are the only instances of the `Boolean` interface.  They respond to:

```
ifTrue: aBlock
ifFalse: aBlock
ifTrue: trueBlock ifFalse: falseBlock
& aBool   "and"
| aBool   "or"
not
printString
```

### 3.3 Numbers

`SmallInt` and `Float` respond to:

```
+ - * /          "arithmetic"
= ~= < > <= >=   "comparison"
//               "integer division"
\\               "modulo"
abs  negated  sqrt  floor  ceiling  rounded
asFloat  asInteger
printString
```

Arithmetic promotes integers to floats when mixed.

### 3.4 Strings

Strings are immutable.  Mutability is provided by `WriteStream`.

```
size
at: index             "1-based"
copyFrom: start to: stop
, aString             "concatenation"
= aString
includesSubString: sub
asUppercase  asLowercase
asSymbol
asInteger  asFloat
printString
```

### 3.5 Arrays

Arrays are fixed-size after creation.

```
new: size
new: size withAll: value
at: index
at: index put: value
size
do: aBlock
collect: aBlock
select: aBlock
detect: aBlock
inject: initial into: aBlock
with: aBlock
printString
```

### 3.6 ByteArray

Like Array but holds only bytes (0–255).

### 3.7 Typed declarations and zero values

Every variable declaration **must** include a type annotation.  The bare `| x |` form is a parse error in v2 — use `| x: Any |` to retain fully dynamic behaviour.

```picoceci
| x: Int  y: Float  running: Bool  name: String |
```

When a typed variable is declared but not yet assigned, it is automatically initialised to its type's *zero value*:

| Type keyword | Zero value | Notes |
|---|---|---|
| `Int` | `0` | 63-bit signed integer |
| `Float` | `0.0` | IEEE-754 double |
| `Bool` | `false` | |
| `String` | `''` | empty string (not nil) |
| `Char` | `$\0` | NUL character |
| `Symbol` | `#''` | empty interned symbol |
| `ByteArray` | `#[]` | empty byte array |
| `Array` | `#()` | empty array |
| `Nil` | `nil` | explicit nil type |
| `Any` | `nil` | explicitly dynamic; accepts any value, no type check on assignment |
| `<ObjectName>` | `nil` | user-defined object type; nil until assigned |
| `<InterfaceName>` | `nil` | interface type; nil until assigned |

`Any` is the explicit escape hatch for code that genuinely needs dynamic behaviour.  It must be stated explicitly rather than implied by omission.  Assigning a value of the wrong type to a non-`Any` variable raises a `TypeError` at runtime.

---

## 4. Expressions

Evaluation order follows Smalltalk precedence (highest to lowest):

1. **Unary messages** (left to right)  
2. **Binary messages** (left to right)  
3. **Keyword messages** (left to right; only one keyword message at the same level)  
4. **Assignment** (right to left)  
5. **Cascade** — applies further messages to the *same receiver* as the previous message

Parentheses override precedence.

### 4.1 Unary messages

```
42 factorial.
'hello' reversed.
myObject close.
```

### 4.2 Binary messages

```
3 + 4.
x < y.
a , b.        "string concat"
```

Operator characters: `+ - * / < > = ~ @ , & | \ ? ! %`  
Multi-character combinations are allowed: `<= >= ~= <<`.

### 4.3 Keyword messages

```
collection at: 2.
dict at: #key put: value.
stream nextPutAll: 'hello'; nl.
```

### 4.4 Assignment

```picoceci
| x: Int  y: Int |
x := 42.
y := x + 1.
```

Variables must be declared in a `| ... |` declaration before use within a scope.  Every declaration **requires** a type annotation (see §3.7).  Assigning a value whose kind does not match the declared type raises a `TypeError` at runtime:

```picoceci
| count: Int |
count := 'hello'.   "TypeError: count expects Int, got String"
```

Use `| count: Any |` to allow any value without a type check.

### 4.5 Cascade

```
Transcript
    print: 'a';
    print: 'b';
    nl.
```

The semicolon sends the next message to the *original receiver* (`Transcript`).

### 4.6 Return

```
^value
```

`^` returns from the enclosing *method*.  Returning from a block exits the block (not the method) unless it is a non-local return from inside a method body — in that case it exits the method.

### 4.7 Self and super

- `self` — the current object receiver.
- `super` — the same object, but message lookup starts from the *composed object* being delegated to (analogous to Go's embedded struct promotion, not inheritance).

---

## 5. Objects and Composition

There are **no classes** in picoceci.  Instead, `object` defines a named prototype — a template for creating instances.

### 5.1 Object declaration

```picoceci
object Counter {
    | count: Int |

    inc [
        count := count + 1.
        ^self
    ]

    dec [
        count := count - 1.
        ^self
    ]

    value [
        ^count
    ]

    printString [
        ^'Counter(', count printString, ')'
    ]
}
```

- `| count: Int |` — typed instance variable declaration (slot).  `count` is automatically initialised to `0` (the zero value for `Int`); no `init` method is needed for zeroing.
- Methods are unary (`inc`, `value`) or keyword (`at:`, `at:put:`) or binary (`+`).
- `init` is still called automatically by `new` when defined, but is needed only for non-zero initialisation.
- Methods can take parameters using keyword syntax: `add: n [ count := count + n. ^self ]`.

### 5.2 Creating instances

```picoceci
| c: Counter |
c := Counter new.
```

### 5.3 Composition

`compose` includes all slots and methods of another object:

```picoceci
object LoggedCounter {
    compose Counter.

    inc [
        super inc.
        Console println: 'incremented to ', self value printString.
        ^self
    ]
}
```

Rules:

- `compose` copies slot names and methods from the named object's *template*.
- If the current object defines a method with the same name, it **overrides** the composed method.
- `super <message>` dispatches to the composed object's method for that message.
- Multiple composition is allowed; conflict resolution is declaration order (last wins), but the compiler warns on ambiguous overrides.
- There is **no runtime class hierarchy** — `isKindOf:` is replaced by interface checking.

### 5.4 Object literals

For simple ad-hoc objects:

```picoceci
| point: Any |
point := object { x := 3. y := 4 }.
Console println: point x printString.
```

These are anonymous objects with no named template.  They satisfy any interface whose messages they respond to.  Because no named type exists for the literal, declare the variable as `Any` (or as an interface type that the literal satisfies).

---

## 6. Interfaces

An `interface` declares a set of messages an object must respond to.

```picoceci
interface Incrementable {
    inc
    dec
    value
}

interface Printable {
    printString
}
```

### 6.1 Interface satisfaction

picoceci uses **structural typing** — an object satisfies an interface if it responds to all the messages the interface declares.  No explicit declaration (`implements`) is needed.

### 6.2 Interface variables

Declare a variable with an interface name as its type to hold any object satisfying that interface:

```picoceci
| c: Incrementable |
c := LoggedCounter new.
(c satisfies: Incrementable)
    ifTrue: [ Console println: 'yes' ].
```

### 6.3 Interface parameters (future)

Method parameters can be annotated with an interface name for documentation and runtime checking:

```picoceci
addTo: (Incrementable) target times: n [
    n timesRepeat: [ target inc ]
]
```

The runtime raises `InterfaceError` if the argument does not satisfy `Incrementable`.

---

## 7. Control Flow

Control flow uses keyword messages sent to booleans, numbers, or blocks.

### 7.1 Conditionals

```picoceci
x > 0
    ifTrue:  [ Console println: 'positive' ]
    ifFalse: [ Console println: 'non-positive' ].
```

```picoceci
(x = 0)
    ifTrue: [ ^0 ].
```

### 7.2 Loops

```picoceci
1 to: 10 do: [ :i | Console println: i printString ].

[ x > 0 ] whileTrue: [ x := x - 1 ].

[ x < 0 ] whileFalse: [ x := x + 1 ].

5 timesRepeat: [ Console println: 'tick' ].
```

### 7.3 Collection enumeration

```picoceci
#(1 2 3) do: [ :each | Console println: each printString ].

| doubled: Array |
doubled := #(1 2 3) collect: [ :each | each * 2 ].

| evens: Array |
evens := #(1 2 3 4) select: [ :each | each \\ 2 = 0 ].

| sum: Int |
sum := #(1 2 3) inject: 0 into: [ :acc :each | acc + each ].
```

---

## 8. Blocks (Closures)

A block is a first-class object encapsulating deferred computation.

```picoceci
[ :x :y | x + y ]
```

### 8.1 Invoking blocks

| Arity | Message |
|---|---|
| 0 | `value` |
| 1 | `value: arg` |
| 2 | `value: a value: b` |
| n | `valueWithArguments: anArray` |

### 8.2 Blocks are closures

Blocks capture variables from their enclosing scope.

```picoceci
| adder: Block |
adder := [ :n | [ :x | x + n ] ].
(adder value: 5) value: 3.   "=> 8"
```

### 8.3 Non-local return

A `^` inside a block that is lexically nested inside a method exits the *method*, not just the block.

---

## 9. Error Handling

### 9.1 Signal

Any object can be signalled as an error:

```picoceci
Error signal: 'something went wrong'.
```

### 9.2 Handling

```picoceci
[ someRiskyOperation ]
    on: Error
    do: [ :err | Console println: err messageText ].
```

### 9.3 Ensure

```picoceci
[ file read ]
    ensure: [ file close ].
```

### 9.4 Built-in error kinds

| Name | Meaning |
|---|---|
| `Error` | Base error |
| `MessageNotUnderstood` | Object received unknown message |
| `TypeError` | Assignment type mismatch (v2 typed variables) |
| `InterfaceError` | Argument does not satisfy interface |
| `IndexOutOfBounds` | Array / string index out of range |
| `IOError` | Filesystem / network failure |
| `TaskError` | FreeRTOS task failure |
| `CapabilityError` | Canal capability violation |

---

## 10. Concurrency

picoceci exposes FreeRTOS primitives through a set of built-in objects.

### 10.1 Tasks

```picoceci
| task: Any |
task := Task spawn: [
    [ true ] whileTrue: [
        Console println: 'tick'.
        Task delay: 1000   "milliseconds"
    ]
].
task priority: 2.
task name: 'blinker'.
```

`Task spawn: aBlock` creates and starts a FreeRTOS task running the block.

| Message | FreeRTOS equivalent |
|---|---|
| `Task spawn: aBlock` | `xTaskCreate` |
| `Task spawn: aBlock stackSize: n` | `xTaskCreate` with stack size |
| `task suspend` | `vTaskSuspend` |
| `task resume` | `vTaskResume` |
| `task delete` | `vTaskDelete` |
| `Task delay: ms` | `vTaskDelay` |
| `Task yield` | `taskYIELD` |
| `Task currentPriority` | `uxTaskPriorityGet` |

### 10.2 Queues

Queues carry a type parameter that restricts what may be sent.  Use `Queue<<TypeName>>` to declare a typed queue:

```picoceci
| q: Queue<<Int>> |
q := Queue new: 10.

"Producer"
Task spawn: [
    q send: 42.
    q send: 99
].

"Consumer"
Task spawn: [
    [ true ] whileTrue: [
        | item: Int |
        item := q receive.
        Console println: item printString
    ]
].
```

Sending a value whose type does not match raises a `TypeError` at the point of send.  An unparameterised `Queue<<Any>>` accepts any value (equivalent to v1 behaviour).

| Message | FreeRTOS equivalent |
|---|---|
| `Queue new: size` | `xQueueCreate` |
| `q send: item` | `xQueueSend` |
| `q send: item timeout: ms` | `xQueueSend` with timeout |
| `q receive` | `xQueueReceive` (block forever) |
| `q receive timeout: ms` | `xQueueReceive` with timeout |
| `q count` | `uxQueueMessagesWaiting` |

### 10.3 Semaphores

```picoceci
| sem: Any |
sem := Semaphore new.           "binary semaphore"
sem := Semaphore counting: 4.  "counting semaphore, max 4"

sem take.
sem give.
sem take timeout: 500.
```

| Message | FreeRTOS equivalent |
|---|---|
| `Semaphore new` | `xSemaphoreCreateBinary` |
| `Semaphore counting: n` | `xSemaphoreCreateCounting` |
| `Semaphore mutex` | `xSemaphoreCreateMutex` |
| `sem take` | `xSemaphoreTake` (block) |
| `sem take timeout: ms` | `xSemaphoreTake` with timeout |
| `sem give` | `xSemaphoreGive` |

### 10.4 Timers

```picoceci
| t: Any |
t := Timer after: 500 do: [ Console println: 'fired' ].
t := Timer every: 1000 do: [ led toggle ].
t stop.
t start.
t reset.
```

### 10.5 Channels (higher-level)

`Channel` is a picoceci-level abstraction over Queue with Go-like syntax and a mandatory type parameter:

```picoceci
| ch: Channel<<Float>> |
ch := Channel new: 5.
ch <- 3.14.           "send — TypeError if not Float"
| v: Float |
v := <-ch.            "receive"
```

Multiple typed channels can be declared together:

```picoceci
| tempChan:  Channel<<Float>>
  alertChan: Channel<<String>>
  cmdQueue:  Queue<<Symbol>>
|
```

Sending a value of the wrong type raises a `TypeError` at the point of send, before it reaches any consumer task.  Use `Channel<<Any>>` to allow mixed-type payloads.

---

## 11. Memory Model

### 11.1 Object layout

Each heap-allocated object is a contiguous block:

```
[ header: 4B | refcount or GC mark: 4B | kind: 2B | slot-count: 2B | slots... ]
```

### 11.2 Allocation strategy

picoceci uses one of two strategies selectable at build time:

1. **Reference counting** (default for MCU targets) — zero-overhead collection of acyclic graphs; cycle collector runs periodically.
2. **Arena allocator** — all allocations are in one or more arenas that are freed atomically; useful for request-scoped processing.

### 11.3 Stack vs heap

Blocks and small integers are stack-allocated when the compiler can prove they do not escape.  All other objects are heap-allocated.

### 11.4 Limits

| Resource | Default | Configurable |
|---|---|---|
| Heap size | 64 KB | build constant |
| Stack size per task | 4 KB | `Task spawn: ... stackSize: n` |
| Max object slots | 255 | fixed |
| Max method size (bytecodes) | 64 KB | fixed |
| Symbol table | 2 KB | build constant |

---

## 12. Module System

### 12.1 Files

A picoceci source file is called a **module**.  By convention, each top-level `object` or `interface` lives in its own file named after it (`Counter.pc`, `Queue.pc`).

### 12.2 Import

```picoceci
import 'Counter'.
import '/sdcard/libs/Sensor'.
```

`import` searches in order:

1. The built-in standard library (compiled into the runtime).
2. The SD card path `/sdcard/picoceci/libs/`.
3. Absolute paths.

Circular imports are detected and raise a compile-time error.

### 12.3 Namespaces

All names imported from a module are merged into the current namespace.  To avoid collision, prefix object names:

```picoceci
import 'sensors/DHT22'.     "brings in DHT22Sensor"
import 'sensors/BME280'.    "brings in BME280Sensor"
```

---

## 13. Interop with TinyGo / Canal

### 13.1 Calling TinyGo functions

TinyGo functions can be exposed to picoceci via a **bridge declaration**:

```picoceci
"Generated glue — not written by hand"
foreign gpio_set_pin_value(pin Int, value Int)
```

Bridge declarations are generated by the `picoceci-bindgen` tool from TinyGo source.

### 13.2 Canal capabilities

A Canal capability is a handle to a kernel resource.  picoceci wraps capabilities as objects:

```picoceci
| cap: Any |
cap := Canal capability: #uart0.
cap send: 'hello\n' asBytes.
cap close.
```

Canal capability objects respond to:

| Message | Meaning |
|---|---|
| `Canal capability: #name` | Acquire capability by name |
| `cap send: bytes` | Write bytes to capability |
| `cap receive: n` | Read up to n bytes |
| `cap close` | Release capability |
| `cap delegate: anotherTask` | Transfer capability ownership |

### 13.3 GPIO / peripheral objects (built-in)

```picoceci
| led: Any |
led := GPIO pin: 2 direction: #output.
led high.
led low.
led toggle.

| btn: Any |
btn := GPIO pin: 0 direction: #input pullup: true.
btn waitForEdge: #rising timeout: 5000.
```

### 13.4 UART

```picoceci
| uart: Any |
uart := UART new: 0 baud: 115200.
uart println: 'Hello from picoceci'.
uart readLine.
```

### 13.5 I²C / SPI

```picoceci
| i2c: Any |
i2c := I2C new: 0 sda: 21 scl: 22 speed: 400000.
i2c writeTo: 16r48 bytes: #[1 2 3].
| data: ByteArray |
data := i2c readFrom: 16r48 count: 4.
```

---

## 14. Grammar Summary

See [docs/grammar.ebnf](docs/grammar.ebnf) for the complete formal grammar.  A compact summary:

```
program         = (statement '.'?)* EOF

statement       = '^' expression
                | varDecl
                | expression

varDecl         = '|' typedName { typedName } '|'
typedName       = identifier ':' typeName
typeName        = 'Int' | 'Float' | 'Bool' | 'String' | 'Char'
                | 'Symbol' | 'ByteArray' | 'Array' | 'Any' | 'Nil'
                | IDENTIFIER
                | IDENTIFIER '<<' typeName '>>'

expression      = assignment | cascade

assignment      = identifier ':=' expression

cascade         = keywordExpr (';' (unaryMsg | binaryMsg | keywordMsg))*

keywordExpr     = binaryExpr (KEYWORD binaryExpr)*

binaryExpr      = unaryExpr (BINOP unaryExpr)*

unaryExpr       = primary UNARY*

primary         = literal
                | identifier
                | block
                | '(' expression ')'
                | objectLiteral

literal         = INTEGER | FLOAT | STRING | SYMBOL | CHARACTER
                | BYTEARRAY | ARRAY | 'true' | 'false' | 'nil'

block           = '[' (':' identifier)* ('|' varDecl)? statement* ']'

objectDecl      = 'object' IDENTIFIER '{' varDecl? method* '}'

method          = (UNARY | BINOP | KEYWORD+) '[' varDecl? statement* ']'

interfaceDecl   = 'interface' IDENTIFIER '{' methodSig* '}'

methodSig       = UNARY | BINOP | (KEYWORD identifier)+

composeDecl     = 'compose' IDENTIFIER '.'

importDecl      = 'import' STRING '.'
```

---

*End of picoceci Language Specification v2.0-draft*
