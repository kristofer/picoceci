# Module 06 — Discovery Protocol

**Goal:** Understand how picoceci's `Discovery` singleton automates what Module 05
built manually. No hard-coded addresses, including the registry address.

**New concepts:** UDP multicast, peer tables with TTL, `NodeCache`, gossip-based
synchronization, conflict resolution.

**Status:** `Discovery` and `NodeCache` are design-complete and specified in
`docs/discovery-spec.md`. This module walks through the design and shows what the
code will look like when the implementation is complete.

---

## 6.1 What Module 05 Could Not Solve

The manual registry eliminated hard-coded node IPs, but it kept one: the registry
node's own IP. Someone had to tell every node where the registry lives.

Discovery eliminates this last hard-coded address by using **UDP multicast**: a
single UDP packet sent to a multicast group address (e.g. `239.255.0.1`) reaches
every device on the LAN simultaneously. No one needs to know anyone's address in
advance.

```
node boots:
  1. sends UDP multicast: "I am sensor-alpha at 192.168.1.103 port 7001"
  2. every other node receives it and adds it to their peer table

node needs to talk to 'collector':
  1. checks local peer table → found: 192.168.1.102:7001
  2. connects directly → done
  3. if not in peer table → queries a NodeCache service → tries again
```

---

## 6.2 The Discovery Singleton

Once implemented, `Discovery` is a singleton that manages peer tracking:

```picoceci
"Start Discovery on boot — spawns its own background task"
Task spawn: [ Discovery start ] name: 'discovery'.

"Register this node's name (can register multiple)"
Discovery registerName: 'sensor-alpha'.
Discovery registerName: 'telemetry-source'.

"Look up a peer by name — returns an endpoint record or nil"
let endpoint: Any.
endpoint := Discovery lookup: 'collector'.

"If found, connect to it — same <- syntax as local channels"
endpoint notNil ifTrue: [
    let ch: Any.
    ch := NetworkChannel connectToName: 'collector'.   "uses Discovery internally"
    ch <- 'READING:23.5'.
].
```

### Discovery messages (planned API)

| Message | Description |
|---------|-------------|
| `Discovery start` | Begin announcing and listening |
| `Discovery stop` | Stop all discovery activity |
| `Discovery announce` | Broadcast presence immediately (called by `start`) |
| `Discovery registerName: aName` | Register a service name for this node |
| `Discovery lookup: aName` | Return endpoint for name, or `nil` |
| `Discovery peers` | Return all known peers (array of endpoint records) |

---

## 6.3 Comparing Discovery to the Manual Registry

| Aspect | Manual Registry (M05) | Discovery (M06) |
|--------|----------------------|-----------------|
| Bootstrap address | Registry IP hard-coded everywhere | None — UDP multicast |
| Registration | TCP to registry node | Local UDP broadcast |
| Lookup | TCP query to registry | Local peer table (in memory) |
| Failure if registry crashes | Entire naming breaks | Graceful degradation |
| Multiple redundant registries | Manual second registry | Built-in via NodeCache |
| Conflict resolution | Your exercise (#3 in M05) | Built-in, Option A |

---

## 6.4 NodeCache: the Optional Stable Index

Pure multicast has one weakness: **new nodes miss announcements** that happened
before they booted. If `collector` announced itself 30 minutes ago but `sensor-new`
just powered on, sensor-new will have an empty peer table until collector announces
again (controlled by TTL).

`NodeCache` solves this: it is a persistent service that collects and rebroadcasts
known peers on demand. Any node can optionally run a NodeCache:

```picoceci
"This node also runs a NodeCache — accepts registrations, answers queries"
NodeCache startOn: 2324.

"Another NodeCache to sync with (optional — for redundancy)"
NodeCache addPeerCache: '192.168.1.110:2324'.
```

When `Discovery lookup: name` fails in the local peer table, it automatically
queries any known NodeCache services before returning `nil`.

### NodeCache messages (planned API)

| Message | Description |
|---------|-------------|
| `NodeCache startOn: port` | Start cache service on TCP port |
| `NodeCache stop` | Shut down |
| `NodeCache register: aNodeInfo` | Add/update an entry |
| `NodeCache resolve: aName` | Return endpoint for name, or `nil` |
| `NodeCache entries` | All known entries |
| `NodeCache addPeerCache: endpoint` | Sync with another cache |
| `NodeCache syncWithPeers` | Force immediate sync |
| `NodeCache conflicts` | Return any duplicate-name conflicts |

---

## 6.5 The Wire Protocol

Discovery uses two transports:

```
UDP multicast (announcements):
  group: 239.255.0.1 port 7800
  frame: nodeId | name | tcpPort | ttlMs | timestamp

TCP (NodeCache queries):
  NodeCache listens on a configurable port (e.g. 2324)
  requests: QUERY:<name>, LIST, SUMMARY, DELTA:<revision>
  responses: ENTRY:<nodeId>:<name>:<ip>:<port>:<ttl>, NOTFOUND, ...
```

The announcement frame fields:

| Field | Type | Description |
|-------|------|-------------|
| `nodeId` | UUID string | Stable across reboots (derived from WiFi MAC) |
| `name` | String | Symbolic service name (optional) |
| `tcpPort` | Int | Port where this node's service listens |
| `ttlMs` | Int | How long peers should keep this record |
| `timestamp` | Int | Boot-relative ms when announced |

---

## 6.6 Boot Flow

Once Discovery is in the runtime, every node boots like this:

```picoceci
"1. WiFi up"
Wifi connectSSID: 'ship-net' password: 'hunter2'.
[ Wifi status = #connected ] whileFalse: [ Duration ms: 200 ].

"2. Start Discovery"
Task spawn: [ Discovery start ] name: 'discovery'.

"3. Optionally start NodeCache on this node"
NodeCache startOn: 2324.

"4. Register this node's identity"
Discovery registerName: 'sensor-alpha'.

"5. Wait briefly for peer table to populate"
Duration ms: 2000.

"6. Look up needed peers and proceed — <- syntax works identically for local and remote"
let ch: Any.
ch := NetworkChannel connectToName: 'collector'.
ch <- ('READING:', temperature printString).
```

The 2-second wait is a simple way to give multicast announcements time to arrive.
In production you would implement a retry loop with back-off instead.

---

## 6.7 Conflict Resolution

Two nodes might register the same name. `Discovery` detects this and resolves it
deterministically: the winner is the tuple `(priority, firstSeenTimestamp, nodeId)`
with the lowest value, where:

- **priority**: 0 means "primary", higher means "backup"
- **firstSeenTimestamp**: earlier boot wins ties
- **nodeId**: UUID serves as final tiebreaker

This is Option A from the discovery spec (`docs/discovery-spec.md`). The
`NodeCache conflicts` message exposes any detected conflicts for operator review.

---

## 6.8 Graceful Degradation

Discovery is designed to keep working even when parts fail:

| Failure | Behaviour |
|---------|-----------|
| Multicast unavailable | Fall back to NodeCache-only mode |
| All NodeCache nodes down | Use local peer table only |
| One of N NodeCache nodes down | Use remaining caches; continue syncing |
| Peer TTL expires | Remove from table; will reappear when peer re-announces |

The goal: a network of nodes that have never been explicitly told about each other
can still communicate, and can recover automatically from partial failures.

---

## 6.9 What to Build Now

Since `Discovery` is not yet in the runtime, the best way to prepare is:

1. **Complete Module 05** and make your manual registry as robust as possible.
2. **Keep your node names consistent** — the same names you use in module 05 will
   work unchanged once Discovery is available.
3. **Abstract your lookup calls** into a helper block so you only need to change
   one place when you migrate:

```picoceci
"Today (manual registry):"
lookupNode: name [
    ^resolveViaRegistry: name
]

"Tomorrow (Discovery):"
lookupNode: name [
    ^Discovery lookup: name
]
```

4. **Read `docs/discovery-spec.md`** in full — it describes every design decision
   and lists the open questions for the implementation sprint.

---

## Exercises

1. Draw the message sequence diagram for a full Discovery boot cycle: one sensor
   node, one collector, one NodeCache. Label every UDP and TCP message.
2. Write a `DiscoveryStub` object that implements the same API as the planned
   `Discovery` singleton but uses the Module 05 manual registry internally. This
   is a **shim** — all code written against `DiscoveryStub` will work with the real
   `Discovery` with no changes.
3. What happens if two nodes boot at exactly the same time and announce the same
   name simultaneously? Trace through the conflict resolution rules and determine
   which node wins.
4. Design a test for NodeCache gossip synchronization: if NodeCache-A knows about
   node X and NodeCache-B does not, describe the sequence of messages that brings
   NodeCache-B up to date. How many round-trips does it take?
