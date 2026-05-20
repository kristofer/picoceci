# Phase 1 Execution Checklist: Bridge Runtime Foundations

Status: Draft for implementation
Scope: Define and enforce a stable Bridge ABI v1 boundary and bridge build isolation, without changing default runtime behavior.

## Phase 1 Definition of Done

1. Bridge ABI v1 is documented with explicit ownership, timeout, and error semantics.
2. Experimental bridge code compiles behind a single build tag and does not affect default build/test paths.
3. Build tooling clearly distinguishes default and bridge flows, including failure diagnostics for unresolved runtime symbols.
4. A minimal verification matrix exists and passes for default flow.

## Ticket BR1-01: Bridge ABI v1 Spec Freeze

Goal: Lock the C/Go contract before adding new bridge services.

Files:

- docs/freertos-bridge.md
- targets/esp32s3_idf_bridge.c
- pkg/net/net_tinygo_esp32s3_idf.go

Tasks:

1. Add a Bridge ABI v1 section in docs/freertos-bridge.md covering:

- Function list (wifi stack init, connect/disconnect, tcp listen/accept/read/write/close).
- Parameter contracts (nullability, string encoding, timeout units).
- Ownership contracts for handles and buffers.
- Return code families and mapping to Go errors.

2. Align function comments in targets/esp32s3_idf_bridge.c with the documented ABI v1 semantics.
2. Ensure Go FFI declarations in pkg/net/net_tinygo_esp32s3_idf.go exactly match C signatures and timeout expectations.

Acceptance criteria per file:

- docs/freertos-bridge.md:
  - Contains a dedicated Bridge ABI v1 section with a complete function table.
  - Contains one error mapping table from bridge return codes to user-facing error kinds/messages.
  - States blocking rules for each call.
- targets/esp32s3_idf_bridge.c:
  - Every exported picoceci_bridge_* symbol has a short contract comment.
  - No exported symbol uses implicit ownership behavior.
  - No behavior contradicts docs/freertos-bridge.md ABI tables.
- pkg/net/net_tinygo_esp32s3_idf.go:
  - FFI declarations are a 1:1 match with bridge C exports.
  - Error paths include bridge return code in message text.
  - No implicit assumptions about success-only states.

## Ticket BR1-02: Build-Tag and Target Isolation

Goal: Guarantee bridge mode cannot regress default mode.

Files:

- pkg/net/net_tinygo.go
- pkg/net/net_tinygo_esp32s3_idf.go
- targets/esp32s3-n16r8.json
- targets/esp32s3-n16r8-idfbridge.json

Tasks:

1. Keep default TinyGo net implementation active only when bridge tag is absent.
2. Keep bridge net implementation active only when esp32s3_idf_bridge tag is present.
3. Verify default target JSON does not include bridge-specific tags or extra files.
4. Verify idfbridge target JSON includes only the minimum bridge additions.

Acceptance criteria per file:

- pkg/net/net_tinygo.go:
  - Build tag excludes bridge mode.
  - File compiles in default tinygo mode.
- pkg/net/net_tinygo_esp32s3_idf.go:
  - Build tag requires bridge mode.
  - File does not compile into non-bridge default build.
- targets/esp32s3-n16r8.json:
  - No esp32s3_idf_bridge tag.
  - No bridge C file in extra-files.
- targets/esp32s3-n16r8-idfbridge.json:
  - Includes esp32s3_idf_bridge tag.
  - Includes bridge C file wiring (directly or via generated local target).

## Ticket BR1-03: Tooling and Diagnostics Hardening

Goal: Make bridge and default workflows explicit and debuggable.

Files:

- Makefile
- README.md

Tasks:

1. Keep separate default and bridge targets (esp32-build vs esp32-build-idf).
2. Add concise diagnostic output for bridge build failures that indicates likely unresolved IDF/LwIP symbol causes.
3. Document expected behavior and known limitations of bridge mode in README.md.

Acceptance criteria per file:

- Makefile:
  - Default targets do not depend on bridge target generation.
  - Bridge targets always use bridge-specific target JSON generation path.
  - Bridge build failure output includes a next-step hint.
- README.md:
  - Contains a bridge mode section with explicit caveat status.
  - Contains command examples for default mode and bridge mode.
  - Explains when to use host TCP bridge fallback.

## Ticket BR1-04: Runtime Guardrails in App Entry

Goal: Preserve recovery path and avoid boot lockouts during bridge experiments.

Files:

- target/esp32s3/main.go
- pkg/tinygo/console_tinygo.go

Tasks:

1. Keep serial console path available even when bridge WiFi/TCP path fails.
2. Ensure startup logs provide mode and failure reason context.
3. Keep bridge failure non-fatal for local serial REPL availability.

Acceptance criteria per file:

- target/esp32s3/main.go:
  - Bridge initialization errors are logged and do not prevent serial REPL startup.
  - Listener failures do not crash the process loop.
  - At least one clear line indicates selected network mode path.
- pkg/tinygo/console_tinygo.go:
  - Console initialization still succeeds in default and bridge builds.
  - No early allocation/logging regressions are introduced.

## Ticket BR1-05: Verification Matrix and Baseline Gates

Goal: Establish repeatable checks before Phase 2 feature expansion.

Files:

- pkg/net/net_test.go
- docs/IMPLEMENTATION_PLAN.md

Tasks:

1. Add or confirm tests for state transitions and listener/session behavior in host-compatible paths.
2. Record a Phase 1 verification matrix in docs/IMPLEMENTATION_PLAN.md.

Acceptance criteria per file:

- pkg/net/net_test.go:
  - Includes tests for connect status transitions and listen lifecycle behavior.
  - Includes close semantics checks for listener/session.
- docs/IMPLEMENTATION_PLAN.md:
  - Contains explicit Phase 1 gates with pass/fail criteria.
  - Distinguishes mandatory default-path pass vs bridge-path experimental pass.

## Execution Order

1. BR1-01
2. BR1-02
3. BR1-03
4. BR1-04
5. BR1-05

## Command Gate Checklist

Run and record outcomes for each ticket completion:

1. make esp32-build
2. make esp32-build-idf
3. go test ./pkg/net ./pkg/eval
4. go test ./...

Expected gate policy:

- Default path gates must pass.
- Bridge path may fail only for known unresolved external symbols, and failure output must remain actionable.
