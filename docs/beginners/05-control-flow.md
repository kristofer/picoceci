# Lesson 05 — Control Flow

**Goal:** Write conditional logic with `ifTrue:`, `ifFalse:`, and
`ifTrue:ifFalse:`, iterate with `to:do:` and `to:by:do:`, combine boolean
conditions, and understand how `^` returns from a block.

---

## Reference

### Boolean messages

`true` and `false` respond to these messages:

| Message | Returns | Meaning |
|---------|---------|---------|
| `ifTrue: aBlock` | block result or `nil` | Run block if true |
| `ifFalse: aBlock` | block result or `nil` | Run block if false |
| `ifTrue: t ifFalse: f` | one of the block results | Choose between two branches |
| `& aBool` | Boolean | Logical AND |
| `\| aBool` | Boolean | Logical OR |
| `not` | Boolean | Logical NOT |
| `xor: aBool` | Boolean | Exclusive OR |

### Conditionals

```picoceci
let x: Int.
x := 7.

x > 5
    ifTrue: [ Console println: 'big' ]
    ifFalse: [ Console println: 'small' ].
"prints: big"
```

`ifTrue:ifFalse:` is a single keyword message — both parts belong together.
Put them on separate indented lines for readability:

```picoceci
(x \\ 2 = 0)
    ifTrue:  [ Console println: 'even' ]
    ifFalse: [ Console println: 'odd' ].
```

### Nested conditionals

Because `ifTrue:ifFalse:` returns the value of the chosen block, you can nest
them. Use parentheses to keep precedence clear:

```picoceci
let score: Int.
score := 75.
let grade: String.
grade := (score >= 90)
    ifTrue:  [ 'A' ]
    ifFalse: [
        (score >= 80)
            ifTrue:  [ 'B' ]
            ifFalse: [
                (score >= 70)
                    ifTrue:  [ 'C' ]
                    ifFalse: [ 'F' ]
            ]
    ].
Console println: grade.   "=> C"
```

### Range iteration: to:do:

`to:do:` sends numbers from the receiver up to the argument, one at a time,
to the block:

```picoceci
1 to: 5 do: [ :i | Console println: i printString ].
"prints 1 2 3 4 5"
```

### to:by:do: — custom step

```picoceci
1 to: 10 by: 2 do: [ :i | Console println: i printString ].
"prints 1 3 5 7 9 (odd numbers)"

10 to: 1 by: -1 do: [ :i | Console println: i printString ].
"prints 10 9 8 ... 1 (countdown)"
```

### Combining booleans

```picoceci
let x: Int. x := 5.
let y: Int. y := 10.

(x > 0 & (y > 0))
    ifTrue: [ Console println: 'both positive' ].

(x > 10 | (y > 8))
    ifTrue: [ Console println: 'at least one is large' ].
```

**Important:** `&` and `|` are binary messages, so binary-message precedence
applies. Parenthesise each side if it contains a keyword message:

```picoceci
((x > 0) & (x < 10))   "correct"
x > 0 & x < 10          "wrong — parsed as x > (0 & x) < 10"
```

### Returning from inside a block: ^

`^` inside a block that is the body of a *method* exits the method. Inside a
plain block at the top level of a file, it exits the block evaluation.

At the top level you will mostly use it inside loops to break out early:

```picoceci
let found: Bool.
found := false.
1 to: 100 do: [ :i |
    (i * i = 49) ifTrue: [
        Console println: ('sqrt of 49 is ', i printString).
        found := true.
    ].
].
```

---

## Worked Examples

**fizzbuzz.pc** — classic FizzBuzz using `to:do:` and nested conditionals:

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

**gcd.pc** — greatest common divisor using `whileTrue:`:

```picoceci
let a: Int. a := 48.
let b: Int. b := 18.
[ b ~= 0 ] whileTrue: [
    let t: Int.
    t := b.
    b := a \\ b.
    a := t.
].
Console println: ('GCD: ', a printString).   "=> GCD: 6"
```

---

## Exercises

### Exercise 5.1 — Grade classifier

Write `ex05_grade.pc`. Declare a variable `score := 88` and print a letter
grade according to:
- 90–100 → A
- 80–89 → B
- 70–79 → C
- 60–69 → D
- below 60 → F

```
"Expected output for 88:
B"
```

Try changing `score` to 72, 55, and 95 and verify your conditionals work.

### Exercise 5.2 — Multiplication table

Write `ex05_times_table.pc`. Use nested `to:do:` loops to print the 5×5
multiplication table:

```
"Expected output:
1 2 3 4 5
2 4 6 8 10
3 6 9 12 15
4 8 12 16 20
5 10 15 20 25"
```

Hint: use `Console print:` for the numbers in a row and `Console nl` at the
end of each row.

### Exercise 5.3 — FizzBuzz extended

Write `ex05_fizzbuzz.pc`. Print FizzBuzz for numbers 1 through 30. Use the
classic rules: multiples of 3 → `Fizz`, multiples of 5 → `Buzz`, multiples of
both → `FizzBuzz`, otherwise the number itself.

### Exercise 5.4 — Collatz sequence

Write `ex05_collatz.pc`. Starting from `n := 27`, apply the Collatz rule
repeatedly until `n = 1`:
- if `n` is even: `n := n // 2`
- if `n` is odd: `n := n * 3 + 1`

Print each value of `n` including the starting value and the final `1`. At the
end print the total number of steps taken.

```
"Expected output begins:
27
82
41
124
...
1
steps: 111"
```

### Exercise 5.5 — Prime sieve (simple)

Write `ex05_primes.pc`. Print all prime numbers from 2 to 50 using trial
division: for each candidate `n`, check if any number from 2 to `n - 1`
divides it evenly.

```
"Expected output:
2
3
5
7
11
13
17
19
23
29
31
37
41
43
47"
```
