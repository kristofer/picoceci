# picoceci: A Small Language for Big Dreams
## A Whitepaper on Message-Passing Objects for the IoT Age

*Version 0.1 · May 2026*

---

> *"I once dreamed of a spacecraft that could fix itself. Not the grand, gilded kind you see in the movies — no warp drives, no photon torpedoes — just a real ship, humming through the dark, tended by a thousand tiny electronic watchmen. Each one small enough to hold in your palm, each one smart enough to know when something was wrong. This is that dream, put into code."*

---

## Prologue: A Letter from Your Uncle

Pull up a chair. I've been thinking about this for a while, and I want to tell you about a little language I've been dreaming up. It's called **picoceci**, and it's probably the most fun I've had since I first typed `print "hello"` on a machine that weighed as much as a refrigerator.

You see, for decades now I've watched two opposing forces battle it out in the software world. On one side, languages that are powerful but dense — all braces and semicolons and type declarations stacked three lines high. On the other side, languages that are friendly and readable but collapse under the weight of any real complexity. What I wanted was something in between. Something that a student could pick up in an afternoon, but that could also run — really *run* — on a microcontroller the size of a postage stamp, aboard an imagined spacecraft, managing the air you breathe.

That language is picoceci. Let me tell you about it.

---

## 1. What Is picoceci?

picoceci (pronounced "pee-ko-cheh-chee" — or however you like, really) is a small, message-passing language with a big heart. It borrows Smalltalk's joyful syntax and Go's no-nonsense composability. It runs on microcontrollers via [TinyGo](https://tinygo.org/) and is designed from the ground up for distributed, networked IoT systems.

The name is a nod to *pico* (small, in the spirit of microcontrollers) and *ceci* (Italian for chickpea — small, high-protein, full of goodness). Like a chickpea, picoceci is small, but it packs a punch.

### Core Tenets

| Principle | What It Means in Practice |
|---|---|
| **Everything is an object** | Numbers, booleans, arrays, tasks — all objects, all responsive to messages |
| **Messages, not calls** | You don't *call* a method; you *send a message* to an object |
| **Composition, not inheritance** | Objects are built by composing other objects, not by climbing class hierarchies |
| **Interfaces, not types** | Any object that responds to the right messages satisfies an interface |
| **Small footprint** | The whole runtime fits in under 128 KB of RAM on an ESP32-S3 |
| **Concurrent by nature** | Tasks, queues, semaphores, and channels are first-class citizens |

---

## 2. The Vision: A Network of Tiny Watchmen

Here is the dream that drives this whole project. Imagine a spacecraft — a real one, not the Hollywood kind. Not the kind with a single HAL-9000 watching over everything and going quietly mad. Instead: hundreds of small microcontrollers scattered across the ship like a nervous system. Each one is a node. Each one watches something specific: the CO₂ level in a corridor, the humidity in the greenhouse compartment, the temperature around the fuel cells, the pressure in an airlock.

Each node runs a small picoceci program — a *Task*, in our parlance. Each Task is simple, focused, easy to understand and easy to replace if it fails. The nodes talk to each other over a network, passing messages, sharing data, raising alerts. If one fails, the others adapt. The ship doesn't have a single brain that can break; it has a resilient, self-healing mesh of small, smart, dedicated sensors and actuators.

This is not science fiction. The technology exists today. What has been missing is a language expressive enough to make programming these devices *enjoyable* — enjoyable enough that students will learn it, experiment with it, break things with it, and ultimately build something wonderful with it.

picoceci is meant to be that language.

---

## 3. Why Smalltalk Syntax?

"Smalltalk?" I hear you say. "Isn't that ancient? Didn't that die in the nineties?"

Well, yes and no. The *commercial ecosystem* around Smalltalk faded, but the *ideas* in Smalltalk are as fresh as ever. Alan Kay's vision of objects communicating by message is, if anything, more relevant today than it was in 1972. When you have hundreds of devices on a network, each one an independent agent responding to requests, you are living inside a Smalltalk program whether you call it that or not.

Here's what Smalltalk syntax looks like in picoceci:

```picoceci
"Send the message 'println:' to the Console object"
Console println: 'Hello from orbit!'.

"Ask a temperature sensor for its reading"
| temp |
temp := cabin temperatureSensor reading.
Console println: 'Cabin temp: ', temp printString, ' C'.
```

That's it. No semicolons at the end of every line (the `.` is optional in many places). No curly braces wrapping everything. No type declarations cluttering the page. It reads almost like English — or at least like a polite telegram.

The three types of message in picoceci follow Smalltalk's elegant hierarchy:

- **Unary messages** — `sensor reading`, `led toggle`, `task suspend` — one word, sent to an object, no arguments.
- **Binary messages** — `temp > 30`, `count + 1`, `name , ' OK'` — an operator, one argument.  
- **Keyword messages** — `Queue new: 10`, `Task spawn: [ ... ]`, `timer every: 1000 do: [ ... ]` — one or more colon-terminated keywords, each followed by an argument.

This three-tier system gives you tremendous expressiveness with almost no syntactic noise. A novice can read picoceci code on their first day and have a pretty good idea of what it does. That matters enormously when you are teaching students to write resilient embedded software.

---

## 4. Why Go Semantics?

Syntax is the face of a language; semantics are its soul. Under picoceci's friendly Smalltalk face beats a Go-like heart.

### Composition Over Inheritance

Go famously has no class inheritance. Instead, you embed one struct inside another and the outer struct automatically gets the methods of the inner one. This simple mechanism, once you understand it, renders class hierarchies almost entirely unnecessary.

picoceci does the same thing, but with objects and the `compose` keyword:

```picoceci
object Sensor {
    | name lastReading |
    init: aName [ name := aName. lastReading := nil ]
    name        [ ^name ]
    reading     [ ^lastReading ]
    update: v   [ lastReading := v ]
    printString [ ^name , ': ' , lastReading printString ]
}

object AlertingSensor {
    compose Sensor.
    | threshold alertChannel |
    init: aName threshold: t channel: ch [
        super init: aName.
        threshold := t.
        alertChannel := ch
    ]
    update: v [
        super update: v.
        v > threshold ifTrue: [
            alertChannel <- ('ALERT: ' , name , ' = ' , v printString)
        ]
    ]
}
```

`AlertingSensor` *is* a `Sensor` — it has all the same methods — but it adds alerting behaviour without any of the machinery of inheritance. There is no class hierarchy to navigate, no method resolution order to memorise, no fragile base class problem. The `compose` keyword copies slots and methods cleanly, like Go embedding.

### Interfaces

Go uses *structural typing*: if a value has all the methods of an interface, it satisfies the interface — no declaration required. picoceci works the same way:

```picoceci
interface TemperatureSensor {
    reading
    update: v
}
```

Any object that responds to `reading` and `update:` satisfies `TemperatureSensor`. You can write functions (or in picoceci's case, blocks and methods) that accept a `TemperatureSensor` and they will work with any compliant object — real hardware sensors, simulated test sensors, or anything else you dream up.

### Channels

Go's channels are one of its most celebrated features — a clean, safe way for concurrent routines to communicate without shared memory. picoceci has channels too:

```picoceci
| readings alertChan |
readings  := Channel new: 20.
alertChan := Channel new: 5.

"Sensor task: poll and publish readings"
Task spawn: [
    [ true ] whileTrue: [
        readings <- (cabin temperatureSensor reading).
        Task delay: 5000
    ]
] name: 'temp-publisher'.

"Monitor task: watch for danger"
Task spawn: [
    [ true ] whileTrue: [
        | t |
        t := <-readings.
        t > 30 ifTrue: [ alertChan <- ('HIGH TEMP: ' , t printString) ]
    ]
] name: 'temp-monitor'.
```

Channels decouple producers from consumers. Each Task does one thing well, communicates only through well-defined channels, and can be tested, replaced, or restarted without affecting the rest of the system. That is exactly what you want in a fault-tolerant network of spacecraft MCUs.

---

## 5. Objects All the Way Down

One of Smalltalk's most powerful ideas — and one that picoceci fully embraces — is that *everything* is an object. Not "almost everything" with a few primitive exceptions. *Everything*.

The number `42`? It's an object. Send it messages:

```picoceci
42 printString.      "=> '42'"
42 asFloat.          "=> 42.0"
(42 between: 1 and: 100) ifTrue: [ Console println: 'in range' ].
```

The block `[ :x | x * 2 ]`? An object. Store it, pass it around, invoke it later:

```picoceci
| doubler |
doubler := [ :x | x * 2 ].
Console println: (doubler value: 21) printString.  "=> 42"
```

`true` and `false`? Objects:

```picoceci
| sensor |
sensor sensorOK
    ifTrue:  [ Console println: 'Sensor nominal' ]
    ifFalse: [ Console println: 'Sensor fault — check connections' ].
```

This uniformity is not just aesthetic. It means that the mental model you need to reason about picoceci code is simple and consistent: *objects receive messages and return objects*. No special cases. No syntax for "this is a primitive operation and that is a method call." One rule, applied everywhere.

---

## 6. A Spacecraft in Code: Practical Examples

Let me show you what a real picoceci program for an IoT node might look like. These examples are meant for students — they start simple and get progressively more interesting.

### 6.1 Hello, Sensor

The simplest node: read a temperature sensor and print it to the serial console.

```picoceci
"temperature_hello.pc — the 'Hello World' of sensor programming"
| temp |
temp := I2C new: 0 sda: 21 scl: 22 speed: 400000.
[ true ] whileTrue: [
    | reading |
    reading := temp readFrom: 16r48 count: 2.
    Console println: 'Temperature: ' , reading printString , ' raw'.
    Task delay: 2000
].
```

Even here, the structure is clear: open the I²C bus, loop forever reading and printing. Two thousand milliseconds between readings — no busy-waiting, yielding control back to the FreeRTOS scheduler between each sample.

### 6.2 A Proper Sensor Object

Now let's wrap that in an object, giving it a name and a clean interface:

```picoceci
"TempSensor.pc"
object TempSensor {
    | bus address lastC |
    init: aBus address: anAddr [
        bus := aBus.
        address := anAddr.
        lastC := 0.0
    ]
    poll [
        | raw |
        raw := bus readFrom: address count: 2.
        lastC := ((raw at: 1) * 256 + (raw at: 2)) / 16.0.
        ^self
    ]
    celsius    [ ^lastC ]
    fahrenheit [ ^lastC * 1.8 + 32 ]
    printString [ ^'TempSensor(' , lastC printString , 'C)' ]
}

| i2c sensor |
i2c    := I2C new: 0 sda: 21 scl: 22 speed: 400000.
sensor := TempSensor new init: i2c address: 16r48.

[ true ] whileTrue: [
    sensor poll.
    Console println: sensor printString.
    Task delay: 2000
].
```

Now the sensor is a named, reusable object. It hides the I²C protocol details. The main loop is three lines long and easy to understand.

### 6.3 Environmental Monitor: Air Quality, Temperature, Humidity

A real environmental node might watch several parameters at once. Here is how picoceci handles it — each sensor in its own Task, all reporting through a shared channel:

```picoceci
"EnvMonitor.pc — multi-sensor environmental node"
import 'TempSensor'.
import 'HumiditySensor'.
import 'CO2Sensor'.

| reportChan i2c |
reportChan := Channel new: 30.
i2c := I2C new: 0 sda: 21 scl: 22 speed: 400000.

"Temperature task"
Task spawn: [
    | s |
    s := TempSensor new init: i2c address: 16r48.
    [ true ] whileTrue: [
        s poll.
        reportChan <- { #temp. s celsius }.
        Task delay: 5000
    ]
] name: 'temp'.

"Humidity task"
Task spawn: [
    | s |
    s := HumiditySensor new init: i2c address: 16r44.
    [ true ] whileTrue: [
        s poll.
        reportChan <- { #humidity. s percent }.
        Task delay: 10000
    ]
] name: 'humidity'.

"CO₂ task"
Task spawn: [
    | s |
    s := CO2Sensor new init: i2c address: 16r62.
    [ true ] whileTrue: [
        s poll.
        reportChan <- { #co2. s ppm }.
        Task delay: 15000
    ]
] name: 'co2'.

"Reporter task — format and log all readings"
Task spawn: [
    [ true ] whileTrue: [
        | msg |
        msg := <-reportChan.
        Console println: (msg at: 1) printString , ': ' , (msg at: 2) printString
    ]
] name: 'reporter'.
```

Notice what has happened here. Each sensor is a separate Task. They all run concurrently, each on its own polling interval. They communicate only through `reportChan` — there is no shared state, no mutex, no possibility of one sensor task accidentally corrupting another's data. The reporter task is blissfully ignorant of which sensor sent a reading or when; it just pulls messages from the channel and logs them.

This pattern — many independent Tasks, communicating through typed channels — is the heartbeat of picoceci's concurrency model. And it scales: you can add a fourth sensor (CO₂? pressure? radiation?) by copying twelve lines of code.

### 6.4 The Alert System: Composition in Action

Now let us add alerting. Rather than modify any existing sensor object, we *compose* a new one:

```picoceci
"AlertingTempSensor.pc"
import 'TempSensor'.

object AlertingTempSensor {
    compose TempSensor.
    | alertThreshold alertChan |
    init: aBus address: anAddr threshold: t channel: ch [
        super init: aBus address: anAddr.
        alertThreshold := t.
        alertChan := ch
    ]
    poll [
        super poll.
        self celsius > alertThreshold ifTrue: [
            alertChan <- ('TEMP ALERT: ' , self celsius printString , 'C in cabin A')
        ].
        ^self
    ]
}
```

`AlertingTempSensor` has all the behaviour of `TempSensor` (because it composes it), plus it sends an alert whenever the temperature exceeds a threshold. The original `TempSensor` is untouched. The logic is in one place, easy to find and easy to test.

This is what the issue calls "the bright ideas of Go": composition as the primary mechanism for building complex behaviour from simple pieces.

### 6.5 Self-Healing: Watchdog Tasks

A spacecraft cannot afford to have a sensor node crash and stay crashed. picoceci's error handling and task model make it straightforward to write self-healing nodes:

```picoceci
"WatchdogSensor.pc — restart a sensor task if it crashes"
import 'TempSensor'.

| i2c |
i2c := I2C new: 0 sda: 21 scl: 22 speed: 400000.

[ true ] whileTrue: [
    [
        | sensor |
        sensor := TempSensor new init: i2c address: 16r48.
        [ true ] whileTrue: [
            sensor poll.
            Console println: sensor printString.
            Task delay: 5000
        ]
    ] on: Error do: [ :err |
        Console println: 'Sensor task error: ' , err messageText.
        Console println: 'Restarting in 3 seconds...'.
        Task delay: 3000
    ]
].
```

If the I²C bus glitches, if the sensor returns garbled data, if any error occurs — the outer loop catches it, logs it, waits three seconds, and tries again. The node never stays dead. This is exactly the kind of robustness you need when the nearest repair technician is a hundred thousand kilometres away.

---

## 7. The Network Layer: Many Nodes, One Organism

So far we have been looking at individual nodes. But the real power of picoceci emerges when the nodes talk to each other.

Imagine a habitat module aboard a spacecraft. There might be sixteen environmental sensor nodes, each an ESP32-S3 running picoceci. One of them is designated the *aggregator* for its zone; the others are *reporters*. The reporters send their readings to the aggregator over a lightweight message protocol. The aggregator collates the data, detects anomalies, and forwards summary reports to the ship's central systems.

In picoceci, this network communication would look much like local channel communication — the `NetworkChannel` abstraction hides whether the other end is a local task or a remote node:

```picoceci
"ZoneAggregator.pc — collects readings from up to 16 sensor nodes"
import 'NetworkChannel'.

| incoming |
incoming := NetworkChannel listenOn: 7001.

| readings |
readings := Dictionary new.

Task spawn: [
    [ true ] whileTrue: [
        | msg nodeId sensorType value |
        msg        := <-incoming.
        nodeId     := msg at: #node.
        sensorType := msg at: #sensor.
        value      := msg at: #value.
        readings at: nodeId put: (readings at: nodeId ifAbsent: [ Dictionary new ])
                 thenDo: [ :d | d at: sensorType put: value ].
        Console println: 'Zone A: ' , nodeId printString ,
                         ' ' , sensorType printString ,
                         ' = ' , value printString
    ]
] name: 'aggregator'.
```

The node sending data looks like this:

```picoceci
"SensorNode.pc — reports to zone aggregator"
import 'TempSensor'.
import 'NetworkChannel'.

| i2c sensor outgoing nodeId |
nodeId  := 'node-A7'.
i2c     := I2C new: 0 sda: 21 scl: 22 speed: 400000.
sensor  := TempSensor new init: i2c address: 16r48.
outgoing := NetworkChannel connectTo: '192.168.1.10' port: 7001.

[ true ] whileTrue: [
    sensor poll.
    outgoing <- { #node. nodeId. #sensor. #temp. #value. sensor celsius }.
    Task delay: 10000
].
```

Each node knows its own ID, knows where to report, and does its job. If the aggregator is unreachable, the channel buffers the messages. If the node restarts, it reconnects and resumes. The network is resilient because each component is small, focused, and independent.

---

## 8. Why This Matters for Education

I want to take a moment and talk about students. This language is, at its heart, a teaching tool.

One of the perennial difficulties in embedded systems education is that the gap between "Hello, World" and "real application" is enormous. Students learn to blink an LED in C and then feel utterly lost when asked to write a multi-sensor monitoring system. The conceptual scaffolding — tasks, queues, state machines, error handling — is all there in FreeRTOS or Arduino, but the syntax and idioms are so far removed from the student's mental model that the learning curve feels like a cliff.

picoceci is designed to lower that cliff. Because:

1. **The object model is simple and consistent.** One rule: send a message, get a response. Students can start here and trust that this model applies everywhere.

2. **Composition is learnable.** Students can understand "an AlertingSensor is a Sensor that also sends alerts" long before they can reason about class hierarchies, virtual dispatch, and the Liskov substitution principle.

3. **Tasks and channels are intuitive.** "This task watches the temperature. This task watches the CO₂. They report through this channel to the logger." That sentence maps almost directly to picoceci code. The concepts and the code are aligned.

4. **Error handling is explicit and local.** The `on: Error do:` pattern makes it clear where errors are expected and what to do about them. There are no hidden exception propagation rules to memorise.

5. **Small programs do real things.** A picoceci program that monitors a temperature sensor and prints alerts is twenty lines long and runs on real hardware. Students can hold the whole program in their head at once — and then extend it.

A student who works through the examples in this whitepaper will come away with:

- An understanding of message-passing objects
- Practical experience with concurrent tasks and channels
- A working mental model of composition-based code reuse
- Intuitions about fault tolerance and self-healing systems
- The beginning of an appreciation for what distributed sensor networks can do

Those are not small things. Those are the intellectual tools that the next generation of aerospace engineers, embedded systems developers, and IoT architects will need.

---

## 9. The Canal Connection

picoceci does not run in isolation. It is designed to run on top of **Canal**, a capability-based microkernel for TinyGo.

A capability kernel is one in which access to resources — files, network sockets, GPIO pins, I²C buses — is represented as an *unforgeable token* that you must hold to use the resource. You cannot access a resource you were not explicitly given a capability for. This is a powerful security model for networked IoT devices: a compromised node cannot reach beyond the resources it was initially granted.

picoceci objects map naturally onto Canal capabilities. A `TempSensor` object, in the Canal model, *is* a capability object — you can pass it to another task, compose it into a larger object, or revoke it, and the security properties are enforced at the kernel level. No raw pointers, no arbitrary memory access, no possibility of one node's code reaching into another node's address space.

For spacecraft applications, this is not just nice to have. It is essential. A software fault in the humidity sensor node should not be able to corrupt the data from the CO₂ sensor, let alone affect the flight control system. Canal's capability model, combined with picoceci's message-passing objects and isolated Tasks, provides the foundation for that kind of strong isolation.

---

## 10. Relation to Other Languages

It would be dishonest to present picoceci as wholly novel. It stands on the shoulders of some remarkable languages:

| Inspiration | What picoceci borrows |
|---|---|
| **Smalltalk** | Message syntax, blocks, "everything is an object", symbol literals |
| **Go** | Structural interfaces, composition over inheritance, channels, goroutine-like tasks |
| **Self** | Prototype-like object composition (though picoceci uses named objects, not prototypes) |
| **Pharo** | Modern Smalltalk idioms; proof that Smalltalk ideas are still vital |
| **Erlang** | The philosophy of many small isolated processes communicating by message |
| **MicroPython** | The existence proof that a high-level language *can* run well on microcontrollers |

What picoceci tries to do that none of these does in quite the same way is combine Smalltalk's approachability with Go's composability and put the result inside a sub-128-KB MCU runtime designed for a networked, fault-tolerant world.

---

## 11. Current Status and Roadmap

picoceci is in active development. Here is where things stand:

| Phase | Status | Description |
|---|---|---|
| Language specification | ✅ Complete | Full EBNF grammar, type system, concurrency model |
| Lexer & Parser | ✅ Complete | Handles all language constructs |
| Tree-walking interpreter | ✅ Complete | Runs picoceci programs on desktop (Go host) |
| Bytecode compiler & VM | 🚧 In progress | For better MCU performance |
| TinyGo / ESP32-S3 target | 🚧 In progress | Interpreter embedded in TinyGo binary |
| Canal integration | 📋 Planned | Capability-kernel IPC bridge |
| Standard library | 🚧 In progress | GPIO, I²C, SPI, UART, network |
| Developer tooling | 📋 Planned | Debugger, profiler, VS Code extension |

The desktop interpreter already runs, which means you can begin learning and experimenting with picoceci today on any machine that runs Go. The MCU target follows once the bytecode VM is stable.

---

## 12. Getting Started

If you want to try picoceci today, here is how:

```bash
git clone https://github.com/kristofer/picoceci
cd picoceci
go build ./...
./picoceci repl
```

You will find yourself in the REPL. Try:

```picoceci
Console println: 'Hello from picoceci!'.
```

Then try:

```picoceci
object Counter {
    | count |
    init  [ count := 0 ]
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}
| c |
c := Counter new.
c inc; inc; inc.
Console println: c value printString.
```

Then look in the `testdata/programs/` directory for more examples. Then write your own.

For the full language reference, see [LANGUAGE_SPEC.md](../LANGUAGE_SPEC.md). For implementation details, see [IMPLEMENTATION_PLAN.md](../IMPLEMENTATION_PLAN.md).

---

## 13. A Final Word from Your Uncle

I want to close the way I opened: with a dream.

I dream of a spacecraft — a real one — where the air quality in every corridor is monitored by a small, inexpensive device the size of a matchbox, running a picoceci program written by a student who had never touched embedded systems before the semester started. Where that device talks to seventeen others like it, and together they build a picture of the ship's environmental health that is richer and more reliable than any single centralised sensor system could provide. Where, when one of those devices fails — because things fail in space; space is *hard* — the others adapt, the network heals, and nobody dies because a single point of failure took down the whole monitoring system.

I dream of students learning to think in messages and objects and channels, and then carrying those ideas into careers in aerospace, medicine, urban infrastructure, climate monitoring — all the places where small, smart, networked devices are going to matter enormously in the decades ahead.

I dream of code that is small enough to understand completely, elegant enough to be a pleasure to read, and robust enough to be trusted with real responsibilities.

picoceci is my attempt to make that dream concrete. It's a small language with a big heart, and there's plenty of room for you in it.

Come build something.

---

## Appendix A: Language Quick Reference

### Literals

```picoceci
42          "integer"
3.14        "float"
'hello'     "string"
#hello      "symbol"
$A          "character"
true false nil
#(1 2 3)    "array literal"
#[1 2 3]    "byte array"
```

### Variables and Assignment

```picoceci
| x y z |          "declare locals"
x := 42.           "assignment"
```

### Messages

```picoceci
sensor reading              "unary"
count + 1                   "binary"
Queue new: 10               "keyword"
q send: 42 timeout: 500     "multi-keyword"
led toggle; blink; off.     "cascade (same receiver)"
```

### Objects

```picoceci
object Foo {
    | slot1 slot2 |
    init: a and: b [ slot1 := a. slot2 := b ]
    sum            [ ^slot1 + slot2 ]
}
| f |
f := Foo new init: 3 and: 4.
Console println: f sum printString.   "=> 7"
```

### Composition

```picoceci
object Bar {
    compose Foo.
    doubled [ ^super sum * 2 ]
}
```

### Interfaces

```picoceci
interface Readable {
    reading
}
```

### Blocks

```picoceci
[ :x | x * 2 ] value: 5.   "=> 10"
[ 1 + 1 ] value.            "=> 2"
```

### Control Flow

```picoceci
x > 0 ifTrue: [ ... ] ifFalse: [ ... ].
1 to: 10 do: [ :i | ... ].
[ condition ] whileTrue: [ ... ].
5 timesRepeat: [ ... ].
```

### Concurrency

```picoceci
| ch |
ch := Channel new: 10.

Task spawn: [ ch <- 42 ] name: 'sender'.
Task spawn: [
    | v |
    v := <-ch.
    Console println: v printString
] name: 'receiver'.
```

### Error Handling

```picoceci
[ riskyOperation ]
    on: Error
    do: [ :err | Console println: err messageText ].

[ file read ] ensure: [ file close ].
```

---

## Appendix B: Sensor Code Patterns

### Pattern 1: Poll-and-publish

```picoceci
"A task that polls a sensor and publishes to a channel."
| sensor ch |
sensor := TempSensor new init: i2c address: 16r48.
ch := Channel new: 10.

Task spawn: [
    [ true ] whileTrue: [
        sensor poll.
        ch <- sensor celsius.
        Task delay: 5000
    ]
] name: 'temp-poller'.
```

### Pattern 2: Threshold alert

```picoceci
"A task that monitors a channel and raises alerts."
Task spawn: [
    [ true ] whileTrue: [
        | v |
        v := <-ch.
        v > 28.0 ifTrue: [
            alertChan <- ('HIGH TEMP: ' , v printString , 'C')
        ]
    ]
] name: 'temp-alerter'.
```

### Pattern 3: Self-healing loop

```picoceci
"Restart a sensor task if it crashes."
[ true ] whileTrue: [
    [ sensorLoop ] on: Error do: [ :e |
        Console println: 'Sensor error: ' , e messageText.
        Task delay: 3000
    ]
].
```

### Pattern 4: Multi-sensor aggregator

```picoceci
"Compose readings from multiple sensors into one report."
| sensors reportChan |
sensors := Array
    with: (TempSensor new init: i2c address: 16r48)
    with: (HumiditySensor new init: i2c address: 16r44).
reportChan := Channel new: 20.

sensors do: [ :s |
    Task spawn: [
        [ true ] whileTrue: [
            s poll.
            reportChan <- s.
            Task delay: 10000
        ]
    ] name: s name
].

Task spawn: [
    [ true ] whileTrue: [
        | s |
        s := <-reportChan.
        Console println: s printString
    ]
] name: 'reporter'.
```

---

*picoceci is MIT licensed. Contributions welcome at https://github.com/kristofer/picoceci*
