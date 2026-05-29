# Lesson 09 — Collections

**Goal:** Use `Array` for fixed-size sequences, `OrderedCollection` for growing
lists, `Dictionary` for key→value maps, and the powerful functional methods
(`collect:`, `select:`, `inject:into:`) that transform collections without
explicit loops.

---

## Reference

### Array — fixed-size sequence

An `Array` is created at a fixed size. Elements can be any type.

**Creating arrays:**

```picoceci
"Array literal — elements must be literals"
let primes: Array.
primes := #(2 3 5 7 11).

"Empty array of given size (all nil)"
let arr: Array.
arr := Array new: 5.

"Filled array"
let zeros: Array.
zeros := Array new: 5 withAll: 0.

"Small arrays from arguments"
let pair: Array.
pair := Array with: 'alpha' with: 'beta'.
```

**Accessing and mutating:**

```picoceci
primes at: 1.             "=> 2 (1-based)"
primes at: 3.             "=> 5"
arr at: 2 put: 99.        "set element"
primes size.              "=> 5"
primes first.             "=> 2"
primes last.              "=> 11"
```

**Key Array messages:**

| Message | Returns | Description |
|---------|---------|-------------|
| `size` | Int | Number of elements |
| `at: n` | Object | Get element (1-based) |
| `at: n put: v` | v | Set element |
| `first` / `last` | Object | First/last element |
| `includes: v` | Bool | Membership test |
| `indexOf: v` | Int | First index of v (0 if absent) |
| `copyFrom: s to: e` | Array | Sub-array |
| `reversed` | Array | New reversed array |
| `do: b` | self | Iterate |
| `doWithIndex: b` | self | Iterate with `[:v :i | ...]` |
| `collect: b` | Array | Map — transform each element |
| `select: b` | Array | Filter — keep elements where block is true |
| `reject: b` | Array | Inverse filter |
| `detect: b` | Object | First element where block is true |
| `detect: b ifNone: b2` | Object | First match, or default if none |
| `inject: init into: b` | Object | Fold / reduce |
| `allSatisfy: b` | Bool | True if block is true for all elements |
| `anySatisfy: b` | Bool | True if block is true for any element |
| `printString` | String | Human-readable |

### collect: — transform every element

```picoceci
let doubled: Array.
doubled := #(1 2 3 4 5) collect: [ :n | n * 2 ].
Console println: doubled printString.
"=> (2 4 6 8 10 )"
```

### select: and reject: — filter

```picoceci
let evens: Array.
evens := #(1 2 3 4 5 6) select: [ :n | n \\ 2 = 0 ].
Console println: evens printString.
"=> (2 4 6 )"

let odds: Array.
odds := #(1 2 3 4 5 6) reject: [ :n | n \\ 2 = 0 ].
Console println: odds printString.
"=> (1 3 5 )"
```

### inject:into: — fold / reduce

```picoceci
let sum: Int.
sum := #(1 2 3 4 5) inject: 0 into: [ :acc :n | acc + n ].
Console println: sum printString.
"=> 15"

let product: Int.
product := #(1 2 3 4 5) inject: 1 into: [ :acc :n | acc * n ].
Console println: product printString.
"=> 120"
```

### detect: — find first match

```picoceci
let first: Any.
first := #(4 7 2 9 1) detect: [ :n | n > 5 ].
Console println: first printString.
"=> 7"

first := #(1 2 3) detect: [ :n | n > 10 ] ifNone: [ 'none found' ].
Console println: first.
"=> none found"
```

---

### OrderedCollection — dynamic list

`OrderedCollection` grows and shrinks as needed. Use it when the number of
elements is not known in advance.

```picoceci
let oc: OrderedCollection.
oc := OrderedCollection new.
oc add: 'alpha'.
oc add: 'beta'.
oc add: 'gamma'.
Console println: oc size printString.     "=> 3"
Console println: (oc at: 2).              "=> beta"
oc removeFirst.
Console println: (oc at: 1).             "=> beta"
```

| Message | Description |
|---------|-------------|
| `new` | Empty collection |
| `add: v` | Append |
| `addFirst: v` | Prepend |
| `remove: v` | Remove first occurrence |
| `removeFirst` | Remove and return first |
| `removeLast` | Remove and return last |
| `at: n` | Access by index (1-based) |
| `size` | Count |
| `includes: v` | Membership |
| `do: b` | Iterate |
| `collect: b` | Map (returns OrderedCollection) |
| `select: b` | Filter |
| `inject: init into: b` | Fold |
| `asArray` | Convert to fixed Array |

---

### Dictionary — key → value map

```picoceci
let d: Any.
d := Dictionary new.
d at: #name put: 'picoceci'.
d at: #version put: 3.

Console println: (d at: #name).              "=> picoceci"
Console println: (d at: #version) printString. "=> 3"
Console println: (d includesKey: #author) printString. "=> false"

let v: Any.
v := d at: #missing ifAbsent: [ 'default' ].
Console println: v.   "=> default"
```

| Message | Description |
|---------|-------------|
| `new` | Empty dictionary |
| `at: key put: val` | Set entry |
| `at: key` | Get (error if absent) |
| `at: key ifAbsent: b` | Get or evaluate block |
| `includesKey: key` | Key exists? |
| `removeKey: key` | Remove |
| `keys` | Array of keys |
| `values` | Array of values |
| `size` | Entry count |
| `do: b` | Iterate values |
| `keysAndValuesDo: b` | Iterate `[:k :v | ...]` pairs |

---

## Worked Examples

**word_frequency.pc** — count word occurrences with a Dictionary:

```picoceci
let text: String.
text := 'the cat sat on the mat the cat'.
let words: Array.
words := text substrings.
let freq: Any.
freq := Dictionary new.
words do: [ :w |
    let count: Int.
    count := freq at: w asSymbol ifAbsent: [ 0 ].
    freq at: w asSymbol put: count + 1.
].
Console println: (freq at: #the) printString.   "=> 3"
Console println: (freq at: #cat) printString.   "=> 2"
Console println: (freq at: #sat) printString.   "=> 1"
```

**pipeline.pc** — chaining collection operations:

```picoceci
let numbers: Array.
numbers := #(1 2 3 4 5 6 7 8 9 10).

"Sum of squares of even numbers"
let result: Int.
result := (numbers select: [ :n | n \\ 2 = 0 ])
          collect: [ :n | n * n ];
          inject: 0 into: [ :acc :n | acc + n ].

"Not chainable directly — need intermediate variables:"
let evens: Array.
evens := numbers select: [ :n | n \\ 2 = 0 ].
let squares: Array.
squares := evens collect: [ :n | n * n ].
let sum: Int.
sum := squares inject: 0 into: [ :acc :n | acc + n ].
Console println: sum printString.   "=> 220"
```

---

## Exercises

### Exercise 9.1 — Array statistics

Write `ex09_stats.pc`. Given the array `#(4 8 15 16 23 42)`:
1. Print its size
2. Print the sum (using `inject:into:`)
3. Print the maximum value (using `inject:into:` with an initial value of `0`)
4. Print all values greater than 15 (using `select:`)
5. Print each value doubled (using `collect:`)

```
"Expected output:
6
108
42
(16 23 42 )
(8 16 30 32 46 84 )"
```

### Exercise 9.2 — OrderedCollection as a queue

Write `ex09_queue.pc`. Use an `OrderedCollection` as a simple FIFO queue:
enqueue five strings with `add:`, then dequeue them one at a time with
`removeFirst`, printing each. Finally print whether the collection `isEmpty`
(hint: `size = 0`).

```
"Expected output:
first
second
third
fourth
fifth
empty: true"
```

### Exercise 9.3 — Phone book

Write `ex09_phonebook.pc`. Create a `Dictionary` mapping three names (as
Symbols) to phone numbers (as Strings). Then:
1. Look up one name and print the number
2. Check whether a name that does not exist returns a default `'not found'`
3. Print all names using `keys` and `do:`

```
"Expected output (adapt to your own data):
555-1234
not found
Alice
Bob
Carol"
```

### Exercise 9.4 — Histogram

Write `ex09_histogram.pc`. Given the array `#(3 1 4 1 5 9 2 6 5 3 5)`:
1. Count how many times each value appears (use a `Dictionary`)
2. Print a histogram line for each unique value in ascending order using a
   `to:do:` from 1 to 9:
   - Print `'n: ***'` where the `*` count equals the frequency
   - Skip values with frequency 0

```
"Expected output:
1: **
2: *
3: **
4: *
5: ***
6: *
9: *"
```

### Exercise 9.5 — Top-N filter

Write `ex09_topn.pc`. Given the array `#(17 4 33 8 22 11 45 3 29 7)`:
1. Sort a copy into descending order using `inject:into:` to build an
   `OrderedCollection` (insert each element in the right position)
2. Print the top 3 values

```
"Expected output:
45
33
29"
```

Hint: build an `OrderedCollection`, then iterate the original array,
inserting each value at the correct position by scanning the current contents.
