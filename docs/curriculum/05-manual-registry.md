# Module 05 — Manual Registry

**Goal:** Nodes find each other by name, not by hard-coded IP address. A dedicated
registry node acts as the directory service.

**New concepts:** Service registry pattern, name-to-address binding, TTL and
heartbeats, fault-tolerant lookup.

**Example files:** `07_registry_node.pc`, `07_registered_node.pc`

---

## 5.1 The Hard-Coded IP Problem

Every example so far has hard-coded IP addresses:

```picoceci
ch := NetworkChannel connectTo: '192.168.1.102' port: 7001.
```

This breaks whenever:
- A node gets a new DHCP lease
- You swap a failed board
- The network topology changes
- You add a second collector for redundancy

The fix is **indirection**: nodes register under a name, and clients look up the
name to get the current address. This is exactly what DNS does on the internet,
and what the `Discovery` singleton (Module 06) will do automatically. Module 05
builds it manually so you understand what Discovery is solving.

---

## 5.2 Registry Design

The registry is a single node that:

1. **Listens** for `REGISTER` messages from nodes wanting to announce themselves.
2. **Stores** each registration as `name → { ip, port, lastSeen }`.
3. **Answers** `QUERY` messages with the current address for a name.
4. **Expires** entries whose `lastSeen` is more than `ttlMs` in the past.

```
Registration flow:
  node boots → connects to registry → sends REGISTER:sensor-alpha:7001
  registry stores { name='sensor-alpha', ip=client_ip, port=7001, lastSeen=now }

Query flow:
  client → QUERY:sensor-alpha → registry → ADDR:192.168.1.103:7001
                                             (or NOTFOUND if not registered)

Heartbeat flow:
  sensor sends HEARTBEAT:sensor-alpha every 10 seconds
  registry updates lastSeen
  if 30 seconds pass with no heartbeat → entry expires
```

---

## 5.3 Registry Wire Protocol

```
REGISTER:<name>:<port>\n         node → registry (registers self)
HEARTBEAT:<name>\n               node → registry (refreshes TTL)
QUERY:<name>\n                   client → registry (lookup by name)
ADDR:<ip>:<port>\n               registry → client (found)
NOTFOUND\n                       registry → client (name unknown/expired)
ACK:ok\n                         registry → node (REGISTER succeeded)
ACK:err:<reason>\n               registry → node (REGISTER failed)
BYE:<name>\n                     node → registry (deregister gracefully)
```

---

## 5.4 Registry Node Implementation

```picoceci
object RegistryEntry {
    let name: String.
    let ip: String.
    let port: Int.
    let lastSeen: Int.       "Timestamp ms"

    isExpired [
        ^(Timestamp now ms - lastSeen) > 30000
    ]

    summary [
        ^name , ' -> ', ip , ':', port printString
    ]
}

object Registry {
    let entries: Dictionary.      "name -> RegistryEntry"
    let mutex: Semaphore.

    init [
        entries := Dictionary new.
        mutex := Semaphore mutex.
    ]

    register: name ip: ip port: port [
        let e: RegistryEntry.
        e := RegistryEntry new.
        e name := name.
        e ip := ip.
        e port := port.
        e lastSeen := Timestamp now ms.
        mutex critical: [ entries at: name put: e ].
        ^#ok
    ]

    heartbeat: name [
        mutex critical: [
            let e: Any.
            e := entries at: name ifAbsent: [ nil ].
            e notNil ifTrue: [ e lastSeen := Timestamp now ms ].
        ].
    ]

    lookup: name [
        let e: Any.
        mutex critical: [ e := entries at: name ifAbsent: [ nil ] ].
        (e isNil | e isExpired) ifTrue: [ ^nil ].
        ^e
    ]

    prune [
        mutex critical: [
            let toRemove: OrderedCollection.
            toRemove := OrderedCollection new.
            entries do: [ :e |
                e isExpired ifTrue: [ toRemove add: e name ]
            ].
            toRemove do: [ :n | entries removeKey: n ifAbsent: [] ].
            (toRemove size > 0) ifTrue: [
                Console println: 'pruned ', toRemove size printString, ' expired entries'
            ].
        ].
    ]
}
```

Run the pruner on a timer:

```picoceci
let registry: Registry.
registry := Registry new.

Timer every: 10000 do: [ registry prune ].
```

---

## 5.5 Handling a Client Session

Each client connection runs in its own Task and speaks the registry protocol:

```picoceci
Wifi listenOn: 7000 do: [ :session |
    Task spawn: [
        let keepGoing: Bool.
        keepGoing := true.
        let clientIp: String.
        clientIp := session remoteAddress.   "IP of connecting node"
        [ keepGoing ] whileTrue: [
            let raw: String.
            raw := session receive asString trimSeparators.
            (raw size = 0) ifTrue: [ keepGoing := false ] ifFalse: [
                (raw startsWith: 'REGISTER:') ifTrue: [
                    "parse name and port from REGISTER:name:port"
                    "register and send ACK:ok"
                ].
                (raw startsWith: 'HEARTBEAT:') ifTrue: [
                    "refresh TTL"
                ].
                (raw startsWith: 'QUERY:') ifTrue: [
                    "lookup and send ADDR or NOTFOUND"
                ].
                (raw = 'BYE') ifTrue: [ keepGoing := false ].
            ].
        ].
        session close.
    ] name: 'registry-session'.
].
```

---

## 5.6 Client-Side: Registering and Looking Up

Every regular node (sensor, collector, controller) has two behaviours:

**On boot** — connect to the registry, register, then start the heartbeat:

```picoceci
let registryIP: String.
registryIP := '192.168.1.100'.     "registry node IP — the one hardcoded address"
let myName: String.
myName := 'sensor-alpha'.
let myPort: Int.
myPort := 7001.

"Register"
let regCh: Any.
regCh := NetworkChannel connectTo: registryIP port: 7000.
regCh send: ('REGISTER:', myName, ':', myPort printString, String nl) asByteArray.
let ack: String.
ack := regCh receive asString trimSeparators.
Console println: 'registry: ', ack.
regCh close.

"Heartbeat task"
Task spawn: [
    [ true ] whileTrue: [
        Duration ms: 10000.
        [
            let hbCh: Any.
            hbCh := NetworkChannel connectTo: registryIP port: 7000.
            hbCh send: ('HEARTBEAT:', myName, String nl) asByteArray.
            hbCh close.
        ] on: Error do: [ :e |
            Console println: 'heartbeat failed: ', e message.
        ].
    ]
] name: 'heartbeat-reg'.
```

**When connecting to a peer** — query the registry for the address:

```picoceci
resolveNode: name [
    let ch: Any.
    ch := NetworkChannel connectTo: registryIP port: 7000.
    ch send: ('QUERY:', name, String nl) asByteArray.
    let reply: String.
    reply := ch receive asString trimSeparators.
    ch close.
    (reply startsWith: 'ADDR:') ifTrue: [
        ^reply copyFrom: 6 to: reply size   "strip 'ADDR:' prefix"
    ].
    ^nil   "NOTFOUND"
]
```

---

## 5.7 Limitations of a Central Registry

This design works well but has one critical weakness: **the registry is a single
point of failure**. If it crashes, no node can find any other node.

Mitigations at this level:

1. **Local cache:** each node caches resolved addresses for 60 seconds. If the
   registry is down, the node keeps using the last known address.
2. **Retry with back-off:** failed lookups retry 3 times with 2-second gaps.
3. **Registry supervisor:** run the registry under `TaskSupervisor` so it restarts
   after crashes.

But these are partial fixes. The full solution is a **distributed** registry — no
single node owns all the data. That is what Module 06 (Discovery) and Module 07
(Mesh) address.

---

## Exercises

1. Add `LIST\n` as a registry command. The registry replies with a newline-delimited
   list of all registered names and their addresses.
2. Implement the local address cache on the client side. Resolved addresses should
   be cached for 60 seconds before a fresh lookup is needed.
3. Add priority support: `REGISTER:name:port:priority\n`. If two nodes register the
   same name, the one with the lower priority number wins. *(This is Option A from
   the Discovery spec's conflict-resolution design.)*
4. Run the registry under `TaskSupervisor`. Add a test where you kill the registry
   Task mid-run and verify it restarts and re-registers its own service within 5
   seconds.
