# Module 02 — Point-to-Point

**Goal:** Two nodes exchange structured messages reliably over a persistent TCP
connection.

**New concepts:** `NetworkChannel`, message framing, push vs. pull, reconnection.

**Example files:** `03_collector_node.pc`, `04_ping_sender.pc`, `04_pong_receiver.pc`

---

## 2.1 Client-Side Connections

Module 01 showed the server side (`Wifi listenOn:do:`). The client side uses
`NetworkChannel`:

```picoceci
let ch: Any.
ch := NetworkChannel connectTo: '192.168.1.102' port: 7001.
ch send: 'HELLO:sensor-1\n' asByteArray.
let reply: String.
reply := ch receive asString.
ch close.
```

`NetworkChannel connectTo:port:` blocks until the connection is established or
fails. On success it returns a channel object with the same `send:`/`receive`/`close`
interface as a server-side session.

> **Note:** `NetworkChannel` is part of the net module.  
> Future modules will replace the IP address with a symbolic name via the
> `Discovery` singleton (see Module 06). For now, hard-code the collector's IP.

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

The `LineProtocol` helper (included in the module examples) handles this:

```picoceci
object LineProtocol {
    let _session: Any.

    on: aSession [
        let p: LineProtocol.
        p := LineProtocol new.
        p _session := aSession.
        ^p
    ]

    send: aString [
        _session send: (aString , String nl) asByteArray.
        ^self
    ]

    receive [
        ^_session receive asString trimSeparators
    ]

    close [
        _session close.
        ^self
    ]
}
```

Use it anywhere you have a session or channel:

```picoceci
let proto: LineProtocol.
proto := LineProtocol on: ch.
proto send: 'READING:23.5'.
let msg: String.
msg := proto receive.
```

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
    | connected |
    connected := false.
    [
        let ch: Any.
        ch := NetworkChannel connectTo: '192.168.1.102' port: 7001.
        connected := true.
        let proto: LineProtocol.
        proto := LineProtocol on: ch.
        [ true ] whileTrue: [
            proto send: 'READING:', temperature printString.
            Duration ms: 2000.
        ].
    ] on: Error do: [ :e |
        Console println: 'lost connection: ', e message.
    ].
    connected ifFalse: [ Duration ms: 5000 ].   "back-off before retry"
].
```

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
