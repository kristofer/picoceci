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

Candidate messages:

- `Discovery start`
- `Discovery stop`
- `Discovery announce`
- `Discovery peers`
- `Discovery lookup: aName`
- `Discovery registerName: aName`

### `NodeCache` (optional singleton/service)

Core responsibilities:

- accept registrations from nodes (`register:name:addr:ttl:`)
- answer name/address lookups (`resolve:`)
- mirror recent announcements for warm restart

Candidate messages:

- `NodeCache startOn: port`
- `NodeCache stop`
- `NodeCache register: aNodeInfo`
- `NodeCache resolve: aName`
- `NodeCache entries`

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

## Runtime Model

- Boot flow:
  1. start `Task spawn: [ Discovery start ] name: 'discovery'`
  2. optionally start `NodeCache`
  3. register local service names
- Failure mode:
  - if multicast unavailable, fallback to NodeCache-only mode
  - if NodeCache unavailable, continue in peer-to-peer mode

## Jini/Go-Inspired Notes

- Jini-style service registration/lookup semantics adapted to constrained LAN use.
- Go channel-style event loop recommendation for implementation:
  - inbound announce channel
  - timer/ticker channel for TTL pruning
  - command channel for lookup/register API

## Security (initial constraints)

- LAN scope only by default.
- Ignore malformed frames.
- Future: optional shared-key signature on announcements.

## Non-Goals (v0)

- cross-subnet routing
- persistent distributed consensus
- strong identity/auth in first release
