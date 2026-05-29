# Lesson 03 — Variables and Types

**Goal:** Declare variables with `let`, understand picoceci's type system, use
zero values, and see how the runtime enforces types at assignment.

---

## Reference

### Declaring variables

Every variable must be declared before use. picoceci v3 uses the `let` keyword:

```picoceci
let x: Int.
let name: String.
let ratio: Float.
let done: Bool.
let item: Any.
```

The `: Type` annotation is required for typed declarations. Each declaration
ends with `.`.

### Zero values

A declared-but-unassigned variable automatically holds its type's zero value:

| Type | Zero value |
|------|-----------|
| `Int` | `0` |
| `Float` | `0.0` |
| `Bool` | `false` |
| `String` | `''` (empty string) |
| `Char` | `$\0` (NUL) |
| `Symbol` | `#''` (empty symbol) |
| `Array` | `#()` (empty array) |
| `ByteArray` | `#[]` (empty byte array) |
| `Any` | `nil` |
| Any object type | `nil` |

```picoceci
let count: Int.
Console println: count printString.   "=> 0"

let flag: Bool.
Console println: flag printString.    "=> false"

let label: String.
Console println: label.               "=> (empty line)"
```

### Assigning values

Use `:=` to assign. The value's type must match the declaration:

```picoceci
let n: Int.
n := 42.

let greeting: String.
greeting := 'Hello!'.

let pi: Float.
pi := 3.14159.
```

### Type inference with :=

You can infer the type from the initial value:

```picoceci
let n := 42.         "inferred as Int"
let s := 'hello'.    "inferred as String"
let f := 3.14.       "inferred as Float"
```

Once inferred, the type is locked — assigning the wrong type later is still a
runtime error.

### The Any type

`Any` is the explicit escape hatch for truly dynamic values:

```picoceci
let x: Any.
x := 42.
x := 'now a string'.   "fine — Any accepts anything"
x := true.             "also fine"
```

Use `Any` sparingly. Specific types catch mistakes early.

### Type errors

Assigning a value of the wrong type raises a `TypeError` at runtime:

```picoceci
let count: Int.
count := 'oops'.   "TypeError: count expects Int, got String"
```

### Multiple declarations on one line

Separate multiple declarations with `.`:

```picoceci
let x: Int. let y: Int. let z: Int.
```

Or one per line (more readable for larger programs):

```picoceci
let width: Int.
let height: Int.
let area: Int.
```

### Type keywords

The built-in type keywords are:

| Keyword | What it holds |
|---------|--------------|
| `Int` | Whole numbers (63-bit on 64-bit platforms) |
| `Float` | Decimal numbers (IEEE-754 double) |
| `Bool` | `true` or `false` |
| `String` | Immutable text |
| `Char` | A single Unicode character |
| `Symbol` | An interned string constant |
| `Array` | Fixed-size collection of any values |
| `ByteArray` | Fixed-size collection of bytes (0–255) |
| `Any` | Accepts any value (dynamic) |
| `Nil` | Holds only `nil` |
| `ObjectName` | A specific user-defined object type |

---

## Worked Examples

**zero_values.pc** — observing zero values:

```picoceci
let i: Int.
let f: Float.
let b: Bool.
let s: String.
Console println: i printString.    "=> 0"
Console println: f printString.    "=> 0.0"
Console println: b printString.    "=> false"
Console println: ('empty: ', s).   "=> empty: "
```

**swap.pc** — swapping two values using a temp variable:

```picoceci
let a: Int.
let b: Int.
let tmp: Int.
a := 10.
b := 20.
tmp := a.
a := b.
b := tmp.
Console println: a printString.   "=> 20"
Console println: b printString.   "=> 10"
```

**accumulate.pc** — accumulating into a variable:

```picoceci
let total: Int.
total := 0.
total := total + 10.
total := total + 25.
total := total + 7.
Console println: total printString.   "=> 42"
```

---

## Exercises

### Exercise 3.1 — Temperature converter

Write `ex03_temp.pc`. Declare a `Float` variable `celsius` and assign `100.0`.
Compute the Fahrenheit equivalent using the formula `(celsius * 9.0 / 5.0) + 32.0`
and store it in a `Float` variable `fahrenheit`. Print both values.

```
"Expected output:
100.0
212.0"
```

### Exercise 3.2 — String building

Write `ex03_string.pc`. Declare separate `String` variables for a first name,
last name, and a greeting prefix. Then assemble a full greeting by concatenation
and print it.

```
"Expected output (example with 'Ada' and 'Lovelace'):
Hello, Ada Lovelace!"
```

### Exercise 3.3 — Counter with zero value

Write `ex03_counter.pc`. Declare a `counter` variable of type `Int`. Without
assigning it explicitly, print its value. Then add `5` to it twice and print
the value after each addition.

```
"Expected output:
0
5
10"
```

### Exercise 3.4 — Type inference

Write `ex03_infer.pc` that uses `:=` inference (no explicit type annotation)
for three variables: one integer, one float, one string. Print each variable
after assignment.

### Exercise 3.5 — The Any escape hatch

Write `ex03_any.pc`. Declare a variable `x` of type `Any`. Assign it three
different values in sequence: an integer, a string, and a boolean. Print
`x printString` after each assignment to show the value changing type.

```
"Expected output:
99
launch
true"
```
