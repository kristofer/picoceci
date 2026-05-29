# picoceci — Beginner's Curriculum

Ten lessons that take you from your first `Console println:` to objects that
compose, tasks that run in parallel, and collections that transform data. Every
exercise runs on your laptop with the desktop `picoceci` binary — no hardware
required.

---

## Before You Start

### Install picoceci (desktop)

```sh
git clone https://github.com/kristofer/picoceci
cd picoceci
go build ./cmd/picoceci
```

This produces a `picoceci` binary. Put it somewhere on your `$PATH`.

### Two ways to run picoceci

**Interactive REPL** — type expressions and see results immediately:

```sh
picoceci repl
```

```
picoceci> 3 + 4.
7
picoceci> 'hello' reversed.
'olleh'
picoceci> .help
meta-commands: .globals  .help  .version
```

**Run a file** — write a `.pc` file and execute it:

```sh
picoceci run hello.pc
```

The REPL is great for experimenting. Files are great for saving your work and
submitting exercises.

---

## Lessons

| # | Title | Key concepts |
|---|-------|-------------|
| [01](01-hello.md) | Hello, picoceci | REPL, `Console println:`, literals, `printString` |
| [02](02-messages.md) | Messages | Unary, binary, keyword; precedence; cascade |
| [03](03-variables.md) | Variables and Types | `let`, typed declarations, zero values |
| [04](04-blocks.md) | Blocks | Closures, `value`, `timesRepeat:`, `whileTrue:` |
| [05](05-control-flow.md) | Control Flow | Booleans, `ifTrue:ifFalse:`, `to:do:` |
| [06](06-objects.md) | Objects and Methods | `object`, slots, methods, `init`, `self` |
| [07](07-composition.md) | Composition | `compose`, override, `super`, interfaces |
| [08](08-strings.md) | Strings and Streams | String API, `WriteStream`, `ReadStream` |
| [09](09-collections.md) | Collections | Array, OrderedCollection, Dictionary |
| [10](10-concurrency.md) | Concurrency | `Task`, `Channel`, `Queue`, error handling |

---

## How Exercises Work

Each lesson ends with 3–5 exercises. Every exercise asks you to write a `.pc`
file and produce specific output when you run it with `picoceci run`.

The expected output is shown as a comment inside the exercise description, like:

```
"Expected output:
hello from picoceci
3628800"
```

Run your solution:

```sh
picoceci run my_solution.pc
```

If the output matches, you're done.

---

## Style Notes

- End every statement with `.`
- Comments are `"double-quoted"` — they never nest
- Declare every variable with `let` before using it
- Prefer specific types (`Int`, `String`, `Bool`) over `Any` when you know the type
- `Console println:` prints its argument followed by a newline; use `printString`
  to convert numbers and other objects to strings first
