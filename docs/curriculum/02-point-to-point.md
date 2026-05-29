# Module 02 — Point-to-Point

**Goal:** Two nodes exchange structured messages reliably over a persistent TCP
connection.

**New concepts:** `NetworkChannel`, the `<-` channel syntax for network sessions,
message framing, push vs. pull, reconnection.

**Example files:** `03_collector_node.pc`, `04_ping_sender.pc`, `04_pong_receiver.pc`

---

## 2.1 Client-Side Connections

Module 01 showed the server side (`Wifi listenOn:do:`). The client side uses
`NetworkChannel`:

```picoceci
let ch: Any.
ch := NetworkChannel connectTo: '192.168.1.102' port: 7001.
ch <- 'HELLO:sensor-1'.
let reply: String.
reply := <-ch.
ch close.
```

`NetworkChannel connectTo:port:` blocks until the connection is established or
fails. On success it returns a channel object. **Both server-side sessions and
client-side channels support the same `<-` send / `<-` receive syntax** — the
topology difference lives in the Go binding, not in your picoceci code.

> **Note:** Future modules replace the IP address with a symbolic name via the
> `Discovery` singleton (see Module 06). For now, hard-code the collector's IP.

### The <- syntax

`ch <- value` sends `value` as a newline-terminated text line.  
`<-ch` (prefix form) receives the next newline-terminated line and returns it as a `String`.

These are the same operators used for in-process `Channel` objects. A local
queue and a remote TCP connection look identical to a picoceci program:

```picoceci
"Local channel — works today"
let q: Channel.
q <- 'READING:23.5'.
let msg: String.
msg := <-q.

"Remote session — also works, same syntax"
let sess: Any.
sess <- 'READING:23.5'.
msg := <-sess.
```

The underlying methods are still available if you need them: `session writeln: str`
and `session readLine` do exactly what `<-` and `receive` do, respectively.

---

## 2.2 Push vs. Pull

There are two architectures for getting data from a sensor to a collector:

### Push (sensor drives)

The sensor connects to the collector and streams readings. The collector is passive.

```
[sensor] --TCP--> [collector]
 sends READING:23.5 every second
```

Pros: sensor controls timing; collector needs no polling logic.  
Cons: sensor must know the collector's address; collector can't request on demand.

### Pull (collector drives)

The collector connects to the sensor and requests readings when it wants them.

```
[collector] --TCP--> [sensor]
 sends CMD:get-reading
 receives READING:23.5
```

Pros: collector controls sampling rate; easy to add more collectors.  
Cons: sensor must be a server; collector must handle sensor unavailability.

Module 02 demonstrates **push** — it is simpler and fits a sensor that owns its
own timing (e.g. "read every second"). Module 04 adds the reverse direction for
request-response.

---

## 2.3 Message Framing

TCP is a byte stream — it has no concept of message boundaries. If you send
`READING:23.5\n` and then `READING:24.1\n`, the receiver might get them as:

- Two separate reads (ideal)
- One read: `READING:23.5\nREADING:24.1\n`
- Partial: `READING:23.` then `.5\nREADING:24.1\n`

**Always terminate messages with a newline and parse until `\n`.**

The `<-` operator handles this automatically:

- `session <- 'READING:23.5'` appends `\n` before sending
- `<-session` reads until the next `\n` and returns the line without the newline

You never need to think about `String nl` or `asByteArray` when using `<-`. The
protocol is: one line per message, newline-terminated.

---

## 2.4 Sensor → Collector Example

**Collector** (flash first — it must be listening when the sensor boots):

```
[collector] listens on port 7001
            receives READING:n messages
            logs each reading to Console
```

**Sensor** (flash second):

```
[sensor] connects to collector at hard-coded IP
         sends READING:n every 2 seconds
         reconnects automatically if disconnected
```

See `examples/networking/03_collector_node.pc` and the sensor portion in
`02_sensor_server.pc` for full code.

### Reconnection pattern

A sensor must tolerate the collector being temporarily unavailable:

```picoceci
[ true ] whileTrue: [
    [
        let ch: Any.
        ch := NetworkChannel connectTo: '192.168.1.102' port: 7001.
        Console println: 'connected to collector'.
        [ true ] whileTrue: [
            ch <- ('READING:', temperature printString).
            Duration ms: 2000.
        ].
    ] on: Error do: [ :e |
        Console println: 'lost connection: ', e message, ' — retrying in 5s'.
        Duration ms: 5000.
    ].
].
```

Note the parens around `'READING:', temperature printString` — without them,
the binary `,` operator would bind to the right argument of `<-` first, which
is not what you want. When in doubt, add parens.

---

## 2.5 Ping-Pong: Verifying the Link

Before building a real sensor+collector pair, verify the link with a simple
ping-pong test. One node sends `PING:n`, the other replies `PONG:n`.

```
node A: PING:1 -->
                <-- PONG:1
        PING:2 -->
                <-- PONG:2
```

With `<-` syntax:

```picoceci
"Sender side"
ch <- ('PING:', seq printString).
let reply: String.
reply := <-ch.

"Receiver side"
let msg: String.
msg := <-session.
(msg startsWith: 'PING:') ifTrue: [
    let seq: String.
    seq := msg copyFrom: 6 to: msg size.
    session <- ('PONG:', seq).
].
```

This confirms:
- Both nodes are on the network
- Ports are reachable
- The message framing works

See `04_ping_sender.pc` and `04_pong_receiver.pc`.

---

## 2.6 Naming Nodes

Even with hard-coded IPs, give every node a name. Print it at boot and include it
in every log line:

```picoceci
let nodeName: String.
nodeName := 'sensor-alpha'.
Console println: nodeName, ' online at ', Wifi ipAddress.
```

This makes it much easier to read mixed serial logs when multiple nodes are
printing simultaneously. In later modules the name becomes the service identity
that `Discovery` uses for lookup.

---

## Exercises

1. Make the collector write each reading to the SD card (one reading per line in
   `/readings.log`). Use the `SDCard` and `File` objects from `docs/sdcard.md`.
2. Add a sequence number to each `READING:` message. Have the collector detect and
   log any gaps in the sequence.
3. Implement the pull variant: swap client/server roles so the collector connects
   to the sensor and requests readings on demand with `CMD:get-reading`.
4. Simulate a network failure by calling `Wifi disconnect` in the collector after
   10 readings. Verify the sensor reconnects and resumes without losing its local
   buffer.
