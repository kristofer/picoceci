# picoceci Standard Library Reference

Version: 0.1-draft

This document describes all objects available in the picoceci standard library.
Built-in modules are compiled into the runtime binary and do not require SD card access.

---

## Module: `core` (auto-imported)

The `core` module is always available.

---

### `Boolean`

The abstract interface satisfied by `true` and `false`.

| Message | Return | Description |
|---|---|---|
| `ifTrue: aBlock` | `nil` or block result | Evaluate block if true |
| `ifFalse: aBlock` | `nil` or block result | Evaluate block if false |
| `ifTrue: t ifFalse: f` | block result | Conditional |
| `& aBool` | Boolean | Logical AND |
| `\| aBool` | Boolean | Logical OR |
| `not` | Boolean | Logical NOT |
| `xor: aBool` | Boolean | Exclusive OR |
| `printString` | String | `'true'` or `'false'` |
| `= anObject` | Boolean | Identity comparison |

---

### `Integer`

Arbitrary-precision integers (practical max ~2^62 on 64-bit, ~2^30 on 32-bit).

| Message | Return | Description |
|---|---|---|
| `+ n` | Integer or Float | Addition |
| `- n` | Integer or Float | Subtraction |
| `* n` | Integer or Float | Multiplication |
| `/ n` | Integer or Float | Division (exact → Float if needed) |
| `// n` | Integer | Integer (floor) division |
| `\\ n` | Integer | Modulo |
| `** n` | Integer or Float | Exponentiation |
| `= n` | Boolean | Equality |
| `~= n` | Boolean | Inequality |
| `< n` | Boolean | Less than |
| `> n` | Boolean | Greater than |
| `<= n` | Boolean | Less or equal |
| `>= n` | Boolean | Greater or equal |
| `abs` | Integer | Absolute value |
| `negated` | Integer | Negation |
| `max: n` | Integer | Maximum of self and n |
| `min: n` | Integer | Minimum of self and n |
| `between: lo and: hi` | Boolean | Range check |
| `gcd: n` | Integer | Greatest common divisor |
| `lcm: n` | Integer | Least common multiple |
| `factorial` | Integer | Factorial (use with caution on MCU) |
| `isPrime` | Boolean | Primality test |
| `sqrt` | Float | Square root |
| `asFloat` | Float | Convert to float |
| `printString` | String | Decimal representation |
| `printString: base` | String | Representation in given base |
| `to: n do: aBlock` | self | Iterate from self to n (inclusive) |
| `to: n by: step do: aBlock` | self | Iterate with step |
| `timesRepeat: aBlock` | self | Repeat n times |
| `bitAnd: n` | Integer | Bitwise AND |
| `bitOr: n` | Integer | Bitwise OR |
| `bitXor: n` | Integer | Bitwise XOR |
| `bitShift: n` | Integer | Logical shift (positive=left) |
| `bitInvert` | Integer | Bitwise NOT |

---

### `Float`

IEEE-754 double-precision floating-point.

| Message | Return | Description |
|---|---|---|
| `+ n` | Float | Addition |
| `- n` | Float | Subtraction |
| `* n` | Float | Multiplication |
| `/ n` | Float | Division |
| `= n` | Boolean | Equality (exact) |
| `~= n` | Boolean | Inequality |
| `< > <= >=` | Boolean | Comparison |
| `abs` | Float | Absolute value |
| `negated` | Float | Negation |
| `sqrt` | Float | Square root |
| `floor` | Integer | Floor |
| `ceiling` | Integer | Ceiling |
| `rounded` | Integer | Round to nearest |
| `truncated` | Integer | Truncate toward zero |
| `sin` | Float | Sine (radians) |
| `cos` | Float | Cosine |
| `tan` | Float | Tangent |
| `ln` | Float | Natural logarithm |
| `log` | Float | Base-10 logarithm |
| `exp` | Float | e^self |
| `isNaN` | Boolean | Test for NaN |
| `isInfinite` | Boolean | Test for infinity |
| `asInteger` | Integer | Convert (truncated) |
| `printString` | String | Decimal representation |

---

### `Character`

A Unicode code point.

| Message | Return | Description |
|---|---|---|
| `asInteger` | Integer | Unicode code point |
| `asString` | String | Single-character string |
| `asUppercase` | Character | Uppercase variant |
| `asLowercase` | Character | Lowercase variant |
| `isLetter` | Boolean | Is alphabetic |
| `isDigit` | Boolean | Is decimal digit |
| `isAlphaNumeric` | Boolean | Is letter or digit |
| `isSpace` | Boolean | Is whitespace |
| `= aChar` | Boolean | Equality |
| `< aChar` | Boolean | Code-point order |
| `printString` | String | `$A` notation |

---

### `String`

Immutable UTF-8 string.

| Message | Return | Description |
|---|---|---|
| `size` | Integer | Number of characters (code points) |
| `byteSize` | Integer | Number of UTF-8 bytes |
| `at: index` | Character | Character at 1-based index |
| `copyFrom: start to: stop` | String | Substring |
| `, aString` | String | Concatenation |
| `= aString` | Boolean | Equality |
| `~= aString` | Boolean | Inequality |
| `< aString` | Boolean | Lexicographic less-than |
| `includesSubString: s` | Boolean | Substring search |
| `startsWith: s` | Boolean | Prefix test |
| `endsWith: s` | Boolean | Suffix test |
| `indexOf: char` | Integer | First occurrence (1-based, 0 if not found) |
| `indexOf: char startingAt: n` | Integer | First occurrence from n |
| `replaceAll: old with: new` | String | Global replace |
| `trimSeparators` | String | Strip leading/trailing whitespace |
| `lines` | Array | Split by newline |
| `substrings` | Array | Split by whitespace |
| `substrings: aChar` | Array | Split by character |
| `asUppercase` | String | Uppercase |
| `asLowercase` | String | Lowercase |
| `reversed` | String | Reversed |
| `asInteger` | Integer | Parse decimal integer |
| `asFloat` | Float | Parse float |
| `asSymbol` | Symbol | Intern as symbol |
| `asBytes` | ByteArray | UTF-8 bytes |
| `printString` | String | Self (for display) |
| `do: aBlock` | self | Iterate over characters |
| `readStream` | ReadStream | Stream over characters |

---

### `Symbol`

Like `String` but interned.  All `String` messages are available plus:

| Message | Return | Description |
|---|---|---|
| `asString` | String | Convert to mutable string |
| `= aSymbol` | Boolean | Identity comparison (fast) |

---

### `Array`

Fixed-size heterogeneous array.

| Message | Return | Description |
|---|---|---|
| `new: size` | Array | All elements nil |
| `new: size withAll: val` | Array | All elements val |
| `with: a` | Array | 1-element array |
| `with: a with: b` | Array | 2-element array |
| `with: a with: b with: c` | Array | 3-element array |
| `size` | Integer | Number of elements |
| `at: index` | Object | 1-based access |
| `at: index put: val` | val | Set element |
| `first` | Object | Element at 1 |
| `last` | Object | Element at size |
| `includes: val` | Boolean | Membership test |
| `indexOf: val` | Integer | First index of val (0 if absent) |
| `copyFrom: start to: stop` | Array | Sub-array |
| `reversed` | Array | New reversed array |
| `do: aBlock` | self | Iterate |
| `doWithIndex: aBlock` | self | Iterate with index |
| `collect: aBlock` | Array | Map |
| `select: aBlock` | Array | Filter |
| `reject: aBlock` | Array | Inverse filter |
| `detect: aBlock` | Object | First matching |
| `detect: aBlock ifNone: b` | Object | First matching or default |
| `inject: init into: aBlock` | Object | Fold |
| `with: other collect: aBlock` | Array | Zip-map |
| `allSatisfy: aBlock` | Boolean | Universal quantifier |
| `anySatisfy: aBlock` | Boolean | Existential quantifier |
| `asSortedCollection` | OrderedCollection | Sorted copy |
| `printString` | String | Human-readable |

---

### `ByteArray`

Fixed-size array of bytes (0–255).

| Message | Return | Description |
|---|---|---|
| `new: size` | ByteArray | All zeros |
| `new: size withAll: byte` | ByteArray | Fill with byte |
| `size` | Integer | Byte count |
| `at: index` | Integer | Byte at 1-based index |
| `at: index put: byte` | byte | Set byte |
| `copyFrom: start to: stop` | ByteArray | Sub-array |
| `, aByteArray` | ByteArray | Concatenation |
| `do: aBlock` | self | Iterate |
| `asString` | String | Interpret as UTF-8 |
| `printString` | String | Hex representation |

---

## Module: `io`

### `Console` / `Transcript`

Interchangeable names for the primary text output stream (UART0 on MCU, stdout on desktop).

| Message | Description |
|---|---|
| `print: anObject` | Print `anObject printString` (no newline) |
| `println: anObject` | Print with trailing newline |
| `nl` | Print newline |
| `show: anObject` | Alias for `print:` |
| `store: anObject` | Print object in parseable form |

---

### `ReadStream`

```picoceci
| s: Any |
s := ReadStream on: #(1 2 3 4 5).
s next.         "=> 1"
s next.         "=> 2"
s atEnd.        "=> false"
s peek.         "=> 3"
s position.     "=> 2"
s position: 0.  "rewind"
```

| Message | Return | Description |
|---|---|---|
| `on: collection` | ReadStream | Create |
| `on: coll from: start to: stop` | ReadStream | Windowed |
| `next` | Object | Read and advance |
| `next: n` | Array/String | Read n elements |
| `peek` | Object | Read without advancing |
| `skip: n` | self | Advance by n |
| `atEnd` | Boolean | End test |
| `position` | Integer | Current index |
| `position: n` | self | Seek |
| `do: aBlock` | self | Iterate remaining |

---

### `WriteStream`

```picoceci
| s: Any |
s := WriteStream on: String new.
s nextPutAll: 'Hello'.
s nextPut: $,.
s nextPutAll: ' world'.
s contents.   "=> 'Hello, world'"
```

| Message | Return | Description |
|---|---|---|
| `on: collection` | WriteStream | Create (appends) |
| `on: coll from: start to: stop` | WriteStream | Windowed |
| `nextPut: element` | element | Write one |
| `nextPutAll: collection` | self | Write all |
| `nl` | self | Write newline |
| `contents` | collection | Return accumulated result |
| `size` | Integer | Number of elements written |
| `reset` | self | Reset to start |

---

## Module: `collections`

### `OrderedCollection`

Dynamically resizable array.

| Message | Description |
|---|---|
| `new` | Empty collection |
| `add: element` | Append |
| `addFirst: element` | Prepend |
| `remove: element` | Remove first occurrence |
| `removeFirst` | Remove and return first |
| `removeLast` | Remove and return last |
| `size` | Element count |
| `at: index` | Access by index |
| `includes: element` | Membership test |
| `do: aBlock` | Iterate |
| `collect: aBlock` | Map |
| `select: aBlock` | Filter |
| `inject: init into: aBlock` | Fold |
| `asSortedCollection` | Sorted copy |
| `asArray` | Fixed-size copy |
| `printString` | Human-readable |

---

### `Dictionary`

Hash map with symbol or string keys.

```picoceci
| d: Any |
d := Dictionary new.
d at: #name put: 'picoceci'.
d at: #version put: 1.
d at: #name.           "=> 'picoceci'"
d includesKey: #name.  "=> true"
```

| Message | Description |
|---|---|
| `new` | Empty dictionary |
| `at: key` | Look up key (error if absent) |
| `at: key put: value` | Set key |
| `at: key ifAbsent: aBlock` | Look up or evaluate block |
| `includesKey: key` | Key existence test |
| `removeKey: key` | Remove key |
| `keys` | Array of keys |
| `values` | Array of values |
| `size` | Entry count |
| `do: aBlock` | Iterate values |
| `keysAndValuesDo: aBlock` | Iterate pairs |
| `printString` | Human-readable |

---

### `Set`

Unordered collection of unique values.

| Message | Description |
|---|---|
| `new` | Empty set |
| `add: element` | Add (no-op if already present) |
| `remove: element` | Remove |
| `includes: element` | Membership |
| `size` | Count |
| `do: aBlock` | Iterate |
| `union: aSet` | Union |
| `intersection: aSet` | Intersection |
| `difference: aSet` | Difference |
| `asArray` | Array copy |

---

## Module: `task`

See `LANGUAGE_SPEC.md` §10 and `docs/freertos-bridge.md` for full API.

Summary:

- `Task` — FreeRTOS task wrapper
- `Queue` — FreeRTOS queue
- `Semaphore` — FreeRTOS binary / counting / mutex semaphore
- `Timer` — FreeRTOS software timer
- `Channel` — higher-level typed channel (built on Queue)

---

## Module: `sdcard`

See `docs/sdcard.md` for full API.

Summary:

- `File` — open, read, write, seek, close files on SD card
- `Directory` — list, create, remove directories
- `Path` — platform-independent path manipulation

---

## Module: `gpio`

```picoceci
| led pin |
led := GPIO pin: 2 direction: #output.
pin := GPIO pin: 0 direction: #input pullup: true.

led high.
led low.
led toggle.

pin read.            "=> true or false"
pin waitForEdge: #rising timeout: 1000.
pin onEdge: #falling do: [ Console println: 'fell' ].
```

| Message | Description |
|---|---|
| `GPIO pin: n direction: #input` | Configure input |
| `GPIO pin: n direction: #input pullup: bool` | Input with pullup/pulldown |
| `GPIO pin: n direction: #output` | Configure output |
| `gpio high` | Drive high |
| `gpio low` | Drive low |
| `gpio toggle` | Toggle state |
| `gpio read` | Read current level (Boolean) |
| `gpio waitForEdge: #rising timeout: ms` | Block until edge or timeout |
| `gpio onEdge: #rising do: aBlock` | Attach interrupt handler |

---

## Module: `uart`

```picoceci
| uart: Any |
uart := UART new: 0 baud: 115200.
uart println: 'ready'.
| line: Any |
line := uart readLine.
```

| Message | Description |
|---|---|
| `UART new: port baud: rate` | Open UART |
| `uart print: anObject` | Write printString |
| `uart println: anObject` | Write printString + newline |
| `uart write: bytes` | Write ByteArray |
| `uart read: n` | Read n bytes (blocking) |
| `uart readLine` | Read until newline |
| `uart close` | Release UART |

---

## Module: `i2c`

```picoceci
| i2c |
i2c := I2C new: 0 sda: 21 scl: 22 speed: 400000.
i2c writeTo: 16r48 bytes: #[1 2 3].
| data: Any |
data := i2c readFrom: 16r48 count: 4.
```

| Message | Description |
|---|---|
| `I2C new: bus sda: pin scl: pin speed: hz` | Open I²C bus |
| `i2c writeTo: addr bytes: byteArray` | Write bytes |
| `i2c readFrom: addr count: n` | Read n bytes |
| `i2c writeReadTo: addr write: b read: n` | Combined write-then-read |
| `i2c close` | Release bus |

---

## Module: `spi`

```picoceci
| spi: Any |
spi := SPI new: 0 sck: 18 mosi: 23 miso: 19 cs: 5 speed: 1000000.
| result: Any |
result := spi transfer: #[16r9F 0 0 0].
```

| Message | Description |
|---|---|
| `SPI new: bus sck: pin mosi: pin miso: pin cs: pin speed: hz` | Open SPI bus |
| `spi transfer: byteArray` | Full-duplex transfer, returns ByteArray |
| `spi write: byteArray` | Write only |
| `spi read: n` | Read n bytes |
| `spi close` | Release bus |

---

## Module: `canal`

See `LANGUAGE_SPEC.md` §13.2 and the Canal repository for full semantics.

| Message | Description |
|---|---|
| `Canal capability: #name` | Acquire named capability |
| `cap send: byteArray` | Write to capability |
| `cap receive: n` | Read n bytes from capability |
| `cap close` | Release capability |
| `cap delegate: taskObject` | Transfer ownership |

---

*End of stdlib.md*
