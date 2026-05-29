# Lesson 01 — Hello, picoceci

**Goal:** Run your first picoceci program, understand how output works, and get
comfortable with the REPL and the `picoceci run` command.

---

## Reference

### The REPL

Start the interactive REPL with:

```sh
picoceci repl
```

Type any expression followed by `.` and press Enter. The REPL evaluates it and
prints the result. You can also use meta-commands (starting with `.`):

| Command | What it does |
|---------|-------------|
| `.help` | Show all meta-commands |
| `.globals` | List all globally visible names |
| `.version` | Print the picoceci version |

### Console

`Console` is the built-in output object. It writes to standard output (your
terminal).

| Message | Description |
|---------|-------------|
| `Console println: anObject` | Print `anObject printString` followed by newline |
| `Console print: anObject` | Print without newline |
| `Console nl` | Print just a newline |

### Literals

picoceci has six kinds of literal values you can type directly:

| Kind | Examples | Notes |
|------|---------|-------|
| Integer | `42`, `0`, `-7` | Whole numbers |
| Float | `3.14`, `1.5e-3` | Decimal numbers |
| String | `'hello'`, `'it''s'` | Single-quoted; `''` = literal `'` |
| Boolean | `true`, `false` | |
| Symbol | `#start`, `#at:put:` | Interned string constant |
| Nil | `nil` | The absence of a value |

### printString

Every object responds to `printString`, which returns a `String` describing the
object. You need it whenever you want to print a number or other non-string value:

```picoceci
Console println: 42 printString.       "=> 42"
Console println: 3.14 printString.     "=> 3.14"
Console println: true printString.     "=> true"
Console println: nil printString.      "=> nil"
Console println: #hello printString.   "=> hello"
Console println: 'world'.             "String already works directly"
```

### Comments

Anything between double-quotes is a comment and is ignored:

```picoceci
"This whole line is a comment"
Console println: 'hello'.  "this too"
```

Comments do not nest — the first `"` closes the nearest open one.

### Statements and the period

Each statement ends with `.`. In a file, every statement except the last
*should* end with `.` (the final period is optional before end of file).

---

## Worked Examples

**hello.pc** — the simplest possible program:

```picoceci
"Hello, World! — the simplest picoceci program"
Console println: 'Hello, picoceci!'.
```

Run it:
```sh
picoceci run hello.pc
```
Output:
```
Hello, picoceci!
```

**numbers.pc** — printing different kinds of values:

```picoceci
Console println: 42 printString.
Console println: 3.14 printString.
Console println: true printString.
Console println: 'a string'.
Console println: nil printString.
```

Output:
```
42
3.14
true
a string
nil
```

---

## Exercises

### Exercise 1.1 — Personal greeting

Write a program `ex01_greeting.pc` that prints three lines:
- Your name
- The year you were born (as a number)
- The sentence `picoceci is fun`

```
"Expected output (adapt to your own name and year):
Alice
1995
picoceci is fun"
```

### Exercise 1.2 — Number facts

Write a program `ex01_numbers.pc` that prints the following values, one per
line, by sending `printString` to each literal:

- The integer `1024`
- The float `2.718`
- The boolean `false`
- The symbol `#ready`
- `nil`

```
"Expected output:
1024
2.718
false
ready
nil"
```

### Exercise 1.3 — The REPL meta-commands

Start the REPL and run each meta-command. Write your observations as comments
in a file `ex01_repl_notes.pc`. The file does not need to produce any output —
it is just your notes.

What names does `.globals` list? How many are there?

### Exercise 1.4 — Exploring printString

Write a program `ex01_print.pc` that calls `printString` on five different
types of literal and prints each result. Then add one call to `Console print:`
(without `ln`) followed by `Console nl` to print a line in two steps.

```
"Expected output (one example):
99
hello
true
#done
3.14
two-step"
```

### Exercise 1.5 — Multiple lines with print:

Use only `Console print:` (no `println:`) and `Console nl` to reproduce this
output exactly:

```
"Expected output:
one two three
done"
```

Hint: you will need four `Console print:` calls and two `Console nl` calls.
