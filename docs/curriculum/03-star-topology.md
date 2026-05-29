# Module 03 — Star Topology

**Goal:** One hub node accepts many spoke nodes simultaneously. Data fans in from
all spokes; the hub aggregates and logs it.

**New concepts:** Concurrent TCP sessions, shared state with `Semaphore`, fan-in
collection, per-spoke Tasks.

**Example files:** `05_hub_node.pc`, `05_spoke_node.pc`

---

## 3.1 Why a Star?

Point-to-point (Module 02) works for two nodes. When you add a third sensor you
have choices:

```
Option A — daisy chain:   sensor-1 --> sensor-2 --> collector
Option B — star:          sensor-1 \
                          sensor-2 --> hub
                          sensor-3 /
```

Option A couples every node to its neighbors — if sensor-2 fails, sensor-3 loses
its path to the collector. Option B isolates failures: any spoke can die without
affecting the others. The hub is the only shared dependency.

A star is the right first step when you have a natural "ground station" or
"data-collection point" and N autonomous sensor nodes.

---

## 3.2 Hub Architecture

The hub has three layers:

```
┌──────────────────────────────────────────────┐
│  TCP Listener                                │
│  Wifi listenOn:do: → spawn Task per session  │
├──────────────────────────────────────────────┤
│  Session Handler Task (one per spoke)        │
│  Reads READING:n lines                       │
│  Pushes to shared Queue                      │
├──────────────────────────────────────────────┤
│  Aggregator Task                             │
│  Drains Queue, computes stats, logs/stores   │
└──────────────────────────────────────────────┘
```

The `Queue` decouples session handlers from the aggregator. Each handler pushes
readings without waiting; the aggregator drains at its own pace.

---

## 3.3 Hub Skeleton

```picoceci
"Shared queue — typed, capacity 64"
let readings: Queue.
readings := Queue new: 64.

"Aggregator task — runs forever draining the queue"
Task spawn: [
    [ true ] whileTrue: [
        let line: String.
        line := readings receive.            "blocks if queue empty"
        Console println: 'received: ', line.
    ]
] name: 'aggregator'.

"TCP listener — spawns one handler task per spoke"
Wifi listenOn: 7001 do: [ :session |
    Task spawn: [
        let keepGoing: Bool.
        keepGoing := true.
        [ keepGoing ] whileTrue: [
            let raw: ByteArray.
            raw := session receive.
            (raw size = 0)
                ifTrue: [ keepGoing := false ]
                ifFalse: [ readings send: raw asString trimSeparators ]
        ].
        session close.
    ] name: 'spoke-handler'.
].
```

---

## 3.4 Tracking Connected Spokes

The hub should know which spokes are currently connected. Use a shared
`OrderedCollection` protected by a `Semaphore`:

```picoceci
let spokes: OrderedCollection.
spokes := OrderedCollection new.
let spokesMutex: Semaphore.
spokesMutex := Semaphore mutex.

"When a spoke connects, register it:"
spokesMutex critical: [ spokes add: spokeName ].

"When a spoke disconnects, remove it:"
spokesMutex critical: [ spokes remove: spokeName ifAbsent: [] ].

"Inspect from the REPL:"
picoceci> spokes size printString
=> '3'
```

> **Semaphore critical:** The `critical:` message acquires the mutex, evaluates
> the block, then releases it — even if the block signals an error. Always use
> `critical:` (not raw `wait`/`signal`) for mutual exclusion.

---

## 3.5 Spoke Architecture

A spoke is a sensor node that:

1. Reads its sensor at a fixed rate.
2. Connects to the hub (with reconnection).
3. Streams `READING:nodeId:value` lines.
4. Identifies itself with a `HELLO:nodeId` handshake on connect.

The nodeId is just the board's IP address for now. Module 05 (manual registry)
will assign human-readable names.

```picoceci
let nodeId: String.
nodeId := Wifi ipAddress.    "e.g. '192.168.1.103'"

[ true ] whileTrue: [
    [
        let ch: Any.
        ch := NetworkChannel connectTo: hubIP port: 7001.
        ch <- ('HELLO:', nodeId).
        [ true ] whileTrue: [
            let reading: Float.
            reading := readSensor.
            ch <- ('READING:', nodeId, ':', reading printString).
            Duration ms: 2000.
        ].
    ] on: Error do: [ :e |
        Console println: 'hub unreachable: ', e message.
        Duration ms: 5000.
    ].
].
```

---

## 3.6 Message Format for Multi-Source Data

With multiple spokes sending data, the hub must know which spoke sent each
reading. Extend the protocol to include the sender:

```
HELLO:192.168.1.103\n          "spoke announces its ID on connect"
READING:192.168.1.103:23.5\n  "reading + sender ID"
BYE:192.168.1.103\n            "spoke is disconnecting gracefully"
```

The hub parses the node ID out of each message before queuing it. This also lets
the aggregator track per-node statistics (last seen, reading count, min/max).

---

## 3.7 Aggregation Object

The aggregator can be encapsulated as an object:

```picoceci
object NodeStats {
    let nodeId: String.
    let count: Int.
    let total: Float.
    let min: Float.
    let max: Float.

    init [
        count := 0.
        total := 0.0.
        min := 999999.0.
        max := -999999.0.
    ]

    addReading: value [
        count := count + 1.
        total := total + value.
        (value < min) ifTrue: [ min := value ].
        (value > max) ifTrue: [ max := value ].
        ^self
    ]

    average [
        (count = 0) ifTrue: [ ^0.0 ].
        ^total / count asFloat
    ]

    summary [
        ^nodeId , ': count=', count printString,
         ' avg=', self average printString,
         ' min=', min printString,
         ' max=', max printString
    ]
}
```

---

## Exercises

1. Add a `STATS` command: when a client connects and sends `CMD:stats\n`, the hub
   replies with a summary line for each tracked spoke.
2. Add per-spoke timeout detection. If a spoke has not sent a reading in 10 seconds,
   log a warning and mark it offline.
3. Add a second hub that also listens on port 7002. Run two aggregator instances.
   Modify the spokes to report to both hubs simultaneously. *(This is your first
   redundancy exercise.)*
4. Cap the number of concurrent spoke connections at 8. If a ninth spoke tries to
   connect, send `ERR:hub-full\n` and close the session.
