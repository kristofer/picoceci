# Lesson 06 — Objects and Methods

**Goal:** Define your own objects with slots and methods, create instances with
`new`, use `init` for non-zero setup, send messages to your objects, and return
values from methods with `^`.

---

## Reference

### Defining an object

An `object` declaration is a named template for creating instances:

```picoceci
object Counter {
    let count: Int.

    inc [
        count := count + 1.
        ^self
    ]

    value [
        ^count
    ]
}
```

- **Slots** (`let count: Int.`) are instance variables. Every instance gets its
  own copy. Typed slots are automatically zero-initialised (`0` for `Int`,
  `''` for `String`, etc.).
- **Methods** are unary (`inc`, `value`), binary (`+`), or keyword (`at:`,
  `at:put:`).
- **`^`** returns a value from the method. Without `^`, the method returns `nil`.
- **`self`** refers to the current instance. Returning `self` allows chaining.

### Creating instances

```picoceci
let c: Counter.
c := Counter new.
```

`new` allocates a fresh instance with all slots at their zero values.

### The init method

Use `init` for slots that need a non-zero starting value. `new` calls `init`
automatically if defined:

```picoceci
object Temperature {
    let celsius: Float.
    let unit: String.

    init [
        celsius := 20.0.
        unit := 'C'.
    ]

    value [ ^celsius ]
    unitLabel [ ^unit ]
}

let t: Temperature.
t := Temperature new.
Console println: t value printString.    "=> 20.0"
Console println: t unitLabel.            "=> C"
```

### Methods with parameters

Methods take parameters using keyword syntax:

```picoceci
object Rectangle {
    let width: Float.
    let height: Float.

    width: w height: h [
        width := w.
        height := h.
        ^self
    ]

    area [ ^width * height ]

    printString [
        ^'Rectangle(', width printString, 'x', height printString, ')'
    ]
}

let r: Rectangle.
r := Rectangle new.
r width: 4.0 height: 3.0.
Console println: r area printString.      "=> 12.0"
Console println: r printString.           "=> Rectangle(4.0x3.0)"
```

### Overriding printString

Every object automatically responds to `printString` — by default it returns
something like `'a Counter'`. Override it to show useful information:

```picoceci
object Point {
    let x: Int.
    let y: Int.

    x: ax y: ay [ x := ax. y := ay. ^self ]
    x [ ^x ]
    y [ ^y ]

    printString [ ^'(', x printString, ', ', y printString, ')' ]
}
```

### Slot accessors

Slots are private — you access them only through methods. The convention is a
unary getter and a setter:

```picoceci
object Person {
    let name: String.
    let age: Int.

    name: aName [ name := aName. ^self ]
    name [ ^name ]

    age: n [ age := n. ^self ]
    age [ ^age ]

    printString [
        ^name, ' (age ', age printString, ')'
    ]
}
```

### Chaining with self

Returning `self` from setter methods enables chained calls:

```picoceci
let p: Person.
p := Person new.
p name: 'Ada'; age: 36.
Console println: p printString.   "=> Ada (age 36)"
```

The cascade `;` sends `age: 36` to the same receiver as `name: 'Ada'`.

### Local variables inside methods

Declare local variables with `let` inside the method body:

```picoceci
object MathHelper {
    hypotenuse: a b: b [
        let sumOfSquares: Float.
        sumOfSquares := (a * a) + (b * b).
        ^sumOfSquares sqrt
    ]
}
```

---

## Worked Examples

**stack.pc** — a LIFO stack object:

```picoceci
object Stack {
    let storage: Array.
    let top: Int.

    init [
        storage := Array new: 64.
        top := 0.
    ]

    push: v [
        top := top + 1.
        storage at: top put: v.
        ^self
    ]

    pop [
        let v: Any.
        v := storage at: top.
        top := top - 1.
        ^v
    ]

    peek    [ ^storage at: top ]
    isEmpty [ ^top = 0 ]
    size    [ ^top ]

    printString [ ^'Stack(size=', top printString, ')' ]
}

let s: Stack.
s := Stack new.
s push: 10; push: 20; push: 30.
Console println: s pop printString.     "=> 30"
Console println: s pop printString.     "=> 20"
Console println: s size printString.    "=> 1"
Console println: s isEmpty printString. "=> false"
```

---

## Exercises

### Exercise 6.1 — BankAccount

Write `ex06_bank.pc`. Define an object `BankAccount` with:
- A `Float` slot `balance` initialised to `0.0` in `init`
- A method `deposit: amount` that adds to the balance
- A method `withdraw: amount` that subtracts (do not check for overdraft yet)
- A method `balance` that returns the balance
- `printString` showing `'Balance: '` followed by the balance

Create an account, deposit 500.0, withdraw 120.0, and print it.

```
"Expected output:
Balance: 380.0"
```

### Exercise 6.2 — Temperature object

Write `ex06_temp.pc`. Define an object `Temperature` with:
- A `Float` slot `celsius` (init to 0.0)
- A method `celsius: n` that sets the value
- A method `celsius` that returns it
- A method `fahrenheit` that returns the Fahrenheit equivalent
- `printString` returning `'TempC'` followed by celsius, e.g. `'23.5°C'`

Print the celsius value and fahrenheit equivalent for `100.0`.

```
"Expected output:
100.0
212.0"
```

### Exercise 6.3 — Point

Write `ex06_point.pc`. Define a `Point` object with integer slots `x` and `y`.
Add:
- A method `x: ax y: ay` to set both at once
- A method `distanceTo: other` that returns the Euclidean distance to another Point (use `sqrt`)
- `printString` returning `'(x, y)'`

Create two points, print them both, and print the distance between them.

```
"Expected output (for (0,0) and (3,4)):
(0, 0)
(3, 4)
5.0"
```

### Exercise 6.4 — BankAccount overdraft check

Extend your `BankAccount` from 6.1. Add overdraft protection: if `withdraw:
amount` would make the balance negative, print `'Insufficient funds'` and do
not change the balance.

```
"Expected output for deposit(200), withdraw(50), withdraw(200):
Balance: 150.0
Insufficient funds
Balance: 150.0"
```

### Exercise 6.5 — Die (random not required)

Write `ex06_die.pc`. Define a `Die` object with a `sides: Int` slot.
Add a method `sides: n` setter and an `init` that sets `sides := 6`.
Add a `roll` method that cycles through values deterministically using a counter
slot: `(counter \\ sides) + 1`. Print ten rolls of a 6-sided die.

```
"Expected output:
1
2
3
4
5
6
1
2
3
4"
```
