# picoceci Networking Curriculum

A progressive curriculum for building networks of ESP32-S3 nodes using picoceci.
Each module adds one concept; by the end you have a self-healing mesh that could run
aboard a real spacecraft.

## Prerequisites

Complete the [picoceci Beginner's Guide](../picoceci-beginners-guide.md) first.
You should be comfortable with:

- Object declarations (`object`, `compose`)
- Task concurrency (`Task spawn:name:`, `TaskSupervisor`)
- Typed channels (`Channel`, `Queue`)
- The LED/GPIO/UART hardware singletons

## Hardware per Student Group

| Qty | Item |
|-----|------|
| 3–5 | ESP32-S3-N16R8 dev boards |
| 3–5 | USB-C cables (data-capable) |
| 1   | WiFi access point (shared, 2.4 GHz) |
| –   | Jumpers and breadboard (optional sensor wiring) |

All nodes share the same WiFi network. Note the AP's SSID and password — you will
hard-code them in early modules and factor them out later.

## Modules

| # | Module | Concept | Nodes needed |
|---|--------|---------|-------------|
| [01](01-single-node-wifi.md) | Single Node WiFi | Connect, serve over TCP, use the REPL remotely | 1 |
| [02](02-point-to-point.md) | Point-to-Point | Two nodes exchange typed messages | 2 |
| [03](03-star-topology.md) | Star Topology | Many sensors, one hub, concurrent sessions | 3–5 |
| [04](04-request-response.md) | Request-Response | Controller issues commands; nodes reply | 2–3 |
| [05](05-manual-registry.md) | Manual Registry | Nodes discover each other through a registry node | 3–5 |
| [06](06-discovery.md) | Discovery Protocol | Design walkthrough: how picoceci Discovery works | (any) |
| [07](07-mesh-network.md) | Mesh Network | No hub, no registry — pure peer-to-peer with gossip | 4–5 |

## Example Files

All runnable `.pc` programs live in `examples/networking/`. Each module lists which
files it uses and the order to flash them.

```
examples/networking/
  01_wifi_connect.pc        – M01: WiFi connection lifecycle
  02_sensor_server.pc       – M01: Temperature sensor served over TCP
  03_collector_node.pc      – M02: Collector receives push data from a sensor
  04_ping_sender.pc         – M02: Sends periodic pings; expects pong
  04_pong_receiver.pc       – M02: Receives pings; sends pong replies
  05_hub_node.pc            – M03: Hub accepts N concurrent spoke connections
  05_spoke_node.pc          – M03: Spoke connects to hub, reports readings
  06_command_controller.pc  – M04: Issues commands to managed nodes
  06_bidirectional_sensor.pc– M04: Reports readings AND accepts commands
  07_registry_node.pc       – M05: Hosts the name→address registry
  07_registered_node.pc     – M05: Registers with registry; queries for peers
  08_mesh_node.pc           – M07: Full mesh node with routing table and gossip
```

## Wire Protocol Convention

All multi-node examples use a simple newline-terminated text protocol:

```
TYPE:PAYLOAD\n
```

Examples:

| Message | Meaning |
|---------|---------|
| `READING:23.5\n` | Sensor reading of 23.5 |
| `CMD:calibrate\n` | Command: run calibration |
| `ACK:ok\n` | Acknowledgement, success |
| `REGISTER:sensor-a:192.168.1.101\n` | Registry registration |
| `QUERY:collector\n` | Registry lookup by name |
| `ADDR:192.168.1.105\n` | Registry response with address |
| `HELLO:mesh-node-3\n` | Mesh handshake |
| `ROUTE:sensor-a,sensor-b,collector\n` | Gossip: known peers |

This protocol is intentionally simple so you can test nodes with `nc` (netcat)
before wiring picoceci nodes together.

## Development Tips

**Flash and log.** Use `picoceci run` with the serial monitor open. The first thing
every node prints is its IP address — write it down or assign static DHCP leases.

**Test with netcat first.** Before connecting two boards:

```sh
nc 192.168.1.101 7001
```

Type a message, press Enter, watch the node respond. This isolates firmware bugs
from network bugs.

**LED as a heartbeat.** Every example blinks the LED while running. If the LED stops,
the program has crashed or the main task exited.

**Task isolation.** Each connection runs in its own Task. A crash in one session
never kills the others. Use `TaskSupervisor` to restart crashed tasks automatically.
