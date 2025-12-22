# Manglekit Architecture Veto Rules

**Context:** Manglekit adopts an "Invisible Governance" philosophy. The Kernel (Framework) handles complexity; the Developer writes pure logic.
**Strictness:** These rules are **MANDATORY**. Any code violating them must be rejected.

### 1. ⛔ STRICT IMPORT BOUNDARIES
* **Rule:** User-space code (e.g., `examples/`, `main.go`) **MUST NEVER** import packages from `internal/`.
* **Allowed Imports:** `github.com/duynguyendang/manglekit`, `manglekit/core`, `manglekit/sdk`, `manglekit/adapters`.
* **Forbidden Imports:** `manglekit/internal/engine`, `manglekit/internal/guard`.
* *Why:* We enforce a strict Facade pattern. The engine is a black box.

### 2. ⛔ NO MANUAL "START/END" LOGGING
* **Rule:** Business logic functions **MUST NOT** log lifecycle events like "Action started", "Finished", or "Result is...".
* **Correction:** Rely entirely on the **Guard Middleware** to auto-log input, output, latency, and errors.
* **Exception:** Semantic logs inside the logic (e.g., "Cache hit", "Fallback triggered") are allowed but must use `core.LoggerFromContext(ctx)`.

### 3. ⛔ NO `fmt.Println` or `log.Println`
* **Rule:** Library and Action code **MUST NOT** use standard output printing.
* **Correction:** Always use the structured `core.Logger`.
* **Mechanism:** Retrieve it via `core.LoggerFromContext(ctx)`. If `ctx` is missing, fix the design to pass `ctx`.

### 4. ⛔ PREFER TYPE-SAFE DEFINITIONS
* **Rule:** Avoid using raw `client.RegisterAction` with `interface{}` or manual casting in user code.
* **Correction:** Always use `manglekit.Define[In, Out]` to create Actions.
* **Why:** Ensures compile-time type safety and reduces boilerplate code for the developer.

### 5. ⛔ ZERO 3RD-PARTY DEPS IN CORE
* **Rule:** The `core` and `root` packages **MUST NOT** import heavy 3rd-party logging/tracing libraries (e.g., `zap`, `logrus`).
* **Correction:** Use Go's standard `log/slog` (Go 1.21+) for default implementations.
* **Why:** Manglekit must be lightweight. External libs belong in `adapters/`.

### 6. ⛔ NO MANUAL COMPONENT WIRING
* **Rule:** Users **MUST NOT** be forced to manually initialize `engine`, `guard`, or `tracer` and wire them together.
* **Correction:** `manglekit.NewClient` is the **ONLY** entry point. It must auto-wire internal components behind the scenes.

### 7. ⛔ CONTEXT PROPAGATION IS MANDATORY
* **Rule:** Never drop the `context.Context`.
* **Correction:** Every layer (Client -> Guard -> Action -> Adapter) must pass the `ctx` downstream.
* **Why:** Dropping context breaks Distributed Tracing (TraceID) and Logger Injection.

### 8. ⛔ NO PANICS IN LIBRARY CODE
* **Rule:** Library code (`internal/`, `sdk/`) **MUST NEVER** panic.
* **Correction:** Always return `error` to the caller.
* **Exception:** Functions explicitly named `Must...` (e.g., `MustNewClient`) are allowed to panic on startup configuration errors.

### 9. ⛔ CONFIGURATION MUST HAVE DEFAULTS
* **Rule:** `manglekit.NewClient(ctx)` **MUST** work without any options.
* **Correction:** Provide "Batteries Included" defaults: `Slog` for logging, `NoOp` for memory, `Open` for failure mode.
* **Why:** Zero-Config DevEx. "It just works" out of the box.

### 10. ⛔ KEEP ROOT CLEAN (FACADE ONLY)
* **Rule:** The root `manglekit` package **MUST NOT** contain complex logic implementation.
* **Correction:** Root files (like `manglekit.go`) should only contain Type Aliases, Wrapper Functions, and Factory Methods that delegate to `sdk/` or `internal/`.

---