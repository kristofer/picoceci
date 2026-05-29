# Lesson 07 — Composition

**Goal:** Reuse objects through `compose`, override composed methods, dispatch
to the original with `super`, define interfaces for structural typing, and
understand why picoceci uses composition instead of inheritance.

---

## Reference

### Why composition, not inheritance?

picoceci has no class hierarchy. Instead of `extends`, objects `compose` other
objects. This is the same idea as Go's struct embedding or the "favour
composition over inheritance" principle — you build complex objects by combining
simpler ones, not by specialising them.

Think of `compose` as: "this object has all the slots and methods of that
object, plus these additions or overrides."

### compose

```picoceci
object Animal {
    let name: String.

    name: n [ name := n. ^self ]
    name [ ^name ]

    speak [ Console println: (name, ' says ...') ]
}

object Dog {
    compose Animal.

    speak [ Console println: (name, ' says Woof!') ]
}
```

`Dog` gets all of `Animal`'s slots (`name`) and methods (`name:`, `name`). It
*overrides* `speak` with its own version.

```picoceci
let d: Dog.
d := Dog new.
d name: 'Rex'.
d speak.   "Rex says Woof!"
```

### super — calling the composed method

`super` sends the message to the *composed object's* version of the method.
It lets you extend behaviour rather than replace it entirely:

```picoceci
object LoggedCounter {
    compose Counter.

    inc [
        super inc.
        Console println: ('incremented to ', self value printString).
        ^self
    ]
}

let c: LoggedCounter.
c := LoggedCounter new.
c inc; inc; inc.
Console println: c value printString.
```

Output:
```
incremented to 1
incremented to 2
incremented to 3
3
```

### Multiple composition

An object can compose more than one other object. Methods from later `compose`
declarations override those from earlier ones when names conflict:

```picoceci
object Flyable {
    fly [ Console println: (self name, ' is flying') ]
}

object Swimmable {
    swim [ Console println: (self name, ' is swimming') ]
}

object Duck {
    compose Animal.
    compose Flyable.
    compose Swimmable.

    speak [ Console println: (name, ' says Quack!') ]
}
```

### Interfaces — structural typing

An `interface` declares a set of messages without providing implementations:

```picoceci
interface Measurable {
    area
    perimeter
}
```

Any object that responds to `area` and `perimeter` satisfies `Measurable` —
no explicit declaration needed. You can use the interface as a variable type:

```picoceci
let shape: Measurable.
shape := Rectangle new.         "Rectangle has area and perimeter"
shape := Circle new.            "So does Circle"
Console println: shape area printString.
```

### satisfies: — checking interface conformance

```picoceci
(myObject satisfies: Measurable)
    ifTrue: [ Console println: 'it is measurable' ].
```

### When to use interfaces vs. Any

- Use `Any` when you genuinely do not know the type and do not care about
  the messages it responds to.
- Use an interface type when you need the object to respond to a specific set
  of messages and want that contract enforced.
- Use a concrete object type (`Counter`, `Rectangle`) when you know exactly
  what you have.

---

## Worked Examples

**logged_counter.pc** — compose with override and super:

```picoceci
object Counter {
    let count: Int.
    inc   [ count := count + 1. ^self ]
    dec   [ count := count - 1. ^self ]
    value [ ^count ]
    printString [ ^'Counter(', count printString, ')' ]
}

object LoggedCounter {
    compose Counter.
    inc [
        super inc.
        Console println: 'incremented to ', self value printString.
        ^self
    ]
}

let c: LoggedCounter.
c := LoggedCounter new.
c inc; inc; inc.
Console println: c printString.
```

Output:
```
incremented to 1
incremented to 2
incremented to 3
Counter(3)
```

**shapes.pc** — interface and two implementations:

```picoceci
interface Shape {
    area
    perimeter
    printString
}

object Rectangle {
    let w: Float.
    let h: Float.
    w: x h: y [ w := x. h := y. ^self ]
    area      [ ^w * h ]
    perimeter [ ^2.0 * (w + h) ]
    printString [ ^'Rect(', w printString, 'x', h printString, ')' ]
}

object Circle {
    let r: Float.
    radius: x [ r := x. ^self ]
    area      [ ^3.14159 * r * r ]
    perimeter [ ^2.0 * 3.14159 * r ]
    printString [ ^'Circle(r=', r printString, ')' ]
}

let shapes: Array.
shapes := Array new: 2.
shapes at: 1 put: (Rectangle new w: 4.0 h: 3.0).
shapes at: 2 put: (Circle new radius: 5.0).

shapes do: [ :s |
    Console println: s printString.
    Console println: ('  area=', s area printString).
    Console println: ('  perimeter=', s perimeter printString).
].
```

---

## Exercises

### Exercise 7.1 — Audited bank account

Write `ex07_audit.pc`. Start with a `BankAccount` object from Lesson 06. Add
an `AuditedAccount` that composes `BankAccount` and overrides `deposit:` and
`withdraw:` to also print a log line:
- `'DEPOSIT: 100.0 -> 600.0'` (amount and new balance)
- `'WITHDRAW: 50.0 -> 550.0'`

```
"Expected output:
DEPOSIT: 500.0 -> 500.0
WITHDRAW: 120.0 -> 380.0
Balance: 380.0"
```

### Exercise 7.2 — Vehicle hierarchy

Write `ex07_vehicle.pc`. Define:
- `Vehicle` with a `speed: Float` slot, `accelerate: n` (adds n to speed),
  `brake: n` (subtracts n, minimum 0.0), and `speed` getter
- `Car` that composes `Vehicle` and adds a `fuel: Float` slot; `accelerate:`
  also consumes `0.1 * n` fuel; `refuel: n` adds fuel
- `printString` on each showing type, speed, and (for Car) fuel

Create a car, accelerate twice, brake once, and print it.

```
"Expected output (example):
Car(speed=25.0 fuel=12.3)"
```

### Exercise 7.3 — Printable interface

Write `ex07_printable.pc`. Define an interface `Printable` with one message:
`prettyPrint`. Write two objects — `Celsius` and `Fahrenheit` — both of which
compose an internal `Temperature` (Float slot, setter/getter) and implement
`prettyPrint` to print the temperature with the appropriate unit symbol.

Create one of each, store them in an `Array`, and call `prettyPrint` on each.

### Exercise 7.4 — super chain

Write `ex07_chain.pc`. Define three objects where each composes the previous:
`A`, `B` (composes `A`), `C` (composes `B`). Each defines a method `greet`
that calls `super greet` first (except `A`), then prints its own line. Create
a `C` and call `greet` — show that all three levels fire in order.

```
"Expected output:
Hello from A
Hello from B
Hello from C"
```

### Exercise 7.5 — satisfies:

Write `ex07_satisfies.pc`. Define an interface `Resettable` with one message
`reset`. Write two objects: `Counter` (with `reset` that sets `count := 0`)
and `Timer` (with `reset` that sets `elapsed := 0`). Write a block
`resetAll: col` that iterates a collection and calls `reset` on each element
only if it `satisfies: Resettable`. Test with an array containing a Counter, a
Timer, and a plain string (which should not be reset).
