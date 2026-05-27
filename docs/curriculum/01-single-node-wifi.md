# Module 01 — Single Node WiFi

**Goal:** Get one ESP32-S3 on the network, serving sensor data over TCP.

**New concepts:** `Wifi` singleton, TCP listener, per-session Tasks, LED heartbeat.

**Example files:** `01_wifi_connect.pc`, `02_sensor_server.pc`

---

## 1.1 Connecting to WiFi

The `Wifi` singleton manages the station-mode radio. Connection is asynchronous —
you call `connectSSID:password:` and then poll `Wifi status` until it reaches
`#connected`.

```picoceci
Wifi connectSSID: 'ship-net' password: 'hunter2'.

[ Wifi status = #connected ] whileFalse: [
    Console println: 'waiting...'.
    Duration ms: 500.
].

Console println: 'connected: ', Wifi ipAddress.
```

`Wifi status` returns one of four symbols:

| Symbol | Meaning |
|--------|---------|
| `#idle` | Radio not started |
| `#connecting` | Association in progress |
| `#connected` | IP address assigned |
| `#error` | Association failed |

`Wifi ipAddress` returns the IPv4 string (e.g. `'192.168.1.101'`) or `nil` if
not yet connected.

### Waiting with a timeout

Polling forever is fragile. Add a counter and bail out:

```picoceci
let attempts: Int.
attempts := 0.

[ (Wifi status = #connected) not & (attempts < 20) ] whileTrue: [
    Duration ms: 500.
    attempts := attempts + 1.
].

(Wifi status = #connected)
    ifTrue: [ Console println: 'online: ', Wifi ipAddress ]
    ifFalse: [
        Console println: 'WiFi failed after 10 seconds'.
        LED blinkEvery: 200.   "fast blink = fault"
    ].
```

---

## 1.2 TCP Listener

`Wifi listenOn:do:` opens a port and calls the block for every incoming connection.
The block argument is a **session object** representing the open TCP socket.

```picoceci
Wifi listenOn: 7001 do: [ :session |
    session send: 'hello\n' asByteArray.
    session close.
].
```

`listenOn:do:` runs the block **synchronously** for each connection — only one
client can connect at a time with this pattern. That is fine for a single-client
sensor but scales poorly. The fix is to spawn a Task per session:

```picoceci
Wifi listenOn: 7001 do: [ :session |
    Task spawn: [ handleSession: session ] name: 'client-session'.
].
```

Now each client gets its own task and connections are handled concurrently.

### Session messages

| Message | Description |
|---------|-------------|
| `session send: aByteArray` | Write bytes to the socket |
| `session receive` | Block until bytes arrive; returns `ByteArray` |
| `session close` | Close the connection |

Working with strings:

```picoceci
"Send a string line"
session send: ('READING:23.5', String nl) asByteArray.

"Receive a line and parse it"
let line: String.
line := session receive asString.
```

`String nl` is the newline character `'\n'`. This avoids embedding escape
sequences in string literals.

---

## 1.3 A Minimal Sensor Server

This pattern forms the foundation of every networking example:

1. Connect to WiFi, wait for IP.
2. Start a Task that blinks the LED as a heartbeat.
3. Start a Task that reads sensor data periodically.
4. Start the TCP listener; spawn a handler Task per client.

```picoceci
"Connect"
Wifi connectSSID: 'ship-net' password: 'hunter2'.
[ Wifi status = #connected ] whileFalse: [ Duration ms: 200 ].
Console println: 'IP: ', Wifi ipAddress.

"Heartbeat"
Task spawn: [ LED blinkEvery: 1000 ] name: 'heartbeat'.

"Sensor read loop — stores latest reading in a shared variable"
let temperature: Float.
temperature := 0.0.

Task spawn: [
    [ true ] whileTrue: [
        temperature := readTemperature.   "your hardware read here"
        Duration ms: 1000.
    ]
] name: 'sensor-loop'.

"TCP server — one task per client"
Wifi listenOn: 7001 do: [ :session |
    Task spawn: [
        [ true ] whileTrue: [
            session send: ('READING:', temperature printString, String nl) asByteArray.
            Duration ms: 1000.
        ]
    ] name: 'tcp-client'.
].
```

See `examples/networking/02_sensor_server.pc` for the full version with error
handling and a simulated temperature sensor.

---

## 1.4 The REPL Over TCP

picoceci ships with `PicoceciREPL`, a full interactive interpreter you can attach
to any TCP session. This is invaluable for debugging a node in the field:

```picoceci
Wifi listenOn: 2323 do: [ :session |
    Task spawn: [
        PicoceciREPL serve: session
    ] name: 'tcp-repl'.
].
```

Connect with netcat:

```sh
nc 192.168.1.101 2323
picoceci> temperature printString
=> '22.4'
picoceci> LED toggle
```

You can query live state, call methods, and even redefine objects at runtime.
The REPL runs in its own task so it never blocks sensor reads or other clients.

---

## 1.5 LED Status Conventions

Use the LED to communicate node state at a glance:

| Pattern | Meaning |
|---------|---------|
| 1 Hz blink | Normal operation |
| 4 Hz blink | Fault / waiting for WiFi |
| Solid on | Actively serving a client |
| Off | Node crashed or powered down |

---

## Exercises

1. Modify `02_sensor_server.pc` to also serve the board's free-heap size alongside
   the temperature reading.
2. Add a second listener on port 2323 for the REPL without breaking the sensor
   server on port 7001.
3. Add a watchdog: if `temperature` has not been updated in 5 seconds, blink the
   LED rapidly and print a warning. *(Hint: store the last-updated timestamp with
   `Timestamp now`.)*
4. Add a connection counter. Each time a client connects, increment it and print
   the total at the start of each session.
