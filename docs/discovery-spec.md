# Discovery Networking Spec (Draft)

Status: planning-only draft for issue discussion.

## Goals

- Allow picoceci nodes on LAN (host + ESP32-S3) to discover each other automatically.
- Keep startup minimal: node boots, starts Discovery task, becomes reachable by name.
- Support optional cache/index node to make lookups stable and fast.

## Objects

### `Discovery` (singleton)

Core responsibilities:

- broadcast presence (`announce`)
- listen for peers (`listen`)
- maintain in-memory peer table with TTL (`peers`, `pruneExpired`)
- request peer by symbolic name (`lookup:`)
- provide name-to-channel resolution for `NetworkChannel` endpoints

Candidate messages:

- `Discovery start`
- `Discovery stop`
- `Discovery announce`
- `Discovery peers`
- `Discovery lookup: aName`
- `Discovery registerName: aName`
- `Discovery channelFor: aName`
- `Discovery resolveForNetworkChannel: aName`

### `NodeCache` (optional service; one or more instances)

Core responsibilities:

- accept registrations from nodes (`register:name:addr:ttl:`)
- answer name/address lookups (`resolve:`)
- mirror recent announcements for warm restart
- exchange cache summaries/deltas with peer NodeCache instances
- maintain conflict metadata when duplicate names are detected

Candidate messages:

- `NodeCache startOn: port`
- `NodeCache stop`
- `NodeCache register: aNodeInfo`
- `NodeCache resolve: aName`
- `NodeCache entries`
- `NodeCache addPeerCache: endpoint`
- `NodeCache syncWithPeers`
- `NodeCache conflicts`

## Discovery + `NetworkChannel` Unification (proposed)

`NetworkChannel` usage in docs/examples should route through Discovery instead of
hard-coded host/port whenever symbolic names are available.

Current style in examples:

```picoceci
let outgoing: Any.
outgoing := NetworkChannel connectTo: '192.168.1.10' port: 7001.
```

Proposed unified style:

```picoceci
let outgoing: Any.
Discovery registerName: 'telemetry-collector'.
outgoing := NetworkChannel connectToName: 'telemetry-collector'.
```

Proposed layering:

- `NetworkChannel connectToName:` asks `Discovery lookup:` for endpoint records.
- `Discovery` first checks local peer table, then one-or-more `NodeCache` services.
- `NetworkChannel` remains transport-focused; Discovery owns naming and peer selection.

## Wire Protocol (v0 draft)

- Transport: UDP multicast for announcements + optional TCP for NodeCache queries.
- Announcement frame fields:
  - `nodeId` (stable UUID)
  - `name` (optional symbolic name)
  - `capabilities` (array of symbols)
  - `tcpPort` (control/repl/service port)
  - `ttlMs`
  - `timestamp`
- Dedup key: `nodeId`.
- Expiry: prune when `now > lastSeen + ttlMs`.
- NodeCache sync messages:
  - `cacheSummary` (cacheId, revision, entryCount, digest)
  - `cacheDelta` (upserts + tombstones since revision N)
  - `cacheConflict` (name, candidate records, selected winner metadata)

## Runtime Model

- Boot flow:
  1. start `Task spawn: [ Discovery start ] name: 'discovery'`
  2. optionally start `NodeCache`
  3. register local service names
- Failure mode:
  - if multicast unavailable, fallback to NodeCache-only mode
  - if NodeCache unavailable, continue in peer-to-peer mode
  - if one NodeCache is unavailable, continue with remaining caches

## Jini/Go-Inspired Notes

- Jini-style service registration/lookup semantics adapted to constrained LAN use.
- Go channel-style event loop recommendation for implementation:
  - inbound announce channel
  - timer/ticker channel for TTL pruning
  - command channel for lookup/register API
  - cache-sync channel for NodeCache peer reconciliation

## Multi-NodeCache Synchronization Model (v0)

- Each NodeCache has a unique `cacheId` and monotonically increasing `revision`.
- Caches gossip `cacheSummary`, then request/apply `cacheDelta` updates.
- Deltas include tombstones so removals converge.
- Eventual consistency is acceptable for v0; reads may be stale briefly.

Suggested anti-entropy cadence:

- periodic full-summary exchange (for drift detection)
- frequent small deltas (for low-latency updates)
- on-demand resync when digest mismatch persists

## Duplicate Node Name Conflicts (proposals)

Two or more nodes claiming the same logical name must be detected and resolved.

### Option A: deterministic winner (simple default)

- Winner is lowest `(priority, firstSeenTimestamp, nodeId)` tuple.
- Pros: deterministic, easy to implement.
- Cons: can pin an old/wrong node until TTL expiry or explicit unclaim.

### Option B: lease/claim token (safer for active ownership)

- `registerName:` returns a lease token with expiration.
- Only lease holder can refresh or release the name.
- Pros: clearer ownership, fewer accidental collisions.
- Cons: requires lease renewal and clock/timeout handling.

### Option C: allow multi-owner + client-side selection

- Keep multiple records for same name.
- Discovery returns ranked candidates; `NetworkChannel` picks by policy
  (nearest RSSI/latency, role tag, random, round-robin).
- Pros: resilient and load-balancing friendly.
- Cons: pushes complexity to clients/policies.

Recommended v0 approach:

- implement Option A immediately (deterministic winner + conflict reporting),
- keep conflict set visible via `NodeCache conflicts`,
- design Option B lease semantics for next iteration.

## Security (initial constraints)

- LAN scope only by default.
- Ignore malformed frames.
- Future: optional shared-key signature on announcements.

## Non-Goals (v0)

- cross-subnet routing
- persistent distributed consensus
- strong identity/auth in first release
