# Lesson 04 — Blocks

**Goal:** Understand blocks (closures) as first-class values, invoke them with
`value`, `value:`, and `value:value:`, use them for repetition, and see how
blocks capture variables from their enclosing scope.

---

## Reference

### What is a block?

A block is a piece of code wrapped in `[ ]`. It is a value — you can store it
in a variable and invoke it later.

```picoceci
[ Console println: 'hello' ]
```

Nothing happens when you write that. To run the block, send it `value`:

```picoceci
[ Console println: 'hello' ] value.
"prints: hello"
```

### Blocks with parameters

Parameters are declared with `:name` inside the opening bracket, separated by
spaces:

```picoceci
[ :x | x * x ] value: 5.         "=> 25"
[ :a :b | a + b ] value: 3 value: 4.   "=> 7"
```

| Arity | Message |
|-------|---------|
| 0 | `value` |
| 1 | `value: arg` |
| 2 | `value: a value: b` |
| 3 | `value: a value: b value: c` |
| n | `valueWithArguments: anArray` |

### Blocks return values

The value of the last expression in the block is its return value:

```picoceci
let square: Any.
square := [ :n | n * n ].
Console println: (square value: 7) printString.   "=> 49"
```

### Repetition: timesRepeat:

```picoceci
3 timesRepeat: [ Console println: 'ping' ].
"prints ping three times"
```

### Loops: whileTrue: and whileFalse:

`whileTrue:` sends `value` to the condition block; if it returns `true`, it
evaluates the body block, then repeats:

```picoceci
let i: Int.
i := 1.
[ i <= 5 ] whileTrue: [
    Console println: i printString.
    i := i + 1.
].
"prints 1 through 5"
```

`whileFalse:` is the complement — keep looping while the condition is `false`:

```picoceci
let n: Int.
n := 10.
[ n = 0 ] whileFalse: [
    Console println: n printString.
    n := n - 3.
].
"prints 10, 7, 4, 1"
```

### Blocks are closures

A block *closes over* the variables in the scope where it was written. It can
read and mutate them:

```picoceci
let count: Int.
count := 0.

let inc: Any.
inc := [ count := count + 1. count ].

Console println: (inc value) printString.   "=> 1"
Console println: (inc value) printString.   "=> 2"
Console println: (inc value) printString.   "=> 3"
```

Each call to `inc value` sees and updates the *same* `count` variable.

### Factory pattern — blocks returning blocks

Because blocks are values, you can return one from another:

```picoceci
let makeAdder: Any.
makeAdder := [ :n | [ :x | x + n ] ].

let add5: Any.
add5 := makeAdder value: 5.

Console println: (add5 value: 3) printString.    "=> 8"
Console println: (add5 value: 10) printString.   "=> 15"
```

---

## Worked Examples

**countdown.pc** — counting down with `whileTrue:`:

```picoceci
let n: Int.
n := 5.
[ n > 0 ] whileTrue: [
    Console println: n printString.
    n := n - 1.
].
Console println: 'liftoff!'.
```

Output:
```
5
4
3
2
1
liftoff!
```

**closure_counter.pc** — a closure-based counter factory:

```picoceci
let makeCounter: Any.
makeCounter := [ :start |
    let n: Int.
    n := start.
    [ n := n + 1. n ]
].

let c1: Any.
let c2: Any.
c1 := makeCounter value: 0.
c2 := makeCounter value: 100.

Console println: (c1 value) printString.   "=> 1"
Console println: (c1 value) printString.   "=> 2"
Console println: (c2 value) printString.   "=> 101"
Console println: (c1 value) printString.   "=> 3"
```

---

## Exercises

### Exercise 4.1 — Sum with whileTrue:

Write `ex04_sum.pc`. Use a `whileTrue:` loop to compute the sum of integers
from 1 to 100. Print the result.

```
"Expected output:
5050"
```

### Exercise 4.2 — Fizz-lite

Write `ex04_fizz.pc`. Use `timesRepeat:` with a counter variable to print the
numbers 1 through 15. (You do not need to check for Fizz/Buzz yet — that comes
in Lesson 05.)

```
"Expected output:
1
2
...
15"
```

### Exercise 4.3 — Block arithmetic

Write `ex04_blocks.pc`. Define three blocks stored in variables:
- `double` — takes a number and returns `n * 2`
- `square` — takes a number and returns `n * n`
- `sumOf` — takes two numbers and returns their sum

Use them to compute and print:
1. `double value: 7`
2. `square value: 9`
3. `sumOf value: 12 value: 30`

```
"Expected output:
14
81
42"
```

### Exercise 4.4 — Closure accumulator

Write `ex04_accum.pc`. Create a block `makeAccumulator` that takes a starting
value and returns a new block. Each call to the returned block takes an amount
and *adds it to the running total*, returning the new total.

```picoceci
"Usage:"
let acc: Any.
acc := makeAccumulator value: 0.
Console println: (acc value: 5) printString.    "=> 5"
Console println: (acc value: 10) printString.   "=> 15"
Console println: (acc value: 3) printString.    "=> 18"
```

```
"Expected output:
5
15
18"
```

### Exercise 4.5 — Power function

Write `ex04_power.pc`. Define a block `power` that takes a base and exponent
and computes base^exponent using `whileTrue:` (no `**` operator). Compute:

- `2 ^ 10`
- `3 ^ 5`
- `5 ^ 0` (any number to the zero power is 1)

```
"Expected output:
1024
243
1"
```
