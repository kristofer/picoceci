# Module 04 — Request-Response

**Goal:** A controller node sends commands to managed nodes; nodes execute commands
and reply with status or data.

**New concepts:** Full-duplex TCP, command dispatch tables, `Dictionary`, error
responses, command-acknowledgement sequencing.

**Example files:** `06_command_controller.pc`, `06_bidirectional_sensor.pc`

---

## 4.1 Why Request-Response?

Modules 01–03 push data in one direction. A real spacecraft needs two-way
communication:

- **Ground station → node:** calibrate sensor, change sample rate, safe mode
- **Node → ground station:** telemetry, status, error reports
- **Node → node:** "are you healthy?", "what is your current reading?"

Request-response is the general pattern that covers all of these. Every command
gets an acknowledgement — this is essential in space systems where lost commands
are a safety risk.

---

## 4.2 Session as a Full-Duplex Channel

A TCP session is already full-duplex: both sides can send and receive at any time.
The challenge is that `session receive` blocks — if both sides are waiting to
receive, neither side makes progress.

**Solution:** run send and receive in separate Tasks per connection.

```
[controller]                           [sensor node]
 send-task: writes CMD lines --------> receive-task: reads CMD, dispatches
 receive-task: reads REPLY lines <---- send-task: writes ACK/REPLY lines
```

Each side has a dedicated reader Task and a shared writer. Because only one Task
writes at a time, add a mutex to guard writes:

```picoceci
let writeMutex: Semaphore.
writeMutex := Semaphore mutex.

"safe send from any task:"
writeMutex critical: [
    session <- ('ACK:', seq printString, ':ok').
].
```

---

## 4.3 Command Protocol

Extend the wire format to add sequence numbers:

```
CMD:<seq>:<command>[:<arg>]\n       controller → node
ACK:<seq>:ok\n                      node → controller (success)
ACK:<seq>:err:<reason>\n            node → controller (failure)
REPLY:<seq>:<data>\n                node → controller (data reply)
```

The sequence number `<seq>` is a monotonically increasing integer. The controller
increments it with each command. The node echoes it in every response. This
lets the controller match replies to commands and detect lost messages.

Examples:

```
CMD:1:get-reading\n
REPLY:1:23.5\n

CMD:2:set-rate:500\n
ACK:2:ok\n

CMD:3:calibrate\n
ACK:3:ok\n

CMD:4:unknown-cmd\n
ACK:4:err:unrecognized command\n
```

---

## 4.4 Command Dispatch on the Node

Use a `Dictionary` to map command names to handler blocks. This avoids a long
chain of `ifTrue:ifFalse:` comparisons:

```picoceci
let handlers: Dictionary.
handlers := Dictionary new.

handlers at: #get-reading put: [ :arg :seq |
    session <- ('REPLY:', seq, ':', temperature printString).
].

handlers at: #set-rate put: [ :arg :seq |
    sampleRateMs := arg asInteger.
    session <- ('ACK:', seq, ':ok').
].

handlers at: #calibrate put: [ :arg :seq |
    runCalibration.
    session <- ('ACK:', seq, ':ok').
].

handlers at: #safe-mode put: [ :arg :seq |
    LED blinkEvery: 100.
    session <- ('ACK:', seq, ':ok').
].
```

Dispatch a received command:

```picoceci
"Received line: CMD:3:set-rate:500"
"Parse: parts[0]=CMD parts[1]=3 parts[2]=set-rate parts[3]=500"
let handler: Any.
handler := handlers at: cmdName ifAbsent: [
    session <- ('ACK:', seq, ':err:unrecognized command').
    nil.
].
handler notNil ifTrue: [ handler value: arg value: seq ].
```

---

## 4.5 Controller Architecture

The controller connects to each managed node and maintains a command queue:

```picoceci
object NodeConnection {
    let nodeId: String.
    let channel: Any.
    let seq: Int.
    let pending: Dictionary.    "seq -> callback block"
    let mutex: Semaphore.

    init [
        seq := 0.
        pending := Dictionary new.
        mutex := Semaphore mutex.
    ]

    sendCommand: cmd arg: arg onReply: aBlock [
        seq := seq + 1.
        let thisSeq: Int.
        thisSeq := seq.
        mutex critical: [ pending at: thisSeq printString put: aBlock ].
        let line: String.
        line := (arg notNil)
            ifTrue: [ 'CMD:', thisSeq printString, ':', cmd, ':', arg ]
            ifFalse: [ 'CMD:', thisSeq printString, ':', cmd ].
        channel <- line.
    ]

    handleReply: line [
        "Parse REPLY:seq:data or ACK:seq:ok or ACK:seq:err:reason"
        "Look up pending callback by seq, call it with result"
    ]
}
```

---

## 4.6 Timeout and Retry

Commands can get lost. Implement a timeout: if no reply arrives within N seconds,
consider the command failed:

```picoceci
"Simplified timeout using Timer"
let responded: Bool.
responded := false.

Timer after: 5000 do: [
    responded ifFalse: [
        Console println: 'timeout waiting for reply to seq ', seq printString.
        "retry or escalate"
    ].
].

"When reply arrives, set responded := true before calling the callback"
```

For spacecraft systems the right policy is:
- Retry once after timeout
- If second attempt also times out, mark node as unresponsive
- Alert the operator (or the watchdog supervisor)

---

## 4.7 TaskSupervisor for Command Sessions

Wrap the command session in a `TaskSupervisor` so crashes are automatically
restarted and the controller reconnects:

```picoceci
TaskSupervisor supervise: [
    let ch: Any.
    ch := NetworkChannel connectTo: nodeIP port: 7001.
    Console println: 'connected to ', nodeId.
    runCommandSession: ch.
] name: 'cmd-session-', nodeId.
```

If `runCommandSession:` exits with an error, `TaskSupervisor` waits briefly and
tries again. The controller never needs manual intervention to re-establish a
dropped session.

---

## Exercises

1. Add a `CMD:reboot\n` command. The node should ACK, wait 500 ms, then restart.
   *(Hint: use `Task spawn:name:` to delay the reboot so the ACK can be sent first.)*
2. Add an `ACK` timeout of 3 seconds on the controller side. Log a warning for
   every command that does not receive an ACK in time.
3. Add a `HEARTBEAT:<timestamp>\n` that nodes send every 5 seconds even when no
   command is pending. The controller uses this to detect silent node failures.
4. Extend the command protocol to support **broadcast commands**: the controller
   sends one command that all connected nodes execute. Implement with a `do:` loop
   over all active `NodeConnection` objects.
