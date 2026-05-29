# Discovery System: Architecture Analysis and Recommendations

*Status: design recommendation document — May 2026*  
*Author: analysis based on full codebase review*

---

## Executive Summary

The `discovery-spec.md` is well-reasoned but incomplete as an implementation guide. It specifies the *what* at a high level without resolving the most consequential question: which pieces belong in TinyGo and which belong in picoceci. Getting that boundary wrong will either produce a system that is too slow for a constrained MCU or a system that is too rigid for student experimentation.

**The core recommendation:** build Discovery as a two-layer system. The transport engine, peer table, and wire protocol live in TinyGo (`pkg/discovery/`). A thin set of picoceci builtins expose a clean, ergonomic API. The two layers communicate through a small, versioned Go interface — the same pattern already established by `pkg/net/`, `pkg/freertos/`, and `pkg/sdcard/`.

Before Discovery can be built at all, two pieces of plumbing that the spec takes for granted are **missing from the codebase**: TCP client-side connect and UDP. These must be added to the IDF bridge first.

---

## 1. Current State: What Is Actually Implemented

A close reading of the source reveals a significant gap between the spec and the runtime.

### 1.1 What exists in `pkg/net/`

The `Manager` / `Session` / `Listener` triple in `pkg/net/net.go` is solid and well-structured. Three implementations ship:

| Build tag | File | Reality |
|-----------|------|---------|
| `!tinygo` | `net_stub.go` | Desktop: wraps standard `net.Listen`; WiFi is a no-op |
| `tinygo && !esp32s3_idf_bridge` | `net_tinygo.go` | TinyGo WiFi stub — `joinWiFiNetwork` is an intentional no-op hook |
| `tinygo && esp32s3_idf_bridge` | `net_tinygo_esp32s3_idf.go` | Real IDF bridge via C ABI |

The IDF bridge implements exactly six socket operations:

```
picoceci_bridge_wifi_stack_init
picoceci_bridge_wifi_connect
picoceci_bridge_wifi_disconnect
picoceci_bridge_tcp_listen      ← server socket
picoceci_bridge_tcp_accept      ← blocks until connection arrives
picoceci_bridge_tcp_recv        ← blocking read
picoceci_bridge_tcp_send        ← blocking write
picoceci_bridge_tcp_close
```

### 1.2 Critical gaps in the current ABI

**TCP client connect is absent.** There is no `picoceci_bridge_tcp_connect`. The discovery spec shows:

```picoceci
outgoing := NetworkChannel connectTo: '192.168.1.10' port: 7001.
```

This cannot be implemented today. `NetworkChannel` does not exist as a picoceci object anywhere in the codebase. Every multi-node curriculum example that makes an outbound connection is forward-looking code, not working code.

**UDP does not exist at all.** The entire Discovery announcement mechanism — UDP multicast to `239.255.0.1:7800` — has no ABI surface whatsoever. No bridge function, no Go wrapper, no picoceci binding.

### 1.3 The actual picoceci session API

The curriculum examples and the discovery spec both use `session send: aByteArray` and `session receive`. The real picoceci session object (built in `builtins.go`) exposes a different set of messages:

| Actual message | Type | Description |
|----------------|------|-------------|
| `session readLine` | → String | Read one newline-delimited line (blocking) |
| `session write: aString` | → nil | Write a string to the socket |
| `session writeln: aString` | → nil | Write a string + newline |
| `session remoteAddr` | → String | Remote IP:port |
| `session close` | → nil | Close the connection |

This is a **line-oriented text API**, not a raw byte API. It is a better fit for the simple newline-delimited protocol used in the curriculum examples. The spec's `send:aByteArray` / `receive` framing is aspirational and does not match the implementation.

### 1.4 What the discovery spec specifies vs what can be built today

| Spec feature | Buildable today? | Blocker |
|---|---|---|
| `Wifi connectSSID:password:` | ✅ Yes | Implemented |
| `Wifi listenOn:do:` (server) | ✅ Yes | Implemented |
| `NetworkChannel connectTo:port:` (client) | ❌ No | Missing bridge ABI |
| UDP multicast announce | ❌ No | Missing bridge ABI |
| UDP multicast listen | ❌ No | Missing bridge ABI |
| Discovery peer table (in-memory) | ✅ Yes | Pure Go, no ABI needed |
| NodeCache (TCP-based) | ❌ Partially | Needs TCP client |
| Node UUID from MAC address | ✅ Yes | `machine.GetID()` in TinyGo |
| TTL-based expiry | ✅ Yes | Pure Go timer |
| Name conflict detection | ✅ Yes | Pure Go |

---

## 2. What Belongs in TinyGo vs. picoceci

This is the central architectural decision. The wrong split creates problems:

- **Too much in picoceci:** The interpreted VM evaluates every peer lookup, every announcement, and every TTL prune. On a 240 MHz ESP32-S3 with 8MB PSRAM this is slow but tolerable — until you have 20 nodes gossiping every 15 seconds while a student's program is also running. More seriously, UDP socket operations are timing-sensitive in ways that an interpreted loop is not.

- **Too much in TinyGo:** The system becomes a black box. Students cannot introspect the peer table, customize conflict resolution, or experiment with gossip intervals from picoceci code. The educational value evaporates.

The right boundary: **protocol mechanics in TinyGo, policy and observation in picoceci.**

### 2.1 The TinyGo layer — `pkg/discovery/`

These pieces belong in Go, not in the interpreted VM:

#### Peer table

```go
// PeerRecord is a single discovered peer.
type PeerRecord struct {
    NodeID     string // MAC-derived UUID; dedup key
    Name       string // symbolic name (may be empty)
    Addr       string // "ip:port" of TCP service
    TTLMs      int32  // how long to keep after last seen
    LastSeenMs int64  // monotonic boot-relative ms
}

// PeerTable is the in-memory registry of live peers.
// Thread-safe. Expire() is called from a periodic timer.
type PeerTable struct { ... }
func (t *PeerTable) Upsert(r *PeerRecord)
func (t *PeerTable) Lookup(name string) *PeerRecord
func (t *PeerTable) LookupAll(name string) []*PeerRecord
func (t *PeerTable) Peers() []*PeerRecord
func (t *PeerTable) Expire()
```

**Why TinyGo:** the peer table is accessed from multiple goroutines (the announcement listener task, the TTL-prune timer callback, and any picoceci task calling `Discovery lookup:`). A `sync.RWMutex`-protected Go struct is the right primitive. A picoceci `Dictionary` protected by a `Semaphore mutex` would work but adds interpreter overhead on every lookup.

#### Announcement codec

The wire frame for UDP announcements should be encoded/decoded entirely in Go:

```go
// AnnounceFrame is the v0 UDP multicast payload.
// Encoding: pipe-separated UTF-8 text, NUL-terminated.
//   nodeId|name|tcpPort|ttlMs|timestampMs
// Example:
//   a4:e5:7c:12:34:56|sensor-alpha|7001|30000|12345678
type AnnounceFrame struct {
    NodeID      string
    Name        string
    TCPPort     uint16
    TTLMs       int32
    TimestampMs int64
}

func EncodeFrame(f *AnnounceFrame) []byte
func DecodeFrame(b []byte) (*AnnounceFrame, error)
```

**Why TinyGo:** frame parsing runs on every received UDP packet. Doing this in the picoceci interpreter (substring searches, `asInteger` calls, string splitting) is both slow and fragile. A Go function parses a frame in under a microsecond. It also keeps the wire format versioning and validation entirely outside the picoceci world.

**Why text over binary for v0:** debuggable with `tcpdump`. A student who runs `tcpdump -i any udp port 7800` on their laptop can immediately read the announcements. Binary TLV can wait for v1 when performance profiling justifies it.

#### UDP transport (new ABI additions)

The IDF bridge needs three new symbols. These are the minimum viable additions:

```c
// Join the multicast group and open a UDP receive socket.
// Returns fd >= 0 or < 0 on error.
// group: e.g. "239.255.0.1\0"
// port:  e.g. 7800
int32_t picoceci_bridge_udp_multicast_join(const char *group, uint16_t port);

// Send a UDP datagram to a multicast group.
// dst_ip: dotted-decimal string, e.g. "239.255.0.1\0"
// dst_port: destination port
// Returns bytes sent or < 0 on error.
int32_t picoceci_bridge_udp_send(
    int32_t fd,
    const uint8_t *buf, uint32_t len,
    const char *dst_ip, uint16_t dst_port
);

// Receive a UDP datagram with optional timeout.
// out_src_ip: 4-byte host-order IPv4 of sender (may be NULL)
// out_src_port: port of sender (may be NULL)
// timeout_ms: 0 = non-blocking check, portMAX_DELAY = block forever
// Returns bytes received, 0 on timeout, < 0 on error.
int32_t picoceci_bridge_udp_recv(
    int32_t fd,
    uint8_t *buf, uint32_t len,
    uint32_t *out_src_ip, uint16_t *out_src_port,
    uint32_t timeout_ms
);
```

These map to LwIP `lwip_socket(AF_INET, SOCK_DGRAM, 0)` + `IP_ADD_MEMBERSHIP` + `lwip_sendto` / `lwip_recvfrom`. They follow the same ownership and timeout conventions as the existing TCP ABI.

#### TCP client connect (new ABI addition)

```c
// Connect to a remote TCP server.
// host: dotted-decimal IP or hostname\0
// port: destination port
// timeout_ms: connect timeout
// Returns client fd >= 0 or < 0 on error.
int32_t picoceci_bridge_tcp_connect(
    const char *host, uint16_t port, uint32_t timeout_ms
);
```

This is the single most important missing bridge function. Without it, no multi-node curriculum example can actually run on hardware. Add it before anything else.

#### Node UUID from MAC

```go
// pkg/discovery/nodeid.go
// NodeID returns a stable identifier for this device derived from the WiFi MAC.
// On TinyGo, machine.GetID() returns the hardware MAC bytes.
// On desktop, falls back to a random UUID persisted in a temp file.
func NodeID() string
```

**Why TinyGo:** the MAC address is factory-burned into the ESP32-S3 chip. It is stable across reboots, across firmware updates, and unique by construction. No SD card, no UUID generation, no persistence complexity. Using `machine.GetID()` is a one-liner in TinyGo.

#### The DiscoveryEngine

```go
// pkg/discovery/engine.go

type DiscoveryEngine struct {
    nodeID      string
    names       []string
    caps        []string
    servicePort uint16

    peers       *PeerTable
    udpFD       int32
    stopCh      chan struct{}
}

// Start begins announcing and listening. Spawns two FreeRTOS tasks:
//   "discovery-announce" — sends AnnounceFrame every announceIntervalMs
//   "discovery-listen"  — receives AnnounceFrames and upserts into PeerTable
// Also starts a FreeRTOS timer for TTL pruning.
func (e *DiscoveryEngine) Start() error

// Stop cancels both tasks and the prune timer, closes the UDP socket.
func (e *DiscoveryEngine) Stop()

// RegisterName adds a symbolic name to outgoing announcements.
func (e *DiscoveryEngine) RegisterName(name string)

// Lookup returns the highest-priority live peer with the given name,
// or nil if none is known.
func (e *DiscoveryEngine) Lookup(name string) *PeerRecord

// LookupAll returns all live peers with the given name, ranked by
// (priority, firstSeenMs, nodeId).
func (e *DiscoveryEngine) LookupAll(name string) []*PeerRecord

// Peers returns a snapshot of all currently known live peers.
func (e *DiscoveryEngine) Peers() []*PeerRecord
```

The two goroutines inside `Start()` use the existing FreeRTOS task bridge:

```go
func (e *DiscoveryEngine) announceLoop() {
    ticker := freertos.NewTimer(announceIntervalMs)
    for {
        select {
        case <-ticker.C:
            frame := e.buildFrame()
            buf := EncodeFrame(frame)
            bridgeUDPSend(e.udpFD, buf, multicastGroup, multicastPort)
        case <-e.stopCh:
            return
        }
    }
}

func (e *DiscoveryEngine) listenLoop() {
    buf := make([]byte, maxFrameSize)
    for {
        n := bridgeUDPRecv(e.udpFD, buf, nil, nil, 1000)
        if n > 0 {
            if f, err := DecodeFrame(buf[:n]); err == nil {
                if f.NodeID != e.nodeID { // ignore own announcements
                    e.peers.Upsert(recordFromFrame(f))
                }
            }
        }
        select {
        case <-e.stopCh:
            return
        default:
        }
    }
}
```

**Why FreeRTOS tasks for this, not picoceci Tasks?** The announcement loop and listen loop must run whether or not any picoceci program has started, and they must not be killed if a picoceci task crashes. Using FreeRTOS tasks directly (via the bridge) at priority `tskIDLE_PRIORITY + 1` keeps them alive across picoceci restarts. The picoceci VM talks to the engine only through `e.Lookup()` and `e.RegisterName()` — both of which are mutex-protected reads/writes on the peer table.

#### NodeCache engine (defer to v1)

The NodeCache sync model (cacheSummary/cacheDelta/tombstones) is correct in design but complex in implementation. For v0, the `DiscoveryEngine` with a 30-second TTL and 10-second announcements covers all use cases in the curriculum. NodeCache becomes important only when:

1. The network has > ~20 nodes (multicast traffic becomes significant)
2. Nodes boot minutes after their peers (miss the TTL window)
3. Multi-subnet routing is required

None of these apply in a classroom lab with 5 boards on a shared AP. Implement NodeCache in v1, after the base Discovery is working and students are writing code against it.

---

### 2.2 The picoceci builtin layer

The picoceci surface is thin and stable. It wraps the `DiscoveryEngine` exactly as `makeWifiObject` wraps `Manager`.

#### `Discovery` singleton

```go
// In builtins.go, wired in InitialGlobalsWithSinks():
// sinks.DiscoveryEngine *discovery.DiscoveryEngine (new field)
globals["Discovery"] = makeDiscoveryObject(sinks.DiscoveryEngine)
```

Picoceci API:

```picoceci
Discovery start.                        "starts announce + listen loops"
Discovery stop.                         "stops both"
Discovery registerName: 'sensor-alpha'. "add name to outgoing frame"
Discovery lookup: 'collector'.          "→ 'ip:port' String or nil"
Discovery peers.                        "→ Array of Strings 'name:ip:port'"
```

Return types are kept simple: `lookup:` returns a plain `String` of `'ip:port'`, not a structured object. This keeps the API predictable for students and avoids the complexity of a `PeerRecord` picoceci object. Students can destructure the string themselves (or use `NetworkChannel connectToName:` which does it for them).

#### `NetworkChannel` object

```go
globals["NetworkChannel"] = makeNetworkChannelObject(sinks.WifiManager, sinks.DiscoveryEngine)
```

Picoceci API:

```picoceci
"Direct connect — IP:port hardcoded"
let ch: Any.
ch := NetworkChannel connectTo: '192.168.1.10' port: 7001.

"Discovery-backed connect by name"
ch := NetworkChannel connectToName: 'collector'.

"Both return an object with the same Session interface"
ch readLine.           "→ String (blocking)"
ch write: 'hello\n'.  "→ nil"
ch writeln: 'hello'.  "→ nil (appends newline)"
ch remoteAddr.         "→ 'ip:port' String"
ch close.              "→ nil"
```

`connectToName:` internally calls `engine.Lookup()`, parses the `ip:port` string, calls `picoceci_bridge_tcp_connect`, and returns a session object. On lookup failure it signals `NetworkError signal: 'peer not found: collector'`.

The session object returned by all three variants is identical — a `makeSessionObject(sess)` just like the one produced by `Wifi listenOn:do:`. **NetworkChannel and Wifi server sessions are the same object type.** A student who knows how to read from a Wifi session already knows how to read from a NetworkChannel.

---

## 3. Suitability Assessment of the Spec

### 3.1 What is well-designed

**The goals are exactly right.** "Boot, start discovery, become reachable by name" is the correct zero-configuration target. Students should not have to know or care about IP addresses.

**The layering is sound.** Separating `Discovery` (naming, peer table) from `NetworkChannel` (transport) is the right split. The discovery spec anticipates this correctly.

**TTL-based expiry is appropriate.** For a LAN with frequent announcements, TTL-based soft-state is simpler and more robust than explicit deregistration. If a node crashes, its entry expires naturally. No consensus required.

**Option A conflict resolution is correct for v0.** Deterministic winner by `(priority, firstSeenMs, nodeId)` is predictable and easy to reason about. Lease tokens (Option B) are the right long-term answer but add a clock-synchronization dependency that doesn't belong in v0.

### 3.2 What needs revision

**The spec conflates the API with the implementation.** "Candidate messages" is too vague. The implementation needs a frozen API before code can be written. The recommendations in section 2.2 above constitute that frozen API.

**UDP multicast is specified but the ABI has no UDP.** This is not a design flaw in the spec — it is a prerequisite that the implementation plan does not surface. The three new ABI functions in section 2.1 resolve this.

**The frame format is unspecified.** The spec lists fields but not encoding. Text pipe-delimited is recommended for v0 (see section 2.1). The spec should be updated with the exact format string.

**NodeCache is over-specified for v0.** The spec devotes more words to multi-cache sync than to the basic announce-and-listen flow. For a classroom system with 3–5 nodes, the base discovery is sufficient and NodeCache is a distraction. Defer it.

**"NetworkChannel connectTo:port:" is called "current style"** in the spec but does not exist in the runtime. The spec needs to distinguish between what is implemented and what is planned. The recommendation is:
- Mark `connectTo:port:` as "Phase A: implement now"
- Mark `connectToName:` as "Phase B: after Discovery"

---

## 4. The Modern Abstract Networking Services Vision

Setting aside the immediate v0 implementation, here is what a modern, spacecraft-grade networking layer should look like in picoceci — the vision that the curriculum's Module 07 (mesh) is building toward.

### 4.1 The Channel Abstraction Crosses Node Boundaries

**Local and remote channels look identical to a picoceci program.** The topology difference lives in the Go binding.

In-process channels already use the `<-` syntax:

```picoceci
readings <- 23.5.        "send to local channel"
let v: Float.
v := <-readings.         "receive from local channel"
```

Network sessions now implement the same protocol. The `<-` method on a session object calls `writeln:` under the hood (appends `\n` and writes to the TCP stream); the `receive` unary message — which is what `<-session` desugars to — calls `readLine`:

```picoceci
"Whether readings is a local Channel or a remote session, this code is the same:"
readings <- 23.5.
let v: Float.
v := <-readings.
```

This is not a future goal — it is the **current default**. `makeSessionObject()` in `builtins.go` registers both methods:

- `"<-"` → `io.WriteString(s, displayString(arg)+"\n")` (identical to `writeln:`)
- `"receive"` → `s.ReadLine()` (identical to `readLine`)

The parser desugars `ch <- val` as binary message `"<-"` on `ch`, and `<-ch` as unary `"receive"` sent to `ch`. No special handling is needed for network sessions versus local channels.

This mirrors Go's own philosophy: `io.Reader` and `io.Writer` are the same whether wrapping a file, a buffer, or a network socket.

### 4.2 The Health Broadcast


Every running node should automatically include basic health metrics in its announcement:

```
nodeId|name|tcpPort|ttlMs|timestampMs|heapFreeKB|uptimeMs|taskCount
```

This costs 12–20 extra bytes per UDP frame. The benefit: the peer table becomes a live health dashboard. From the REPL on any node:

```picoceci
Discovery peers do: [ :p |
    Console println: p.
].
"sensor-alpha:192.168.1.103:7001 heap=4200KB up=34521ms tasks=6"
"collector:192.168.1.101:7001 heap=3800KB up=34892ms tasks=9"
```

No separate health-monitoring protocol needed. Discovery does double duty. This is consistent with the "many tiny watchmen" vision from the whitepaper.

### 4.3 Supervisor-Aware Reconnection

`TaskSupervisor` already restarts tasks that crash. Networking tasks should be supervised automatically when using `NetworkChannel`:

```picoceci
"Today: student must manually wrap in TaskSupervisor"
TaskSupervisor supervise: [
    let ch: Any.
    ch := NetworkChannel connectToName: 'collector'.
    [ true ] whileTrue: [ ch writeln: temperature printString. Duration ms: 2000. ].
] name: 'to-collector'.

"Vision: NetworkChannel has a supervised: variant"
let ch: Any.
ch := NetworkChannel connectToName: 'collector' supervised: true.
"Automatically reconnects and re-resolves name on any error"
[ true ] whileTrue: [ ch writeln: temperature printString. Duration ms: 2000. ].
```

The supervised channel is a wrapper that catches errors from the underlying session, resolves the name again (in case the peer moved), reconnects, and resumes. Students get fault tolerance without having to implement it. Operators who want control use the non-supervised variant.

### 4.4 The Service Contract as an Interface

picoceci interfaces are currently used for local type checking. They could also serve as remote service contracts — documentation of what messages a remote node is expected to respond to:

```picoceci
interface TelemetrySource {
    reading        "returns a Float"
    nodeId         "returns a String"
    readingType    "returns a Symbol"
}
```

A named convention — "a node registered as `telemetry-source` responds to `reading`, `nodeId`, and `readingType`" — can be enforced by documentation and test programs. The interface type gives students a place to record and communicate that contract without additional runtime machinery.

---

## 5. Implementation Priority Order

Given the analysis, here is the recommended sequence:

### Phase A — Unblock the curriculum (3–4 days)

1. **Add `picoceci_bridge_tcp_connect` to the IDF bridge C file.** This is the single highest-value addition; it enables every multi-node curriculum example on real hardware.

2. **Add `NetworkChannel` singleton to `builtins.go`.** Implement `connectTo:port:` using the new bridge function. Session object is identical to what `Wifi listenOn:do:` already produces. This is ~80 lines of Go.

3. **Fix the session API documentation.** The curriculum examples use `send:`/`receive` which does not match the real `write:`/`readLine` API. Update `docs/stdlib.md` and the curriculum to match.

### Phase B — Discovery v0 (1 week)

4. **Add three UDP ABI functions** to the IDF bridge C file (`udp_multicast_join`, `udp_send`, `udp_recv`).

5. **Create `pkg/discovery/`** with `PeerTable`, `AnnounceFrame` codec, `DiscoveryEngine`.

6. **Wire `Discovery` singleton** into `InitialGlobalsWithSinks`. Add `DiscoveryEngine *discovery.DiscoveryEngine` to `GlobalSinks`.

7. **Add `connectToName:` to `NetworkChannel`** using `engine.Lookup()`.

8. **Desktop stubs for UDP** — test without hardware using loopback multicast (macOS/Linux support this natively).

### Phase C — Polish (2–3 days)

9. **Add health fields to announce frame** — heap, uptime, task count.

10. **Add supervised channel** (`connectToName:supervised:`) — wrapper around reconnect loop.

### Phase D — NodeCache (deferred, v1)

12. NodeCache TCP-based sync engine, after Phase C is stable and battle-tested in the classroom.

---

## 6. The ABI Extension — Full Proposed v1.1 Table

Appending to the existing ABI v1 table in `docs/freertos-bridge.md`:

| Symbol | Purpose | Inputs | Outputs | Blocking | Notes |
|--------|---------|--------|---------|----------|-------|
| `picoceci_bridge_tcp_connect` | Connect TCP client | `host`, `port`, `timeout_ms` | `fd >= 0` or `< 0` | Blocks until connect or timeout | Phase A |
| `picoceci_bridge_udp_multicast_join` | Open UDP multicast RX socket | `group` (C string), `port` | `fd >= 0` or `< 0` | Non-blocking | Phase B; LwIP `IP_ADD_MEMBERSHIP` |
| `picoceci_bridge_udp_send` | Send UDP datagram | `fd`, `buf`, `len`, `dst_ip`, `dst_port` | bytes sent or `< 0` | Non-blocking (send queue) | Phase B |
| `picoceci_bridge_udp_recv` | Receive UDP datagram | `fd`, `buf`, `len`, `out_src_ip`, `out_src_port`, `timeout_ms` | bytes received, `0` (timeout), `< 0` error | Blocks until data or timeout | Phase B |
| `picoceci_bridge_wifi_mac` | Get WiFi MAC as 6-byte array | `out_mac[6]` | `0` or `< 0` | Non-blocking | Phase B; for node UUID |

---

## 7. Annotated Spec Gaps and Corrections

These are issues to fix in `discovery-spec.md`:

1. **Add wire format specification.** The announce frame should be documented with the exact encoding:
   ```
   nodeId|name|tcpPort|ttlMs|timestampMs|heapKB|uptimeMs
   "a4:e5:7c:12:34:56|sensor-alpha|7001|30000|12345678|4200|98765"
   ```
   Max frame size: 256 bytes. Any frame exceeding 256 bytes is dropped.

2. **The spec says `Discovery channelFor:` and `Discovery resolveForNetworkChannel:`.** These should be removed. `NetworkChannel` resolves names internally by calling `engine.Lookup()`. The Discovery singleton should not vend channels — that mixes two responsibilities.

3. **The spec lists `NodeCache register: aNodeInfo`.** `aNodeInfo` is unspecified. If NodeCache is eventually implemented, this should be typed as `register: aNodeId name: aName addr: anAddr ttl: aTTL`.

4. **The spec's non-goal section says "cross-subnet routing."** Add "mDNS/Bonjour" to non-goals (to avoid confusion with Apple's implementation) and add "strong node authentication" explicitly.

5. **Session API in examples uses `channel send:` and `receive`.** Update all examples in the spec to use the actual API: `ch write:` / `ch readLine`.

---

## 8. Comparison: Discovery vs. Known Alternatives

For context on how the design compares to deployed systems:

| System | Registration | Lookup | TTL | Multi-master |
|--------|-------------|--------|-----|-------------|
| picoceci Discovery v0 | UDP multicast | Local peer table | Yes, in frame | No (single table) |
| mDNS/Bonjour | UDP multicast | Local peer table | Yes, in record | Yes (distributed) |
| Jini (Java) | TCP to Reggie | TCP to Reggie | Yes, lease | Optional |
| Kubernetes Service | etcd | kube-proxy | No (always-up) | Yes (etcd) |
| Consul | TCP/UDP gossip | Local agent | Yes, check interval | Yes (Raft) |
| ZeroMQ zbeacon | UDP broadcast | Application layer | Application | N/A |

Discovery v0 is closest to **mDNS**, which is intentional. mDNS is proven, runs on billions of devices, and the core algorithm (multicast announce + local peer table + TTL prune) is understood well enough to fit in a one-week implementation sprint. The key difference is that picoceci Discovery does not implement the full DNS-SD spec — it is a deliberate simplification that keeps the picoceci runtime below 128 KB.

---

## Summary Table

| Area | Verdict | Action |
|------|---------|--------|
| Spec goals | Correct and achievable | No change needed |
| Spec frame format | Unspecified | Add pipe-delimited text format |
| Spec NodeCache sync | Over-specified for v0 | Defer to v1 |
| Spec session API | Wrong (`send:`/`receive`) | Fixed: use `writeln:`/`readLine` or `<-`/`receive` |
| TCP client connect | Missing from ABI | Add `picoceci_bridge_tcp_connect` (Phase A) |
| UDP | Missing from ABI | Add 3 bridge functions (Phase B) |
| Peer table design | Correct in spec; must be TinyGo | Implement in `pkg/discovery/` |
| Frame codec | Must be TinyGo | Implement in `pkg/discovery/` |
| Announce/listen loops | Must be TinyGo FreeRTOS tasks | Implement in `pkg/discovery/` |
| NodeID | TinyGo, MAC-derived | `machine.GetID()` + hex encode |
| `Discovery` picoceci object | Thin builtin wrapper | Implement in `builtins.go` |
| `NetworkChannel` picoceci object | Thin builtin wrapper | Implement in `builtins.go` |
| Health in frame | Not in current spec | Add heap/uptime/taskCount fields (Phase C) |
| NodeCache | Correct design, premature | Implement after Phase C |
| `<-` syntax for network channels | **Implemented** — sessions have `"<-"` and `"receive"` | Default style; `writeln:`/`readLine` still available |
