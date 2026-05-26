# picoceci: A Beginner's Guide

## A Small Language for Resilient Comput
*Kristofer Younger — ZipCode Wilmington*

*describes picoceci v3.*

---

> *"I once dreamed of a spacecraft that could fix itself. Not the grand, gilded kind
> you see in the movies — just a real ship, humming through the dark, tended by a
> thousand tiny electronic watchmen."*

---

# Preface

This book is for programmers who are new to picoceci — and possibly new to
Smalltalk-style message passing, Go-style composition, or embedded programming on
microcontrollers. You do not need experience with any of these things. You do need
curiosity, a willingness to type code and watch what happens, and perhaps a small
tolerance for a language designer who occasionally gets carried away with
spacecraft metaphors.

picoceci (pronounced "pee-ko-cheh-chee," or however you like) is a small,
interpreted, message-passing language. Its name comes from *pico* (small) and
*ceci* (Italian for chickpea — small, high-protein, full of goodness). The
language borrows its syntax from Smalltalk and its structural semantics from Go.
It is designed to run on microcontrollers — specifically the ESP32-S3 — via
TinyGo, and to serve as the scripting layer for networks of tiny computing nodes
that could one day monitor and control the systems aboard a spacecraft.

That is the vision. But you can start with `Console println: 'Hello, picoceci!'`
and work your way up from there.

This guide is organized into five chapters:

1. **First Steps** — syntax, messages, variables, blocks, and the REPL
2. **Objects and Composition** — building things from smaller things
3. **Patterns in picoceci** — classic CS patterns expressed in the language
4. **Concurrency and Channels** — tasks, queues, and Go-style communication
5. **Toward the Mesh** — embedded hardware, Canal, and resilient spacecraft networks

Each chapter builds on the last. By the end, you will understand how picoceci's
two great influences — Smalltalk's message-passing elegance and Go's pragmatic
composability — combine into something uniquely suited for programming meshes of
small, cooperating computers.

Let's begin.

---

\newpage

# Chapter 1: First Steps

## 1.1 Getting Started

picoceci (v3) runs on any machine with Go installed. Clone the repository and build:

```bash
git clone https://github.com/kristofer/picoceci
cd picoceci
go build ./...
```

Start the REPL (Read-Eval-Print Loop):

```bash
./picoceci repl
```

You will see a prompt. Type your first program:

```picoceci
Console println: 'Hello, picoceci!'.
```

The period at the end is a statement terminator, borrowed from Smalltalk. It is
optional at the end of a line, but it is good style to include it.

What just happened? You *sent a message* to an object. The object is `Console`
(the serial output stream). The message is `println:` — a keyword message that
takes one argument, the string `'Hello, picoceci!'`. The Console received the
message and printed the string followed by a newline.

This is the central idea of picoceci: **everything is an object, and you interact
with objects by sending them messages.**


## 1.2 Messages: The Only Verb

picoceci has exactly three kinds of message, borrowed directly from Smalltalk.
Understanding these three forms is understanding roughly half the language.

### Unary Messages

A unary message is a single word sent to an object with no arguments:

```picoceci
42 factorial.          "send 'factorial' to the integer 42"
'hello' reversed.      "send 'reversed' to the string 'hello'"
true not.              "send 'not' to the boolean true — result: false"
```

Unary messages bind tightest. In any expression, unary messages are evaluated
first, left to right.

### Binary Messages

A binary message is an operator with exactly one argument:

```picoceci
3 + 4.                 "=> 7"
'Hello' , ' world'.    "=> 'Hello world'  (comma is concatenation)"
10 > 5.                "=> true"
17 \\ 5.               "=> 2   (modulo)"
```

Operators can be one or two characters from the set
`+ - * / < > = ~ @ , & | \ ? ! %`. Binary messages bind after unary messages.

### Keyword Messages

A keyword message has one or more colon-terminated keywords, each followed by an
argument:

```picoceci
Console println: 'hi'.                         "one keyword"
array at: 3 put: 'banana'.                     "two keywords"
timer every: 1000 do: [ led toggle ].          "two keywords"
```

Keyword messages bind last (loosest). Use parentheses to override:

```picoceci
Console println: (3 + 4) printString.
```

Without the parentheses, `println:` would try to consume `3` as its argument and
the `+ 4` would be left dangling.

### Precedence Summary

| Priority | Kind     | Example                        |
|----------|----------|--------------------------------|
| 1 (highest) | Unary    | `42 factorial`              |
| 2        | Binary   | `3 + 4`                        |
| 3 (lowest)  | Keyword  | `Console println: x`       |

This three-tier system is the entirety of picoceci's operator precedence. There
are no special cases, no table of 15 precedence levels to memorize. One rule,
applied consistently.


## 1.3 Literals and Types

picoceci is statically typed by declaration. Every variable must carry an explicit
type annotation. Here are the built-in types and their literal forms:

```picoceci
42              "Int — integer"
3.14            "Float — floating point"
'hello'         "String — single-quoted, immutable"
#hello          "Symbol — interned string (fast equality)"
$A              "Char — a Unicode character"
true  false     "Bool"
nil             "Nil — the absence of a value"
#(1 2 3)        "Array — fixed-size, heterogeneous"
#[255 0 128]    "ByteArray — fixed-size, bytes only"
```

Integers support alternate bases using Smalltalk notation:

```picoceci
16rFF           "hexadecimal — 255"
2r1010          "binary — 10"
8r77            "octal — 63"
```


## 1.4 Variables and Assignment

Variables must be declared in a `| ... |` block before use. Every declaration
requires a type:

```picoceci
let x: Int.
let y: Float.
let name: String.
x := 42.
y := 3.14.
name := 'picoceci'.
```

Each type has a *zero value* — the value a variable holds before you assign
anything to it:

| Type     | Zero Value |
|----------|------------|
| `Int`    | `0`        |
| `Float`  | `0.0`      |
| `Bool`   | `false`    |
| `String` | `''`       |
| `Array`  | `#()`      |
| `Any`    | `nil`      |

The type `Any` is the escape hatch. A variable declared as `Any` accepts any
value without a type check. You should use it sparingly — explicit types catch
bugs early, which matters a great deal when your code is running on a spacecraft
a hundred thousand kilometres away.

Assigning a value of the wrong type is a runtime error:

```picoceci
let count: Int .
count := 'hello'.   "TypeError: count expects Int, got String"
```


## 1.5 Blocks: Deferred Computation

A block is picoceci's version of a closure — a chunk of code wrapped in square
brackets that can be stored in a variable, passed as an argument, and invoked
later:

```picoceci
let doubler: Block .
doubler := [ :x | x * 2 ].
Console println: (doubler value: 21) printString.   "=> 42"
```

The `:x` after the opening bracket declares a block parameter. Invoke a block by
sending it `value:` (for one argument), `value` (for zero arguments), or
`value: a value: b` (for two arguments).

Blocks capture variables from their enclosing scope. This makes them closures:

```picoceci
let makeAdder: Any .
makeAdder := [ :n | [ :x | x + n ] ].

let addFive: Block .
addFive := makeAdder value: 5.
Console println: (addFive value: 3) printString.    "=> 8"
Console println: (addFive value: 10) printString.   "=> 15"
```

The inner block `[ :x | x + n ]` remembers the value of `n` from the outer
block's scope. This is exactly how closures work in Go, JavaScript, Python, or
any other language with first-class functions — picoceci just spells them
differently.


## 1.6 Control Flow: Messages, Not Keywords

There are no `if`, `while`, or `for` keywords in picoceci. Control flow is
accomplished by sending messages to booleans, numbers, and blocks:

### Conditionals

```picoceci
let temp: Float .
temp := 28.5.
(temp > 30.0)
    ifTrue:  [ Console println: 'too hot!' ]
    ifFalse: [ Console println: 'comfortable' ].
```

`ifTrue:ifFalse:` is a keyword message sent to a boolean object (`true` or
`false`). The boolean decides which block to evaluate. There is nothing magical
about this — `true` evaluates the first block, `false` evaluates the second.

### Loops

```picoceci
"Count from 1 to 10"
1 to: 10 do: [ :i |
    Console println: i printString
].

"While loop"
let count: Int .
count := 10.
[ count > 0 ] whileTrue: [
    Console println: count printString.
    count := count - 1
].

"Repeat n times"
5 timesRepeat: [ Console println: 'tick' ].
```

`to:do:` is a keyword message sent to an integer. `whileTrue:` is a keyword
message sent to a block. The block is evaluated; if it returns `true`, the
argument block is evaluated, and then the receiver block is evaluated again. It is
turtles all the way down.


## 1.7 Cascades: Many Messages, One Receiver

The semicolon (`;`) sends the next message to the same receiver as the previous
one:

```picoceci
Transcript
    print: 'Name: ';
    print: name;
    nl.
```

All three messages — `print: 'Name: '`, `print: name`, and `nl` — are sent to
`Transcript`. Without the cascade, you would need to write `Transcript` three
times.


## 1.8 Comments

Comments are enclosed in double quotes, exactly as in Smalltalk:

```picoceci
"This is a comment."
let x: Int.  "declare a local variable"
x := 42.    "assign it"
```

Comments do not nest.


## 1.9 Chapter Summary

You now know the core syntax of picoceci:

- Everything is an object. You interact with objects by sending messages.
- Three kinds of message: unary, binary, keyword — binding tightest to loosest.
- Variables are typed and declared in `| ... |` blocks.
- Blocks `[ ... ]` are closures — first-class, passable, invocable.
- Control flow is just messages sent to booleans, integers, and blocks.
- Cascades (`;`) send multiple messages to the same receiver.

With just these tools, you can already write real programs. Let's see how.

---

\newpage

# Chapter 2: Objects and Composition

## 2.1 Defining Objects

There are no classes in picoceci. Instead, `object` defines a named template — a
blueprint for creating instances. Think of it as a Go `struct` with methods
attached:

```picoceci
object Counter {
    let count: Int .

    inc   [ count := count + 1. ^self ]
    dec   [ count := count - 1. ^self ]
    value [ ^count ]
    printString [ ^'Counter(' , count printString , ')' ]
}
```

Slots (instance variables) are declared in the `let` statements inside the object
body. Each slot has a type and is automatically initialized to its zero value —
`count` starts at `0` because it is an `Int`.

Methods are defined with a message pattern followed by a block. The `^` symbol
means "return." `^self` returns the object itself, which enables method chaining.


## 2.2 Creating and Using Instances

```picoceci
let c: Counter.
c := Counter new.
c inc; inc; inc.
Console println: c value printString.       "=> 3"
Console println: c printString.             "=> Counter(3)"
```

`Counter new` creates a fresh instance with all slots at their zero values. If
you define an `init` method, it is called automatically by `new`:

```picoceci
object NamedCounter {
    let name: String.
    let count: Int.

    init: aName [
        name := aName.
        count := 0
    ]
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
    printString [ ^name , ': ' , count printString ]
}

let visitors: NamedCounter.
visitors := NamedCounter new init: 'Page Views'.
visitors inc; inc.
Console println: visitors printString.     "=> Page Views: 2"
```


## 2.3 Composition Over Inheritance

This is one of picoceci's most important design decisions, borrowed directly
from Go. There is no class hierarchy. There is no `extends`. There is no `class
Animal` with `class Dog extends Animal`. Instead, objects are built by
*composing* other objects using the `compose` keyword.

```picoceci
object LoggedCounter {
    compose Counter.

    inc [
        super inc.
        Console println: 'incremented to ' , self value printString.
        ^self
    ]
}

let c: LoggedCounter .
c := LoggedCounter new.
c inc; inc.
Console println: c value printString.
```

Output:
```
incremented to 1
incremented to 2
2
```

What happened here:

1. `compose Counter.` copies all of Counter's slots (`count`) and methods (`inc`,
   `dec`, `value`, `printString`) into LoggedCounter.
2. LoggedCounter *overrides* `inc` with its own version.
3. Inside the override, `super inc` delegates to Counter's original `inc` method.
4. The `value` and `dec` methods are inherited as-is.

This is exactly how Go's struct embedding works: the outer type gets all the
methods of the inner type, can override any of them, and can delegate to the
original via the embedded field. picoceci just spells it `compose` and `super`.

### Why Not Inheritance?

Inheritance creates fragile hierarchies. Change the base class and every subclass
may break. The "diamond problem" arises when a class inherits from two classes
that share a common ancestor. Method resolution order becomes a puzzle.

Composition avoids all of this. Each object is a flat collection of slots and
methods. `compose` is a copy operation, not a pointer up a hierarchy. There is no
class tree to navigate, no fragile base class problem, no diamond.

For spacecraft software — where simplicity and predictability are survival
traits — this is a significant advantage.


## 2.4 Multiple Composition

An object can compose more than one other object:

```picoceci
object Timestamped {
    let createdAt: Int.
    init [ createdAt := System ticks ]
    age  [ ^System ticks - createdAt ]
}

object Nameable {
    let name: String.
    name        [ ^name ]
    name: aName [ name := aName. ^self ]
}

object Device {
    compose Timestamped.
    compose Nameable.
    let status: Symbol.

    init: aName [
        super init.
        self name: aName.
        status := #online
    ]
    status  [ ^status ]
    offline [ status := #offline. ^self ]
    online  [ status := #online. ^self ]
    printString [
        ^name , ' (' , status printString , ', age: ' , self age printString , 'ms)'
    ]
}
```

`Device` has the slots and methods of both `Timestamped` and `Nameable`, plus its
own. If two composed objects define a method with the same name, the last
`compose` wins (and the compiler warns about the ambiguity).


## 2.5 Interfaces: Structural Typing

picoceci uses Go-style *structural typing*. An `interface` declares a set of
messages. Any object that responds to all those messages satisfies the interface —
no explicit `implements` declaration needed:

```picoceci
interface Readable {
    reading
}

interface Printable {
    printString
}
```

Any object with a `reading` method satisfies `Readable`. Any object with a
`printString` method satisfies `Printable`. You can declare a variable with an
interface type:

```picoceci
let sensor: Readable.
sensor := TempSensor new init: i2c address: 16r48.
```

The runtime checks that the assigned object actually satisfies the interface. If
it does not, you get an `InterfaceError`.

You can also test satisfaction explicitly:

```picoceci
(myObject satisfies: Readable)
    ifTrue: [ Console println: myObject reading printString ].
```

Structural typing is powerful because it decouples the *definition* of an object
from the *expectations* of its consumers. The TempSensor author does not need to
know about the `Readable` interface. The monitoring code does not need to know
about TempSensor. They agree only on the shape of the messages — and that
agreement is checked at runtime.


## 2.6 A Complete Example: Stack

Here is a classic data structure implemented as a picoceci object, drawn from the
language's own test suite:

```picoceci
object Stack {
    let storage: Array.
    let top: Int.

    init    [ storage := Array new: 64. top := 0 ]
    push: v [ top := top + 1. storage at: top put: v. ^self ]
    pop     [ let v: Any. v := storage at: top. top := top - 1. ^v ]
    peek    [ ^storage at: top ]
    isEmpty [ ^top = 0 ]
    size    [ ^top ]
}

let s: Stack.
s := Stack new.
s push: 10; push: 20; push: 30.
Console println: s pop printString.       "=> 30"
Console println: s pop printString.       "=> 20"
Console println: s size printString.      "=> 1"
Console println: s isEmpty printString.   "=> false"
```

Notice the cascade: `s push: 10; push: 20; push: 30` sends three `push:`
messages to the same Stack instance. This works because `push:` returns `^self`.


## 2.7 Object Literals

For quick, throwaway objects, you can create anonymous instances inline:

```picoceci
let point: Any.
point := object { x := 3. y := 4 }.
Console println: point x printString.    "=> 3"
```

Anonymous objects have no named template, so the variable must be declared as
`Any` (or as an interface type the literal satisfies). They are useful for
ad-hoc data, test fixtures, or configuration bundles.


## 2.8 Chapter Summary

- `object` defines a named template with typed slots and methods.
- `compose` copies another object's slots and methods — flat, no hierarchy.
- `super` delegates to the composed object's original method.
- `interface` declares a message contract; structural typing checks it.
- Multiple composition is allowed; last `compose` wins on conflicts.
- Object literals create anonymous instances for ad-hoc use.

picoceci's object system is deliberately minimal. There is no metaclass
protocol, no method_missing, no runtime class modification. This simplicity is a
feature: every object is a predictable, inspectable thing. You can reason about
it locally, without chasing up a class hierarchy or wondering what some distant
mixin might have modified.

For embedded systems — and especially for spacecraft — predictability is not a
luxury. It is a requirement.

---

\newpage

# Chapter 3: Patterns in picoceci

This chapter translates familiar CS patterns into picoceci. Each example is a
complete, runnable program. If you have the REPL open, type along.


## 3.1 Hello, World

The smallest possible program:

```picoceci
Console println: 'Hello, picoceci!'.
```

One object (`Console`), one keyword message (`println:`), one string argument.


## 3.2 Factorial (Recursive Block)

```picoceci
let fact: Any.
fact := [ :n |
    (n <= 1)
        ifTrue:  [ 1 ]
        ifFalse: [ n * (fact value: n - 1) ]
].
Console println: (fact value: 10) printString.   "=> 3628800"
Console println: (fact value: 0) printString.    "=> 1"
Console println: (fact value: 5) printString.    "=> 120"
```

The block `fact` refers to itself by name. Because blocks are closures, `fact`
captures the variable `fact` from the enclosing scope. Recursive blocks are
picoceci's equivalent of recursive functions.


## 3.3 Fibonacci

```picoceci
let fib: Any.
fib := [ :n |
    (n <= 1)
        ifTrue:  [ n ]
        ifFalse: [ (fib value: n - 1) + (fib value: n - 2) ]
].
Console println: (fib value: 10) printString.   "=> 55"
```

Same pattern as factorial — a self-referencing block. The naive recursive
Fibonacci is exponential in time complexity, but it is a clear illustration of
how blocks and recursion work in picoceci.


## 3.4 FizzBuzz

```picoceci
1 to: 20 do: [ :i |
    ((i \\ 15) = 0)
        ifTrue:  [ Console println: 'FizzBuzz' ]
        ifFalse: [
            ((i \\ 3) = 0)
                ifTrue:  [ Console println: 'Fizz' ]
                ifFalse: [
                    ((i \\ 5) = 0)
                        ifTrue:  [ Console println: 'Buzz' ]
                        ifFalse: [ Console println: i printString ]
                ]
        ]
].
```

The `\\` operator is modulo (the backslash is doubled because `\` is an operator
character). Notice how picoceci's conditional syntax reads almost like prose: "if
i modulo 15 equals 0, then print FizzBuzz; otherwise..."


## 3.5 GCD (Euclidean Algorithm)

```picoceci
let gcd: Any.
gcd := [ :a :b |
    [ b ~= 0 ] whileTrue: [
        let t: Int.
        t := b.
        b := a \\ b.
        a := t
    ].
    a
].
Console println: (gcd value: 48 value: 18) printString.    "=> 6"
Console println: (gcd value: 100 value: 75) printString.   "=> 25"
Console println: (gcd value: 56 value: 98) printString.    "=> 14"
```

A two-argument block using `whileTrue:` for the iterative Euclidean algorithm.
The `~=` operator means "not equal" (borrowed from Smalltalk).


## 3.6 Palindrome Check

```picoceci
let isPalindrome: Any.
isPalindrome := [ :s | s = s reversed ].
Console println: (isPalindrome value: 'racecar') printString.  "=> true"
Console println: (isPalindrome value: 'hello') printString.    "=> false"
Console println: (isPalindrome value: 'level') printString.    "=> true"
Console println: (isPalindrome value: 'madam') printString.    "=> true"
```

Strings respond to `reversed`, returning a new string with characters in reverse
order. The `=` message on strings does value comparison. One line of logic.


## 3.7 Power (Repeated Multiplication)

```picoceci
let power: Any.
power := [ :base :exp |
    let result: Int.
    result := 1.
    exp timesRepeat: [ result := result * base ].
    result
].
Console println: (power value: 2 value: 10) printString.   "=> 1024"
Console println: (power value: 3 value: 4) printString.    "=> 81"
Console println: (power value: 5 value: 0) printString.    "=> 1"
```

`timesRepeat:` is a message sent to an integer. `5 timesRepeat: [ ... ]`
evaluates the block five times. It is picoceci's equivalent of a simple counted
`for` loop.


## 3.8 Accumulator (Closure Factory)

This pattern demonstrates that blocks are true closures with mutable captured
state:

```picoceci
let makeAccumulator: Any.
let counter1: Any.
let counter2: Any.
makeAccumulator := [ :start |
    let n: Any.
    n := start.
    [ :amount | n := n + amount. n ]
].
counter1 := makeAccumulator value: 0.
counter2 := makeAccumulator value: 100.
Console println: (counter1 value: 5) printString.     "=> 5"
Console println: (counter1 value: 10) printString.    "=> 15"
Console println: (counter2 value: 3) printString.     "=> 103"
Console println: (counter1 value: 1) printString.     "=> 16"
```

Each call to `makeAccumulator` creates a new closure with its own private `n`.
The two counters are completely independent — `counter1` and `counter2` do not
share state. This is the same pattern used in JavaScript module closures, Python
factory functions, and Go closures.


## 3.9 Bubble Sort

```picoceci
let arr: Array.
let n: Int.
let swapped: Bool.
arr := Array new: 6.
arr at: 1 put: 64.
arr at: 2 put: 34.
arr at: 3 put: 25.
arr at: 4 put: 12.
arr at: 5 put: 22.
arr at: 6 put: 11.
n := arr size.
[   swapped := false.
    1 to: (n - 1) do: [ :i |
        ((arr at: i) > (arr at: i + 1))
        ifTrue: [
            let t: Int.
            t := arr at: i.
            arr at: i put: (arr at: i + 1).
            arr at: i + 1 put: t.
            swapped := true
            ]
    ].
    swapped ] whileTrue.
Console println: arr printString.
"=> (11 12 22 25 34 64 )"
```

Arrays are 1-based (following Smalltalk convention). The outer `whileTrue`
(without a colon — the block is both the condition and the body's container)
keeps running until no swaps occur in a pass.


## 3.10 Collection Operations (Map, Filter, Reduce)

picoceci arrays support the classic higher-order collection operations:

```picoceci
let numbers: Array.  
let evens: Array.
let doubled: Array.
let sum: Int.

numbers := #(1 2 3 4 5 6 7 8 9 10).

"Filter: select elements matching a predicate"
evens := numbers select: [ :n | n \\ 2 = 0 ].
Console println: evens printString.
"=> (2 4 6 8 10 )"

"Map: transform each element"
doubled := numbers collect: [ :n | n * 2 ].
Console println: doubled printString.
"=> (2 4 6 8 10 12 14 16 18 20 )"

"Reduce: fold into a single value"
sum := numbers inject: 0 into: [ :acc :n | acc + n ].
Console println: sum printString.
"=> 55"
```

The names come from Smalltalk tradition: `select:` is filter, `collect:` is map,
`inject:into:` is reduce (fold). If you know JavaScript's `.filter()`, `.map()`,
and `.reduce()`, you already understand these — the only difference is the
message syntax.

### Sum of Squares

```picoceci
let numbers: Array.
let sumOfSquares: Int.
numbers := #(1 2 3 4 5 6 7 8 9 10).
sumOfSquares := numbers inject: 0 into: [ :acc :n | acc + (n * n) ].
Console println: sumOfSquares printString.   "=> 385"
```


## 3.11 Dictionary Usage

```picoceci
let config: Any.
config := Dictionary new.
config at: #host put: '192.168.1.10'.
config at: #port put: 7001.
config at: #interval put: 5000.

Console println: (config at: #host).
Console println: (config at: #port) printString.

config keysAndValuesDo: [ :k :v |
    Console println: k printString , ' = ' , v printString
].
```

Dictionaries map symbols (or strings) to values. They are essential for
configuration, message payloads, and the kind of structured data that flows
between nodes in a sensor network.


## 3.12 Error Handling

picoceci uses Smalltalk-style exception handling:

```picoceci
"Catch an error"
[ let x: Int.
  x := 0.
  10 / x
] on: Error do: [ :err |
    Console println: 'Caught: ' , err messageText
].

"Ensure cleanup happens regardless of error"
[ file read ] ensure: [ file close ].
```

The `on:do:` pattern wraps a block and catches errors of the specified type. The
`ensure:` pattern guarantees that the cleanup block runs whether or not an error
occurred — picoceci's equivalent of `try/finally` in Java or `defer` in Go.


## 3.13 Chapter Summary

These patterns — recursion, iteration, higher-order functions, data structures,
error handling — are the bread and butter of programming in any language.
picoceci expresses them with remarkable concision. A factorial is four lines. A
bubble sort is fifteen. A closure factory is five. The message-passing syntax
keeps the *what* (the logic) visible and the *how* (the syntax machinery)
minimal.

This matters for embedded systems, where programs must be small enough to fit in
limited memory and clear enough for a maintenance programmer to understand months
or years after they were written. It matters even more when that maintenance
programmer might be an AI agent reading the code aboard a spacecraft.

---

\newpage

# Chapter 4: Concurrency and Channels

If Chapter 2 is about building individual objects and Chapter 3 is about the
patterns those objects can express, then Chapter 4 is about making objects *work
together at the same time*. This is where picoceci's Go influence becomes most
visible.


## 4.1 Tasks: Lightweight Concurrent Workers

A `Task` is picoceci's unit of concurrency. On the ESP32-S3, each Task maps to a
FreeRTOS task — a real, preemptively scheduled thread of execution with its own
stack. On the desktop, Tasks map to Go goroutines for testing.

```picoceci
Task spawn: [
    [ true ] whileTrue: [
        Console println: 'tick'.
        Task delay: 1000
    ]
] name: 'ticker'.
```

`Task spawn:` takes a block and starts it running concurrently. `Task delay:
1000` pauses the task for 1000 milliseconds without busy-waiting — it yields
control back to the FreeRTOS scheduler so other tasks can run.

You can control tasks after creation:

```picoceci
let worker: Any.
worker := Task spawn: [ ... ].
worker suspend.          "pause the task"
worker resume.           "resume it"
worker priority: 3.      "change its scheduling priority"
worker delete.           "terminate it"
```


## 4.2 Channels: Go-Style Communication

Channels are picoceci's primary mechanism for communication between tasks. If you
have used Go channels, you will feel immediately at home:

```picoceci
let ch: Channel<<Int>>.
ch := Channel new: 10.        "buffered channel, capacity 10"

"Sender task"
Task spawn: [
    1 to: 5 do: [ :i |
        ch <- i.               "send i to the channel"
        Task delay: 500
    ]
] name: 'sender'.

"Receiver task"
Task spawn: [
    5 timesRepeat: [
        let value: Int.
        value := <-ch.         "receive from the channel"
        Console println: 'Got: ' , value printString
    ]
] name: 'receiver'.
```

The `<-` operator sends to or receives from a channel. `ch <- 42` sends; `<-ch`
receives. If the channel is full, the sender blocks. If the channel is empty, the
receiver blocks. No locks, no mutexes, no shared mutable state.

Channels carry a type parameter. `Channel<<Int>>` only accepts integers; sending
a string to it raises a `TypeError` at the point of send, before any consumer
sees the bad data. Use `Channel<<Any>>` if you genuinely need mixed types.


## 4.3 Queues: Lower-Level FreeRTOS Primitives

Channels are built on top of FreeRTOS queues. You can also use queues directly:

```picoceci
let q: Queue<<String>>.
q := Queue new: 10.

Task spawn: [
    q send: 'hello'.
    q send: 'world'
].

Task spawn: [
    let msg: String.
    msg := q receive.
    Console println: msg.       "=> hello"
    msg := q receive.
    Console println: msg.       "=> world"
].
```

Queues support timeouts:

```picoceci
let result: Any.
result := q receive timeout: 5000.
(result = nil)
    ifTrue:  [ Console println: 'timed out' ]
    ifFalse: [ Console println: 'got: ' , result printString ].
```


## 4.4 Semaphores and Mutexes

For the rare case where you need to protect shared state, picoceci exposes
FreeRTOS semaphores:

```picoceci
let mutex: Any.
mutex := Semaphore mutex.

"In any task that needs exclusive access:"
mutex take.
"... critical section ..."
mutex give.
```

There are three kinds:

```picoceci
Semaphore new.             "binary semaphore"
Semaphore counting: 4.    "counting semaphore, max 4"
Semaphore mutex.           "recursive mutex with priority inheritance"
```

In practice, you should prefer channels over mutexes. Channels enforce a
communication pattern that is easier to reason about, easier to test, and less
prone to deadlocks. The Go community has a saying: *"Don't communicate by sharing
memory; share memory by communicating."* picoceci takes this advice seriously.


## 4.5 Timers

Software timers execute a block after a delay or at regular intervals:

```picoceci
"One-shot: fire once after 500ms"
let alarm: Any.
alarm := Timer after: 500 do: [
    Console println: 'time is up!'
].

"Periodic: fire every second"
let heartbeat: Any.
heartbeat := Timer every: 1000 do: [
    Console println: 'heartbeat'
].

heartbeat stop.   "pause"
heartbeat start.  "resume"
heartbeat reset.  "restart the interval"
```


## 4.6 Pattern: Producer-Consumer

The canonical concurrent pattern — one task produces data, another consumes it:

```picoceci
let readings: Channel<<Float>>.
readings := Channel new: 20.

"Producer: poll sensor every 5 seconds"
Task spawn: [
    let sensor: Any.
    let i2c: Any.
    i2c := I2C new: 0 sda: 21 scl: 22 speed: 400000.
    sensor := TempSensor new init: i2c address: 16r48.
    [ true ] whileTrue: [
        sensor poll.
        readings <- sensor celsius.
        Task delay: 5000
    ]
] name: 'temp-producer'.

"Consumer: log each reading"
Task spawn: [
    [ true ] whileTrue: [
        let temp: Float.
        temp := <-readings.
        Console println: 'Temperature: ' , temp printString , ' C'
    ]
] name: 'temp-logger'.
```

The producer and consumer are completely decoupled. They share no state. They
communicate only through the typed channel. Either can be replaced, restarted, or
duplicated without affecting the other.


## 4.7 Pattern: Fan-In (Multiple Producers, One Consumer)

Multiple sensor tasks report through a single channel:

```picoceci
let reportChan: Channel<<Any>>.
reportChan := Channel new: 30.

"Temperature sensor task"
Task spawn: [
    [ true ] whileTrue: [
        reportChan <- { #temp. readTemp }.
        Task delay: 5000
    ]
] name: 'temp'.

"Humidity sensor task"
Task spawn: [
    [ true ] whileTrue: [
        reportChan <- { #humidity. readHumidity }.
        Task delay: 10000
    ]
] name: 'humidity'.

"CO2 sensor task"
Task spawn: [
    [ true ] whileTrue: [
        reportChan <- { #co2. readCO2 }.
        Task delay: 15000
    ]
] name: 'co2'.

"Single consumer: log everything"
Task spawn: [
    [ true ] whileTrue: [
        let msg: Any.
        msg := <-reportChan.
        Console println: (msg at: 1) printString , ': ' , (msg at: 2) printString
    ]
] name: 'reporter'.
```

This fan-in pattern is the heartbeat of environmental monitoring on a spacecraft
node. Each sensor runs independently, on its own polling interval, and reports
through a shared channel. Adding a fourth sensor is a copy-paste of twelve lines.


## 4.8 Pattern: Threshold Alert

A task that watches a data stream and raises alerts when values exceed a limit:

```picoceci
let tempChan: Channel<<Float>>.
let alertChan: Channel<<String>>.
tempChan  := Channel new: 20.
alertChan := Channel new: 10.

"Monitor task"
Task spawn: [
    [ true ] whileTrue: [
        let temp: Float.
        temp := <-tempChan.
        (temp > 30.0)
            ifTrue: [
                alertChan <- ('ALERT: temperature ' , temp printString , 'C')
            ]
    ]
] name: 'temp-monitor'.

"Alert handler task"
Task spawn: [
    [ true ] whileTrue: [
        let alert: String.
        alert := <-alertChan.
        Console println: alert.
        "Could also: flash a warning LED, send a network message, etc."
    ]
] name: 'alert-handler'.
```


## 4.9 Pattern: Self-Healing Watchdog

A spacecraft cannot afford to have a sensor crash and stay crashed. This pattern
wraps a sensor loop in an error handler that restarts it automatically:

```picoceci
let i2c: Any.
i2c := I2C new: 0 sda: 21 scl: 22 speed: 400000.

[ true ] whileTrue: [
    [
        let sensor: Any.
        sensor := TempSensor new init: i2c address: 16r48.
        [ true ] whileTrue: [
            sensor poll.
            Console println: sensor printString.
            Task delay: 5000
        ]
    ] on: Error do: [ :err |
        Console println: 'Sensor error: ' , err messageText.
        Console println: 'Restarting in 3 seconds...'.
        Task delay: 3000
    ]
].
```

If the sensor throws any error — a garbled I²C read, a timeout, a corrupted
value — the outer loop catches it, logs it, waits three seconds, and creates a
fresh sensor. The node never stays dead. This is essential when the nearest repair
technician is a hundred thousand kilometres away.


## 4.10 Chapter Summary

picoceci's concurrency model combines three ideas:

1. **Tasks** — independent, preemptively scheduled workers (from FreeRTOS)
2. **Channels** — typed, bounded, blocking communication (from Go)
3. **Error boundaries** — `on:do:` handlers that catch and recover (from Smalltalk)

Together, these give you a programming model where each task does one thing well,
communicates only through well-defined channels, and can be tested, replaced, or
restarted without affecting the rest of the system.

This is exactly the model you want when building a network of cooperating MCUs
aboard a spacecraft. Each node is a task. Each wire between nodes is a channel.
Each node is wrapped in a watchdog that restarts it on failure. The system as a
whole is resilient because no single failure can bring it down.

---

\newpage

# Chapter 5: Toward the Mesh

This final chapter connects everything — the objects, the patterns, the
concurrency — to picoceci's ultimate purpose: programming resilient meshes of
microcontrollers for environments where failure is not an option.


## 5.1 The Hardware: ESP32-S3

picoceci targets the ESP32-S3-N16R8, a microcontroller from Espressif with:

- Dual-core Xtensa LX7 processor at 240 MHz
- 512 KB internal SRAM + 8 MB PSRAM
- Wi-Fi 802.11 b/g/n and Bluetooth 5.0
- USB, SPI, I²C, UART, GPIO, ADC
- SD card interface

The picoceci runtime — interpreter, bytecode VM, and standard library — fits
within 128 KB of RAM. The rest is available for your programs, FreeRTOS task
stacks, and data buffers. The SD card (up to 32 GB, FAT32 or LittleFS) provides
persistent storage for scripts, logs, and configuration.

picoceci compiles to the ESP32-S3 via TinyGo:

```bash
tinygo flash -target=esp32s3-generic -port=/dev/cu.usbmodem11201 \
    ./target/esp32s3 && tinygo monitor
```


## 5.2 Hardware Interfaces in picoceci

picoceci wraps the ESP32-S3's hardware peripherals as objects with
message-passing interfaces. Here are the key ones:

### GPIO

```picoceci
let led: Any. 
let button: Any.
led := GPIO pin: 2 direction: #output.
button := GPIO pin: 0 direction: #input pullup: true.

led high.
led low.
led toggle.

button read.                                  "=> true or false"
button waitForEdge: #rising timeout: 5000.    "block until press"
button onEdge: #falling do: [                 "interrupt handler"
    Console println: 'button pressed!'
].
```

### I²C

```picoceci
let i2c: Any.
i2c := I2C new: 0 sda: 21 scl: 22 speed: 400000.
i2c writeTo: 16r48 bytes: #[1 2 3].
let data: ByteArray.
data := i2c readFrom: 16r48 count: 4.
```

### UART

```picoceci
let uart: Any.
uart := UART new: 0 baud: 115200.
uart println: 'Hello from picoceci'.
let line: String.
line := uart readLine.
```

### SPI

```picoceci
let spi: Any.
spi := SPI new: 0 sck: 18 mosi: 23 miso: 19 cs: 5 speed: 1000000.
let result: ByteArray.
result := spi transfer: #[16r9F 0 0 0].
```

### SD Card

```picoceci
let file: Any.
file := File open: '/sdcard/log.txt' mode: #append.
file write: 'Temperature: 23.5C\n'.
file close.

let contents: String.
contents := File readAll: '/sdcard/config.pc'.
```

Every peripheral is an object. Every interaction is a message. The mental model
is uniform whether you are manipulating a string or toggling a GPIO pin.


## 5.3 A Complete Sensor Node

Here is a realistic environmental monitoring node — the kind that would be one
among hundreds aboard a spacecraft:

```picoceci
"EnvNode.pc — environmental monitoring node for Zone A, position 7"
import 'TempSensor'.
import 'HumiditySensor'.
import 'NetworkChannel'.

lenodeId: String.
let i2c: Any.
nodeId := 'zone-A-node-07'.
i2c    := I2C new: 0 sda: 21 scl: 22 speed: 400000.

let localReport: Channel<<Any>>. 
let netOut: Any.
localReport := Channel new: 30.
netOut      := NetworkChannel connectTo: '192.168.1.10' port: 7001.

"Temperature task — self-healing"
Task spawn: [
    [ true ] whileTrue: [
        [
            let s: Any.
            s := TempSensor new init: i2c address: 16r48.
            [ true ] whileTrue: [
                s poll.
                localReport <- { #temp. s celsius }.
                Task delay: 5000
            ]
        ] on: Error do: [ :e |
            Console println: 'temp error: ' , e messageText.
            Task delay: 3000
        ]
    ]
] name: 'temp'.

"Humidity task — self-healing"
Task spawn: [
    [ true ] whileTrue: [
        [
            let s: Any.
            s := HumiditySensor new init: i2c address: 16r44.
            [ true ] whileTrue: [
                s poll.
                localReport <- { #humidity. s percent }.
                Task delay: 10000
            ]
        ] on: Error do: [ :e |
            Console println: 'humidity error: ' , e messageText.
            Task delay: 3000
        ]
    ]
] name: 'humidity'.

"Reporter task — forward to zone aggregator"
Task spawn: [
    [ true ] whileTrue: [
        let msg: Any.
        msg := <-localReport.
        netOut <- { #node. nodeId. #sensor. (msg at: 1). #value. (msg at: 2) }
    ]
] name: 'reporter'.

Console println: nodeId , ' online'.
```

Each sensor runs in its own self-healing task. All data flows through the local
channel to the reporter, which forwards it to the zone aggregator over the
network. If a sensor crashes, it restarts. If the network is down, the channel
buffers messages. The node is resilient by design.


## 5.4 The Zone Aggregator

One node per zone collects readings from all nodes in that zone:

```picoceci
"ZoneAggregator.pc — collects readings from up to 16 sensor nodes"
import 'NetworkChannel'.

let incoming: Any. 
let readings: Any.
incoming := NetworkChannel listenOn: 7001.
readings := Dictionary new.

Task spawn: [
    [ true ] whileTrue: [
        let msg: Any.
        let nodeId: Any.
        let sensorType: Any.
        let value: Any.
        msg        := <-incoming.
        nodeId     := msg at: #node.
        sensorType := msg at: #sensor.
        value      := msg at: #value.
        readings at: nodeId put: (
            readings at: nodeId ifAbsent: [ Dictionary new ]
        ).
        (readings at: nodeId) at: sensorType put: value.
        Console println: 'Zone A: ' , nodeId printString ,
                         ' ' , sensorType printString ,
                         ' = ' , value printString
    ]
] name: 'aggregator'.
```


## 5.5 Canal: Capability-Based Security

__Deprecated._

picoceci runs on top of Canal, a capability-based microkernel for TinyGo. In the
Canal model, access to every resource — a GPIO pin, an I²C bus, a file, a network
socket — is represented as an *unforgeable capability token*. You can only use a
resource if you hold its capability. You cannot forge, guess, or steal
capabilities.

In picoceci, Canal capabilities look like ordinary objects:

```picoceci
let cap: Any.
cap := Canal capability: #uart0.
cap send: 'hello\n' asBytes.
let response: ByteArray.
response := cap receive: 64.
cap close.
```

But underneath, the Canal kernel enforces strict isolation. A picoceci task cannot
access a resource it was not explicitly given a capability for. This means:

- A compromised sensor node cannot read data from other nodes.
- A buggy humidity script cannot corrupt the temperature sensor's I²C bus.
- A malicious program cannot access the flight control system.

picoceci objects map naturally onto Canal capabilities. A `TempSensor` object *is*
a capability object — you can pass it to another task, compose it into a larger
object, or revoke it, and the security properties are enforced at the kernel
level.

Capability delegation lets you share access in controlled ways:

```picoceci
"Give the logger task read-only access to the UART"
let readCap: Any.
readCap := Canal capability: #uart0.
cap delegate: loggerTask.
```

For spacecraft applications, this is not a nice-to-have. A software fault in one
subsystem must not propagate to another. Canal's capability model, combined with
picoceci's message-passing isolation, provides the foundation for that guarantee.


## 5.6 The Mesh Vision

Put the pieces together and you begin to see the mesh:

```
   Zone A                    Zone B
┌──────────────┐         ┌──────────────┐
│  node-A-01   │         │  node-B-01   │
│  node-A-02   │         │  node-B-02   │
│  node-A-03   │◄───────►│  node-B-03   │
│     ...      │         │     ...      │
│  node-A-16   │         │  node-B-16   │
│              │         │              │
│  aggregator  │◄───────►│  aggregator  │
└──────┬───────┘         └──────┬───────┘
       │                        │
       └────────┬───────────────┘
                │
         ┌──────┴───────┐
         │   Central    │
         │  Dashboard   │
         └──────────────┘
```

Each box is an ESP32-S3 running picoceci on Canal. Each node monitors one or
more physical sensors. Each zone has an aggregator that collects and summarizes
readings. The aggregators talk to each other and to a central dashboard. If any
node fails, its watchdog restarts it. If an aggregator fails, the nodes buffer
their readings until it comes back. If an entire zone goes dark, the other zones
continue operating and the central dashboard degrades gracefully.

This is not a single point of failure. It is a nervous system — hundreds of
small, independent, cooperating nodes, each one simple enough to understand
completely, each one cheap enough to replace, each one robust enough to recover
from its own failures.

### Properties of the Mesh

| Property | How picoceci Achieves It |
|---|---|
| **Resilience** | Self-healing watchdog loops around every sensor task |
| **Isolation** | Canal capabilities prevent cross-node interference |
| **Decoupling** | Channels separate producers from consumers |
| **Simplicity** | Each node's program is 30–50 lines of readable code |
| **Scalability** | Adding a node is deploying one more ESP32-S3 |
| **Inspectability** | Message-passing makes data flow explicit and traceable |
| **Replaceability** | Structural typing means any conforming object fits |


## 5.7 Why picoceci for Spacecraft

The choice of language matters enormously in safety-critical environments. Here
is why picoceci's particular blend of features is well-suited to the problem:

**Smalltalk's message-passing** maps naturally onto a distributed network. Each
node is an object. Each inter-node message is a message. The local programming
model (send a message, get a response) is the same as the network programming
model. There is no cognitive gap between "writing code for one MCU" and "writing
code for a mesh of MCUs."

**Go's composition** eliminates the fragile hierarchies that make object-oriented
systems hard to maintain. When you compose a TempSensor into an AlertingSensor,
you get a flat, predictable, locally-understandable object. No hidden method
resolution, no diamond problems, no action-at-a-distance.

**Go's channels** give you safe concurrency without shared mutable state. Each
task is isolated. Each channel is typed and bounded. Deadlocks are possible but
structurally discouraged by the pattern of small tasks communicating through
explicit channels.

**Smalltalk's error handling** (`on:do:` and `ensure:`) provides simple, local
error boundaries. A crashed sensor loop is caught, logged, and restarted — all
in five lines of code.

**The small footprint** (under 128 KB) means picoceci runs on commodity
hardware. An ESP32-S3 costs a few dollars. A mesh of a hundred nodes costs a few
hundred dollars. You can prototype a spacecraft monitoring system on a student's
budget.

**The interpreted nature** means code can be updated without reflashing firmware.
Store new picoceci scripts on the SD card, restart the node, and the new code
runs. For a spacecraft in transit, where you cannot physically swap hardware, the
ability to push software updates is essential.


## 5.8 Influence Map

picoceci stands on the shoulders of remarkable languages:

| Influence | What picoceci Borrows |
|---|---|
| **Smalltalk** | Message syntax, blocks, everything-is-an-object, symbols |
| **Go** | Structural interfaces, composition, channels, goroutine-like tasks |
| **Self** | Prototype-like object composition |
| **Pharo** | Modern Smalltalk idioms; proof the ideas are still vital |
| **Erlang** | Philosophy of many small isolated processes communicating by message |
| **MicroPython** | Proof that a high-level language can run well on MCUs |

What picoceci does that none of these does in quite the same way is combine
Smalltalk's approachability with Go's composability and put the result inside a
sub-128-KB MCU runtime designed for networked, fault-tolerant systems.


## 5.9 The Road Ahead

picoceci is a starting point. The language specification is complete. The lexer,
parser, tree-walking interpreter, and bytecode VM are implemented. The TinyGo
target runs on real ESP32-S3 hardware. Canal integration provides capability-based
security.

What remains is the ecosystem:

- **Developer tooling** — debugger, profiler, VS Code extension
- **Network protocols** — standardized inter-node message formats
- **Supervisory abstractions** — Kubernetes-inspired service discovery, health
  checks, and automatic failover for MCU meshes
- **A successor language** — picoceci is intentionally Smalltalk-flavored, but the
  ideas it embodies could be expressed in a syntax more natural to developers
  raised on C-family languages, while remaining concise enough for AI agents to
  read and generate

The dream is a spacecraft — a real one — where every corridor's air quality is
monitored by a matchbox-sized device running a picoceci program, where that
device talks to hundreds of others like it, and where the network heals itself
when things go wrong. The technology exists. The language exists. What remains is
the building.

Come build something.


## 5.10 Chapter Summary

- picoceci targets the ESP32-S3 via TinyGo, fitting in under 128 KB of RAM.
- Hardware peripherals (GPIO, I²C, SPI, UART, SD card) are wrapped as objects.
- Canal provides capability-based security: no resource access without a token.
- The mesh is hundreds of independent nodes, each running simple picoceci tasks,
  communicating through channels and network connections.
- Self-healing watchdog patterns make each node resilient to failure.
- Structural typing and composition make nodes replaceable and composable.
- The interpreted runtime allows software updates without reflashing firmware.

---

\newpage

# Appendix A: Language Quick Reference

## Literals

```picoceci
42              "integer"
3.14            "float"
16rFF           "hex integer (255)"
2r1010          "binary integer (10)"
'hello'         "string"
#hello          "symbol"
$A              "character"
true false nil  "boolean and nil"
#(1 2 3)        "array literal"
#[255 0 128]    "byte array"
```

## Variables and Assignment

```picoceci
let x: Int.
let y: Float.
let name: String.
x := 42.
y := 3.14.
name := 'picoceci'.
let anything: Any.     "Any accepts any type"
anything := 'works'.
```

## Messages

```picoceci
sensor reading.                    "unary"
count + 1.                         "binary"
Console println: 'hi'.             "keyword (one arg)"
dict at: #key put: value.          "keyword (two args)"
led toggle; blink; off.            "cascade (same receiver)"
```

## Objects

```picoceci
object Foo {
    let slot1: Int.
    let slot2: String.
    init: a and: b [ slot1 := a. slot2 := b ]
    sum            [ ^slot1 ]
}
let f: Foo.
f := Foo new init: 3 and: 'bar'.
```

## Composition

```picoceci
object Bar {
    compose Foo.
    doubled [ ^(super sum) * 2 ]
}
```

## Interfaces

```picoceci
interface Readable {
    reading
}
```

## Blocks

```picoceci
[ 1 + 1 ] value.                "=> 2 (zero args)"
[ :x | x * 2 ] value: 5.       "=> 10 (one arg)"
[ :x :y | x + y ] value: 3 value: 4.  "=> 7 (two args)"
```

## Control Flow

```picoceci
x > 0 ifTrue: [ ... ] ifFalse: [ ... ].
1 to: 10 do: [ :i | ... ].
[ condition ] whileTrue: [ ... ].
5 timesRepeat: [ ... ].
```

## Concurrency

```picoceci
let ch: Channel<<Int>>.
ch := Channel new: 10.
Task spawn: [ ch <- 42 ] name: 'sender'.
Task spawn: [
    let v: Int.
    v := <-ch.
    Console println: v printString
] name: 'receiver'.
```

## Error Handling

```picoceci
[ riskyOp ] on: Error do: [ :e | Console println: e messageText ].
[ file read ] ensure: [ file close ].
```

---

\newpage

# Appendix B: Comparison with Go and Smalltalk

| Feature | Smalltalk | Go | picoceci |
|---|---|---|---|
| Paradigm | OOP + message passing | Structural typing + composition | Both |
| Syntax | Keyword messages | C-family | Keyword messages |
| Inheritance | Class hierarchy | None (embedding) | None (`compose`) |
| Polymorphism | Subtype + duck typing | Structural interfaces | Structural interfaces |
| Closures | Blocks `[ ... ]` | `func() { ... }` | Blocks `[ ... ]` |
| Concurrency | Processes (some dialects) | Goroutines + channels | Tasks + channels |
| Error handling | Exception `on:do:` | `error` return values | Exception `on:do:` |
| Typing | Dynamic | Static | Static by declaration |
| Target | VM (desktop) | Native (any arch) | Bytecode VM (MCU) |
| Memory | GC | GC | Ref-counted / arena |

picoceci takes from Smalltalk: the syntax, the block model, the "everything is an
object" philosophy, the error handling pattern.

picoceci takes from Go: structural interfaces, composition over inheritance,
typed channels, no class hierarchy, the emphasis on simplicity and readability.

The result is a language that reads like Smalltalk, reasons like Go, and runs on a
microcontroller.

---

\newpage

# Appendix C: Running the Examples

All examples in this book can be run in the picoceci REPL or from source files.

## Desktop (Go host)

```bash
git clone https://github.com/kristofer/picoceci
cd picoceci
go build ./...

# REPL
./picoceci repl

# Run a file (AST interpreter)
./picoceci run testdata/programs/hello.pc

# Run a file (bytecode VM)
./picoceci run-vm testdata/programs/fib.pc
```

## ESP32-S3 (TinyGo target)

```bash
tinygo flash -target=esp32s3-generic \
    -port=/dev/cu.usbmodem11201 \
    ./target/esp32s3 && tinygo monitor
```

## File Conventions

- `.pc` — picoceci source files (current convention)
- `.ceci` — alternate extension (also supported)
- One object or interface per file, named after the object (e.g., `Counter.pc`)
- Modules are loaded with `import 'Counter'.`
- The SD card path for libraries is `/sdcard/picoceci/libs/`

---

*picoceci is MIT licensed. Contributions welcome at
https://github.com/kristofer/picoceci*

*This document was produced for the picoceci project by Kris Younger,
ZipCode Wilmington.*
