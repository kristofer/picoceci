# Lesson 02 — Messages

**Goal:** Understand the three kinds of messages in picoceci, how precedence
works, how to use parentheses to control evaluation order, and how to send
multiple messages to the same object with cascade.

---

## Reference

### Three kinds of messages

picoceci is a *message-passing* language. You compute by sending messages to
objects. There are exactly three kinds of message, evaluated in this order:

#### 1. Unary messages — highest precedence

A single identifier sent with no arguments:

```picoceci
42 factorial.
'hello' reversed.
3.14 sqrt.
true not.
-7 abs.
```

Multiple unary messages chain left to right:

```picoceci
-7 abs printString.    "first abs, then printString"
```

#### 2. Binary messages — middle precedence

An operator symbol with one argument:

```picoceci
3 + 4.
10 - 3.
6 * 7.
10 / 4.
10 // 3.      "integer division → 3"
10 \\ 3.      "modulo → 1"
'hi' , ' there'.   "string concatenation → 'hi there'"
3 > 2.
5 <= 5.
3 = 3.
3 ~= 4.       "not equal"
```

Binary messages also chain left to right — there is *no arithmetic precedence*.
`2 + 3 * 4` is `(2 + 3) * 4 = 20`, not `2 + (3 * 4) = 14`.

#### 3. Keyword messages — lowest precedence

One or more keyword parts (identifier followed by `:`) each taking an argument:

```picoceci
arr at: 2.
dict at: #key put: 'value'.
1 to: 10 do: [ :i | Console println: i printString ].
```

Multiple keyword parts form a single message:
```picoceci
1 to: 5 by: 2 do: [ :i | Console println: i printString ].
"prints 1, 3, 5"
```

### Full precedence summary

```
unary  >  binary  >  keyword  >  assignment
```

When in doubt, add parentheses:

```picoceci
2 + 3 * 4.           "→ 20  (left-to-right, no * priority)"
2 + (3 * 4).         "→ 14  (parentheses force * first)"

'hello' size + 1.    "→ 6  (unary size first, then +)"
('hello' size) + 1.  "same, explicit"
```

### Cascade — send multiple messages to the same receiver

The semicolon `;` sends the next message to the *same receiver* as the
previous message, without repeating the receiver:

```picoceci
Console
    print: 'one';
    print: ' two';
    print: ' three';
    nl.
"prints: one two three"
```

This is equivalent to:

```picoceci
Console print: 'one'.
Console print: ' two'.
Console print: ' three'.
Console nl.
```

Cascade is particularly useful with collections:

```picoceci
let words: OrderedCollection.
words := OrderedCollection new.
words
    add: 'alpha';
    add: 'beta';
    add: 'gamma'.
```

### printString for numbers

Numbers are not strings. Use `printString` to convert before passing to
`Console println:`:

```picoceci
Console println: (3 + 4) printString.     "=> 7"
Console println: (10 // 3) printString.   "=> 3"
Console println: (10 \\ 3) printString.   "=> 1"
```

---

## Worked Examples

**precedence.pc** — message precedence in action:

```picoceci
"Unary before binary"
Console println: -7 abs printString.        "=> 7"

"Left-to-right binary — no math priority"
Console println: (2 + 3 * 4) printString.   "=> 20"
Console println: (2 + (3 * 4)) printString. "=> 14"

"Keyword message after binary"
Console println: (10 \\ 3 = 1) printString. "=> true"
```

**cascade.pc** — building output with cascade:

```picoceci
Console
    print: 'Result: ';
    print: (6 * 7) printString;
    nl.
"prints: Result: 42"
```

---

## Exercises

### Exercise 2.1 — Arithmetic expressions

Write `ex02_arithmetic.pc`. Print the results of these expressions (one per
line). Figure out the answers mentally first, then verify with picoceci:

1. `10 + 3 * 2` (left-to-right binary — what is it?)
2. `10 + (3 * 2)` (with parentheses)
3. `100 // 7`
4. `100 \\ 7`
5. `2 ** 8`

```
"Expected output:
26
16
14
2
256"
```

### Exercise 2.2 — String messages

Write `ex02_strings.pc`. For the string `'spacecraft'`, print:

1. Its length (`size`)
2. Its reversed form
3. Its uppercase form
4. The result of concatenating it with `' systems'`

```
"Expected output:
10
tfarecaps
SPACECRAFT
spacecraft systems"
```

### Exercise 2.3 — Unary chains

In `ex02_unary.pc`, demonstrate that unary messages chain left to right.
Start with the integer `-36` and send it:
1. `abs` then `printString` — one expression
2. `sqrt` then `printString` — what does `(-36 abs) sqrt` give?
3. `factorial` on `5` then `printString`

```
"Expected output:
36
6.0
120"
```

### Exercise 2.4 — Cascade output

Write `ex02_cascade.pc` that uses a *single cascade* to print the three lines:

```
"Expected output:
Name: Luna
Role: pilot
Status: ready"
```

Use `Console print:` and `Console nl` in a cascade — only one `Console`
expression in the whole file.

### Exercise 2.5 — Precedence puzzle

Without running picoceci first, work out what each of these expressions
evaluates to. Then write `ex02_puzzle.pc` to check your answers:

```picoceci
3 + 4 * 2.
3 + (4 * 2).
10 - 3 - 2.
10 - (3 - 2).
'abc' size * 2 + 1.
```

Write your predictions as comments in the file, one per expression, then print
the actual results.
