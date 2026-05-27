# Module 07 — Mesh Network

**Goal:** Build a network where every node can reach every other node with no
central coordinator. Any node can fail and the rest continue operating.

**New concepts:** Routing tables, gossip protocol, TTL-based expiry, message
forwarding, fault detection.

**Example files:** `08_mesh_node.pc`

---

## 7.1 Why a Mesh?

Every topology so far has had a privileged node:

| Module | Topology | Single point of failure |
|--------|----------|------------------------|
| 03 | Star | The hub |
| 05 | Registry | The registry node |
| 06 | Discovery | NodeCache (until redundancy added) |

In a star, if the hub fails, no spoke can reach any other spoke. On a spacecraft,
a node failure must never partition the network. The solution is a **mesh**: every
node connects directly to every other node it can reach, and the network routes
around failures automatically.

```
Star:                Mesh:
   A                A --- B
   |             / | \ / | \
   H            C---H---D---E
   |             \ | / \ | /
   B                F --- G
(H fails → A can't reach B)   (H fails → A reaches B via C→F→B)
```

---

## 7.2 Routing Table

Every mesh node maintains a **routing table**: a dictionary from destination name
to next-hop address. To reach node X, connect to the node in the `nexthop` field.

```picoceci
object RouteEntry {
    let dest: String.       "destination node name"
    let nexthop: String.    "IP:port of next node on the path"
    let hops: Int.          "number of hops to destination"
    let lastSeen: Int.      "Timestamp ms when route was last confirmed"

    isExpired [
        ^(Timestamp now ms - lastSeen) > 30000
    ]
}

object RoutingTable {
    let routes: Dictionary.   "dest name -> RouteEntry"
    let mutex: Semaphore.
    let myName: String.

    init [
        routes := Dictionary new.
        mutex := Semaphore mutex.
    ]

    addDirect: name at: ipPort [
        let e: RouteEntry.
        e := RouteEntry new.
        e dest := name.
        e nexthop := ipPort.
        e hops := 1.
        e lastSeen := Timestamp now ms.
        mutex critical: [ routes at: name put: e ].
    ]

    addVia: name nexthop: ipPort hops: n [
        let existing: Any.
        mutex critical: [ existing := routes at: name ifAbsent: [ nil ] ].
        (existing isNil | (existing hops > n)) ifTrue: [
            let e: RouteEntry.
            e := RouteEntry new.
            e dest := name.
            e nexthop := ipPort.
            e hops := n.
            e lastSeen := Timestamp now ms.
            mutex critical: [ routes at: name put: e ].
        ].
    ]

    lookup: name [
        let e: Any.
        mutex critical: [ e := routes at: name ifAbsent: [ nil ] ].
        (e isNil | e isExpired) ifTrue: [ ^nil ].
        ^e
    ]

    prune [
        mutex critical: [
            let stale: OrderedCollection.
            stale := OrderedCollection new.
            routes do: [ :e | e isExpired ifTrue: [ stale add: e dest ] ].
            stale do: [ :n | routes removeKey: n ifAbsent: [] ].
        ].
    ]

    allRoutes [
        let result: OrderedCollection.
        result := OrderedCollection new.
        mutex critical: [ routes do: [ :e | result add: e ] ].
        ^result
    ]
}
```

---

## 7.3 Gossip Protocol

Nodes learn about each other through **gossip**: each node periodically tells its
direct neighbors about everything in its routing table. Neighbors merge the new
routes with their own, incrementing the hop count. Eventually every node knows a
path to every other node.

```
node A knows:    node B learns from A:   node C learns from B:
  B (1 hop)        C (1 hop, direct)       A (2 hops via B)
  C (2 hops via B) A (1 hop, direct)       B (1 hop, direct)
                   C (1 hop, direct)
```

The gossip message is a `ROUTE:` line containing all known routes:

```
ROUTE:A:192.168.1.101:7001:1,C:192.168.1.103:7001:1\n
       ^name         ^addr  ^hops
```

---

## 7.4 Mesh Node Wire Protocol

```
HELLO:<name>:<port>\n          initial handshake on connect
ROUTE:<csv-route-list>\n       gossip: share routing table with neighbor
MSG:<dest>:<src>:<payload>\n   routed message (forward if dest != me)
PING:<seq>\n                   liveness check
PONG:<seq>\n                   liveness reply
BYE:<name>\n                   graceful disconnect
```

The `MSG:` message enables routing: if a node receives `MSG:C:A:hello` and it is
not C, it looks up C in its routing table and forwards the message to the nexthop.
This is **store-and-forward** routing, the simplest mesh routing scheme.

---

## 7.5 Mesh Node Architecture

```
┌────────────────────────────────────────────────────────┐
│  TCP Listener (server side)                            │
│  One Task per neighbor that connects to us             │
├────────────────────────────────────────────────────────┤
│  Neighbor Connector (client side)                      │
│  Connects to known neighbors at boot                   │
│  Reconnects automatically on disconnect                │
├────────────────────────────────────────────────────────┤
│  Gossip Timer Task                                     │
│  Every 15s: sends ROUTE to all connected neighbors     │
├────────────────────────────────────────────────────────┤
│  Message Dispatcher                                    │
│  If dest = me: deliver locally                         │
│  If dest != me: forward to nexthop                     │
├────────────────────────────────────────────────────────┤
│  Prune Timer Task                                      │
│  Every 10s: remove expired routes                      │
└────────────────────────────────────────────────────────┘
```

---

## 7.6 Startup Sequence

A mesh node needs a list of **seed neighbors** — at least one neighbor's
IP:port to bootstrap routing. This is the only hard-coded address in a full mesh.
After the initial gossip exchange, the node learns the rest of the network.

```picoceci
"Seed neighbors — minimum one known node to bootstrap"
let seeds: Array.
seeds := Array new: 2.
seeds at: 1 put: '192.168.1.101:7001'.
seeds at: 2 put: '192.168.1.102:7001'.

"Connect to seeds"
seeds do: [ :ipPort |
    Task spawn: [ connectToNeighbor: ipPort ] name: 'neighbor-', ipPort.
].

"Accept incoming neighbor connections"
Wifi listenOn: myPort do: [ :session |
    Task spawn: [ handleNeighbor: session ] name: 'inbound-neighbor'.
].

"Start gossip timer"
Timer every: 15000 do: [ gossipToAll ].

"Start pruner"
Timer every: 10000 do: [ routingTable prune ].
```

---

## 7.7 Handling a Neighbor Connection

Both client-side and server-side connections use the same handler after the
initial handshake:

```picoceci
handleNeighborSession: session name: neighborName [
    routingTable addDirect: neighborName at: (neighborName, ':', myPort printString).
    
    [ true ] whileTrue: [
        let line: String.
        line := session receive asString trimSeparators.
        (line size = 0) ifTrue: [
            routingTable removeKey: neighborName ifAbsent: [].
            ^self
        ].
        (line startsWith: 'ROUTE:') ifTrue: [
            mergeRoutesFrom: neighborName line: line session: session.
        ].
        (line startsWith: 'MSG:') ifTrue: [
            dispatchMessage: line.
        ].
        (line startsWith: 'PING:') ifTrue: [
            let seq: String.
            seq := line copyFrom: 6 to: line size.
            session send: ('PONG:', seq, String nl) asByteArray.
        ].
        (line startsWith: 'BYE:') ifTrue: [ ^self ].
    ].
]
```

---

## 7.8 Message Forwarding

```picoceci
dispatchMessage: line [
    "line = MSG:dest:src:payload"
    "Parse dest, src, payload"
    (dest = myName) ifTrue: [
        Console println: 'MSG from ', src, ': ', payload.
        ^self
    ].
    "Forward to nexthop"
    let route: Any.
    route := routingTable lookup: dest.
    route isNil ifTrue: [
        Console println: 'no route to ', dest.
        ^self
    ].
    let fwdCh: Any.
    fwdCh := activeNeighbors at: route nexthop ifAbsent: [ nil ].
    fwdCh notNil ifTrue: [
        fwdCh send: (line, String nl) asByteArray.
    ].
]
```

---

## 7.9 Fault Detection

Gossip handles *route* failures (stale routes expire). To detect *node* failures
faster, add a heartbeat:

```picoceci
"Send PING every 5 seconds to each neighbor"
"If no PONG arrives within 3 seconds, mark neighbor as unresponsive"
"Remove its direct routes; gossip the removal to other neighbors"
```

When a node fails:
1. Its direct neighbors detect the failure via missed heartbeats.
2. They remove the direct route and gossip the update.
3. The rest of the network converges within 1–2 gossip cycles.
4. Nodes that had multi-hop routes through the failed node switch to alternate
   paths if they exist.

This is the core of fault-tolerant networking: failure is normal, the network
adapts.

---

## 7.10 Spacecraft Relevance

The patterns in this module map directly to real space system architectures:

| picoceci concept | Space systems equivalent |
|-----------------|--------------------------|
| Mesh node | Spacecraft subsystem controller |
| Routing table | AFDX / SpaceWire routing table |
| Gossip protocol | STANAG / MIL-STD health broadcast |
| TTL expiry | Watchdog timeout / dead-reckoning |
| Seed neighbors | Pre-programmed initial topology |
| Store-and-forward | DTN (Delay-Tolerant Networking) |
| `TaskSupervisor` | FDIR (Fault Detection, Isolation, Recovery) |

Building a mesh in picoceci on ESP32-S3 boards gives you hands-on experience with
the same reasoning required for flight software, at a fraction of the cost.

---

## 7.11 Putting It All Together: Suggested Lab Setup

Flash five ESP32-S3 boards as mesh nodes. Assign each a unique name and configure
two seed neighbors each. The topology should form a ring with cross-links:

```
A --- B
|  X  |
D --- C
  \
   E (connects to A and C)
```

1. Boot all five nodes and verify each can reach all others (query the routing
   table from the REPL).
2. Physically disconnect node B's power. Within 30 seconds, verify A can still
   reach C via D.
3. Reconnect B. Verify it rejoins the mesh automatically within two gossip cycles.
4. Send a message from A to E via `MSG:E:A:hello` and watch it hop through the
   network.

---

## Exercises

1. Add a hop-count limit of 5 to `MSG:` forwarding. If a message arrives with
   hop count ≥ 5, drop it and log a warning. This prevents routing loops.
2. Implement **split horizon**: when gossipping routes to neighbor X, do not
   include routes that were learned *from* X. This prevents routing loops and is
   a standard distance-vector optimization.
3. Add **route pinning**: the operator can send `PIN:dest:nexthop` to fix a
   specific route that will not be overwritten by gossip. This is useful for
   hard-wired physical links.
4. Build a **network map**: a special node that collects `ROUTE:` gossip from all
   other nodes and renders a full topology as a text adjacency list. Display it
   via the REPL with `networkMap display`.
5. Implement a **two-node redundant path** for critical messages: instead of
   routing `MSG:` to a single nexthop, send it on two disjoint paths
   simultaneously and have the receiver dedup by sequence number.
