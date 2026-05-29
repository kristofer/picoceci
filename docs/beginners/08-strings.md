# Lesson 08 — Strings and Streams

**Goal:** Work confidently with picoceci's `String` API — slicing, searching,
transforming — and use `WriteStream` to build strings incrementally and
`ReadStream` to consume collections element by element.

---

## Reference

### String basics

Strings are immutable and 1-based indexed.

```picoceci
let s: String.
s := 'spacecraft'.
Console println: s size printString.          "=> 10"
Console println: (s at: 1) printString.       "=> $s"
Console println: (s copyFrom: 1 to: 5).       "=> space"
Console println: s reversed.                  "=> tfarecaps"
Console println: s asUppercase.               "=> SPACECRAFT"
```

### Key String messages

| Message | Returns | Example |
|---------|---------|---------|
| `size` | Int | `'hello' size` → `5` |
| `at: n` | Char | `'abc' at: 2` → `$b` |
| `copyFrom: s to: e` | String | `'hello' copyFrom: 2 to: 4` → `'ell'` |
| `, other` | String | `'foo' , 'bar'` → `'foobar'` |
| `= other` | Bool | `'a' = 'a'` → `true` |
| `< other` | Bool | `'abc' < 'abd'` → `true` |
| `reversed` | String | `'abc' reversed` → `'cba'` |
| `asUppercase` | String | `'hi' asUppercase` → `'HI'` |
| `asLowercase` | String | `'HI' asLowercase` → `'hi'` |
| `trimSeparators` | String | `'  hi  ' trimSeparators` → `'hi'` |
| `startsWith: s` | Bool | `'hello' startsWith: 'hel'` → `true` |
| `endsWith: s` | Bool | `'hello' endsWith: 'llo'` → `true` |
| `includesSubString: s` | Bool | `'hello' includesSubString: 'ell'` → `true` |
| `indexOf: char` | Int | `'hello' indexOf: $l` → `3` (0 if not found) |
| `replaceAll: old with: new` | String | `'aabbcc' replaceAll: 'bb' with: 'XX'` → `'aaXXcc'` |
| `asInteger` | Int | `'42' asInteger` → `42` |
| `asFloat` | Float | `'3.14' asFloat` → `3.14` |
| `asSymbol` | Symbol | `'done' asSymbol` → `#done` |
| `lines` | Array | `'a\nb\nc' lines` → `(a  b  c )` |
| `substrings` | Array | `'a b c' substrings` → `(a  b  c )` |
| `substrings: char` | Array | `'a,b,c' substrings: $,` → `(a  b  c )` |
| `do: aBlock` | self | Iterate over each character |
| `printString` | String | Returns self (for display) |

### Character literals

A character literal is written `$` followed by the character: `$A`, `$z`, `$0`,
`$,`, `$ ` (space).

```picoceci
let c: Char.
c := $A.
Console println: c asString.          "=> A"
Console println: c asInteger printString.   "=> 65"
Console println: c asLowercase asString.   "=> a"
```

### Building strings with WriteStream

When you need to assemble a string from many pieces, `WriteStream` is more
efficient than repeated `,` concatenation (which creates a new string each time):

```picoceci
let ws: Any.
ws := WriteStream on: String new.
ws nextPutAll: 'Hello'.
ws nextPut: $,.
ws nextPutAll: ' world'.
ws nextPut: $!.
Console println: ws contents.   "=> Hello, world!"
```

| Message | Description |
|---------|-------------|
| `WriteStream on: String new` | Create an empty string stream |
| `nextPut: aChar` | Append one character |
| `nextPutAll: aString` | Append a string |
| `nl` | Append a newline character |
| `contents` | Return the assembled string |
| `size` | Number of characters written |
| `reset` | Clear and start over |

### Reading collections with ReadStream

`ReadStream` lets you consume a collection step by step:

```picoceci
let rs: Any.
rs := ReadStream on: #(10 20 30 40 50).
Console println: rs next printString.    "=> 10"
Console println: rs next printString.    "=> 20"
Console println: rs peek printString.    "=> 30 (does not advance)"
Console println: rs atEnd printString.   "=> false"
rs skip: 2.
Console println: rs atEnd printString.   "=> true (consumed all 5)"
```

You can also stream over a string:

```picoceci
let rs: Any.
rs := ReadStream on: 'hello'.
[ rs atEnd ] whileFalse: [
    Console println: rs next asString.
].
"prints h e l l o (one per line)"
```

| Message | Description |
|---------|-------------|
| `ReadStream on: collection` | Create stream |
| `next` | Read and advance |
| `next: n` | Read n elements as array or string |
| `peek` | Read without advancing |
| `skip: n` | Advance by n |
| `atEnd` | True if no more elements |
| `position` | Current index (0-based) |
| `position: n` | Seek to position |

---

## Worked Examples

**palindrome.pc** — check if a string is a palindrome:

```picoceci
let isPalindrome: Any.
isPalindrome := [ :s | s = s reversed ].

Console println: (isPalindrome value: 'racecar') printString.   "=> true"
Console println: (isPalindrome value: 'hello') printString.     "=> false"
Console println: (isPalindrome value: 'level') printString.     "=> true"
```

**word_count.pc** — count words in a sentence:

```picoceci
let sentence: String.
sentence := 'the quick brown fox jumps over the lazy dog'.
let words: Array.
words := sentence substrings.
Console println: words size printString.   "=> 9"
```

**csv_builder.pc** — use WriteStream to produce CSV:

```picoceci
let ws: Any.
ws := WriteStream on: String new.
let headers: Array.
headers := #('name' 'age' 'role').
let i: Int.
i := 1.
[ i <= headers size ] whileTrue: [
    (i > 1) ifTrue: [ ws nextPut: $, ].
    ws nextPutAll: (headers at: i).
    i := i + 1.
].
Console println: ws contents.   "=> name,age,role"
```

---

## Exercises

### Exercise 8.1 — String statistics

Write `ex08_stats.pc`. Given the string `'Hello, picoceci!'`:
1. Print its length
2. Print it in uppercase
3. Print it reversed
4. Print whether it starts with `'Hello'`
5. Print the index of the character `$,`

```
"Expected output:
16
HELLO, PICOCECI!
!icecocip ,olleH
true
6"
```

### Exercise 8.2 — Caesar cipher

Write `ex08_caesar.pc`. Implement a Caesar cipher that shifts each letter by
`13` positions (ROT13). Non-letter characters pass through unchanged. Use a
`WriteStream` to build the result.

For a letter `$A`–`$Z`, shifting by 13: `$A` → `$N`, `$N` → `$A`.
For lowercase `$a`–`$z` similarly.

Hint: `c asInteger` gives the code point; `String with: char` makes a
one-character string; work with the modular arithmetic of offsets from `$A`
(65) and `$a` (97).

Test with `'Hello, World!'` → `'Uryyb, Jbeyq!'`.

```
"Expected output:
Uryyb, Jbeyq!"
```

### Exercise 8.3 — Word reversal

Write `ex08_words.pc`. Given a sentence `'the cat sat on the mat'`:
1. Split into words using `substrings`
2. Reverse each word individually using `reversed`
3. Join them back with spaces using a `WriteStream`
4. Print the result

```
"Expected output:
eht tac tas no eht tam"
```

### Exercise 8.4 — Tokeniser

Write `ex08_tokenise.pc`. Use `ReadStream` to step through the string
`'10 + 20 * 3'` character by character and collect "tokens": sequences of
digits become number tokens, operator characters (`+`, `*`, `-`, `/`) become
operator tokens, spaces are skipped.

Print each token on its own line.

```
"Expected output:
10
+
20
*
3"
```

### Exercise 8.5 — Run-length encoding

Write `ex08_rle.pc`. Encode a string using run-length encoding: repeated
consecutive characters become `<count><char>`. If a character appears only
once, just emit the character (no count prefix).

Example: `'aaabbbccddddee'` → `'3a3b2c4d2e'`

Use a `WriteStream` for the output.

```
"Expected output:
3a3b2c4d2e"
```
