# Lesson 10 — Concurrency

**Goal:** Spawn tasks that run concurrently, communicate between them safely
with `Channel` and `Queue`, protect shared state with `Semaphore`, handle
errors in tasks, and understand how `TaskSupervisor` keeps tasks alive.

---

## Reference

### Tasks

A `Task` runs a block in its own goroutine (a lightweight concurrent thread).
All tasks share the same picoceci interpreter, but they run concurrently:

```picoceci
Task spawn: [
    Console println: 'hello from the task'
] name: 'greeter'.
```

`spawn:name:` returns immediately. The block runs concurrently in the
background. Give every task a descriptive name — it appears in error messages.

**The main program is also a task.** All tasks run until they finish or the
whole program exits.

### Duration — sleeping

```picoceci
Duration ms: 500.   "pause the current task for 500 milliseconds"
```

Use this to yield time to other tasks or wait between operations.

### Channels — typed, buffered communication

`Channel` is the idiomatic way to pass values between tasks. The `<-` operator
sends and receives:

```picoceci
let ch: Channel<<Int>>.
ch := Channel new: 10.     "buffer capacity 10"

"Producer task"
Task spawn: [
    1 to: 5 do: [ :i |
        ch <- i.
        Duration ms: 100.
    ].
] name: 'producer'.

"Consumer — receive 5 values"
1 to: 5 do: [ :i |
    let v: Int.
    v := <-ch.
    Console println: v printString.
].
```

| Syntax | Meaning |
|--------|---------|
| `ch <- value` | Send `value` into `ch` (blocks if full) |
| `<-ch` | Receive from `ch` (blocks if empty) |
| `Channel new: capacity` | Create buffered channel |

### Queue — lower-level communication

`Queue` is a FreeRTOS-style queue with explicit `send:` and `receive` messages:

```picoceci
let q: Any.
q := Queue new: 32.
q send: 'hello'.
let msg: Any.
msg := q receive.
Console println: msg.   "=> hello"
```

| Message | Description |
|---------|-------------|
| `Queue new: size` | Create queue with given capacity |
| `q send: item` | Enqueue (blocks if full) |
| `q send: item timeout: ms` | Enqueue with timeout |
| `q receive` | Dequeue (blocks if empty) |
| `q receive timeout: ms` | Dequeue with timeout |
| `q count` | Current number of items |

### Semaphore — mutual exclusion

Use a **mutex semaphore** to protect shared state from concurrent modification:

```picoceci
let mu: Any.
mu := Semaphore mutex.

"Both tasks share 'counter' — protect all reads and writes:"
mu critical: [
    counter := counter + 1.
].
```

The `critical:` message takes a mutex semaphore and a block, acquiring the
lock before running the block and releasing it afterward.

| Message | Description |
|---------|-------------|
| `Semaphore new` | Binary semaphore |
| `Semaphore mutex` | Mutex (initially unlocked) |
| `Semaphore counting: n` | Counting semaphore, max n |
| `sem take` | Acquire (blocks if 0) |
| `sem give` | Release |
| `sem critical: b` | Acquire, run block, release |

### Error handling in tasks

Wrap risky code in `on:do:` to catch errors:

```picoceci
Task spawn: [
    [
        "risky work here"
        Error signal: 'something went wrong'.
    ] on: Error do: [ :e |
        Console println: ('task error: ', e messageText).
    ].
] name: 'guarded'.
```

Without error handling, an unhandled error in a task prints to stderr and
kills that task — other tasks continue running.

### TaskSupervisor — automatic restart

`TaskSupervisor` restarts a task automatically if it crashes:

```picoceci
TaskSupervisor supervise: [
    "This block will restart on any error"
    [ true ] whileTrue: [
        Console println: 'working'.
        Duration ms: 1000.
    ].
] name: 'worker'.
```

The block stops being supervised when it exits *without* an error (a clean
return). Use `supervise:name:maxRestarts:` to limit restarts:

```picoceci
TaskSupervisor supervise: [
    Console println: 'attempting...'.
    Error signal: 'failed'.
] name: 'limited' maxRestarts: 3.
"restarts 3 times, then gives up"
```

### The producer–consumer pattern

The most common concurrency pattern: one task generates data, another
consumes it. A `Channel` or `Queue` sits between them:

```picoceci
let work: Any.
work := Queue new: 64.

"Producer"
Task spawn: [
    #('alpha' 'beta' 'gamma' 'done') do: [ :item |
        work send: item.
    ].
] name: 'producer'.

"Consumer"
Task spawn: [
    [ true ] whileTrue: [
        let item: Any.
        item := work receive.
        (item = 'done') ifTrue: [ ^nil ].
        Console println: ('processing: ', item).
    ].
] name: 'consumer'.

Duration ms: 200.   "give tasks time to run"
```

### Waiting for tasks to finish

picoceci does not have built-in task join. The idiomatic way to synchronise is
to send a sentinel value through the channel:

```picoceci
let done: Channel<<Bool>>.
done := Channel new: 1.

Task spawn: [
    "... do work ..."
    done <- true.
] name: 'worker'.

<-done.   "block until worker signals completion"
Console println: 'worker finished'.
```

---

## Worked Examples

**ticker.pc** — two tasks running in parallel:

```picoceci
Task spawn: [
    1 to: 5 do: [ :i |
        Console println: ('A: tick ', i printString).
        Duration ms: 100.
    ].
] name: 'task-a'.

Task spawn: [
    1 to: 5 do: [ :i |
        Console println: ('B: tick ', i printString).
        Duration ms: 150.
    ].
] name: 'task-b'.

Duration ms: 1000.   "wait for both to finish"
```

**shared_counter.pc** — mutex protecting a shared variable:

```picoceci
let counter: Int.
counter := 0.
let mu: Any.
mu := Semaphore mutex.
let done: Channel<<Bool>>.
done := Channel new: 2.

Task spawn: [
    100 timesRepeat: [
        mu critical: [ counter := counter + 1 ].
    ].
    done <- true.
] name: 'incrementer-a'.

Task spawn: [
    100 timesRepeat: [
        mu critical: [ counter := counter + 1 ].
    ].
    done <- true.
] name: 'incrementer-b'.

<-done. <-done.   "wait for both tasks"
Console println: counter printString.   "=> 200"
```

---

## Exercises

### Exercise 10.1 — Hello from tasks

Write `ex10_hello.pc`. Spawn three tasks named `'a'`, `'b'`, and `'c'`. Each
task prints `'hello from <name>'` and then signals a shared done channel.
The main program waits for all three before printing `'all done'`.

```
"Expected output (order of first 3 lines may vary):
hello from a
hello from b
hello from c
all done"
```

### Exercise 10.2 — Pipeline

Write `ex10_pipeline.pc`. Create a two-stage pipeline using two queues:

- **Stage 1** (producer task): sends the integers 1–10 into `stage1`
- **Stage 2** (transform task): reads from `stage1`, squares each value, sends
  to `stage2`; sends `-1` when done
- **Main**: reads from `stage2` until it sees `-1`, printing each squared value

```
"Expected output:
1
4
9
16
25
36
49
64
81
100"
```

### Exercise 10.3 — Worker pool

Write `ex10_pool.pc`. Implement a simple worker pool with 3 workers. Use a
single `Queue` for jobs and a `Channel` to collect results.

- Jobs are the integers 1–12
- Each worker computes `job * job` and sends the result to a results channel
- The main program collects all 12 results and prints their sum

```
"Expected output:
650"
```

### Exercise 10.4 — Supervised retry

Write `ex10_retry.pc`. Create a shared `Int` variable `attempt`. Use
`TaskSupervisor supervise:name:maxRestarts:` with `maxRestarts: 4`. Inside the
block, increment `attempt`, print `'attempt N'`, and signal an error
*unless* `attempt = 3` (on the third attempt, print `'success'` and exit
cleanly without an error).

```
"Expected output:
attempt 1
attempt 2
attempt 3
success"
```

### Exercise 10.5 — Rate limiter

Write `ex10_ratelimit.pc`. A producer generates 20 messages as fast as it can.
A consumer reads them but enforces a maximum rate of one message per 50ms by
sleeping `Duration ms: 50` after each. Use a `Channel` of capacity 1 to apply
backpressure — the producer will naturally slow to the consumer's rate.

Print each message as it is consumed, along with a counter:

```
"Expected output (one line per 50ms):
consumed 1
consumed 2
...
consumed 20"
```
