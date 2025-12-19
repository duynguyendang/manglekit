---
context_type: kernel_source_dump
project: manglekit
language: go, datalog
last_updated: 2025-12-17
scan_mode: logic_focused
---

## 1. THE COMPLETE FILE MAP

```text
.
├── AGENTS.md
├── adapters/                  # Integration layer for external capabilities
│   ├── ai/                    # Genkit wrapper for LLMs
│   ├── extractor/             # Structured data extraction actions
│   ├── func/                  # Generic Go function wrapper (Reflection)
│   ├── knowledge/             # RDF/Graph knowledge base adapters
│   ├── logger/                # Logging adapters (Zap, Slog)
│   ├── mcp/                   # Model Context Protocol (MCP) client
│   ├── resilience/            # Circuit Breakers and Retries
│   └── vector/                # Vector Store adapters
├── cmd/                       # Command-line tools
│   └── mkit/                  # The 'mkit' CLI
├── config/                    # Configuration loading and schema
├── core/                      # Pure domain contracts (Interfaces & Types)
├── docs/                      # Documentation and Architecture Rules
├── internal/                  # Internal logic (The Kernel)
│   ├── engine/                # The Neuro-Symbolic Core (Mangle Runtime)
│   ├── logger/                # Internal logging utilities
│   ├── resources/             # Embedded Datalog rules (std.dl, planner.dl)
│   ├── statehelper/           # Conversation state management
│   ├── supervisor/            # The Governance Supervisor (Decorator)
│   ├── telemetry/             # OpenTelemetry helpers
│   ├── testproviders/         # Test mocks
│   ├── tools/                 # Internal CLI tools
│   └── util/                  # Utilities (Schema, Wrapper)
├── providers/                 # Plugin registry for standard providers
└── sdk/                       # Public API and Orchestration
```

## 2. COMPONENTS (The Logic)

### Core (`core/`)
> **Responsibilities:** Defines the immutable contracts and types that bind the system together.
*   `types.go`: Defines `Envelope`, `ActionMetadata`, `Decision`, and Control Plane constants.
*   `governance.go`: Defines `Evaluator` (Engine) interface.
*   `logic.go`: Defines `Action` interface.
*   `infra.go`: Defines `Tracer`, `Logger` interfaces.
*   `data.go`: Defines `MemoryStore`, `FactLoader`.

### SDK (`sdk/`)
> **Responsibilities:** Orchestrates the execution loop and provides the developer-facing API.
*   `Client`: The main entry point. Manages the Engine, Memory, and Action Registry.
*   `ExecuteByName` (`loop.go`): The Semantic State Machine that drives the execution lifecycle (Action -> Decision -> Retry/Route).
*   `Proxy` (`proxy.go`): Exposes registered actions as `core.Action` instances.

### Internal Engine (`internal/engine/`)
> **Responsibilities:** The "Brain" of the framework. Executes Datalog policies against runtime data.
*   `PolicyEngine` (`solver.go`): High-level orchestrator. Runs Assess (Pre-Check), Reflect (Post-Check), and Steering.
*   `MangleRuntime` (`runtime.go`): Low-level wrapper around the Google Mangle library. Manages facts and rules.
*   `Reflection` (`reflection.go`): Converts Go structs into Datalog facts (`value(ID, Field, Val)`).
*   `Flattener` (`flattener.go`): Converts generic Maps/JSON into Datalog facts (`json_link`, `json_str`).

### Internal Supervisor (`internal/supervisor/`)
> **Responsibilities:** The "Guard" of the framework. Wraps actions to enforce policy and telemetry.
*   `SupervisedAction` (`supervisor.go`): Implements the decorator pattern.
    *   **Flow**: Trace -> Assess -> Execute -> Reflect -> Steering.

### Adapters (`adapters/`)
> **Responsibilities:** Connects the core logic to the outside world.
*   `adapters/func`: Wraps native Go functions using reflection to create typed Actions.
*   `adapters/ai`: Adapts Firebase Genkit to `core.Action`.
*   `adapters/mcp`: Adapts MCP Servers to `core.Action`.
*   `adapters/resilience`: Adds Circuit Breaker logic to Actions.

## 3. CRITICAL PATH & DATA (The Flow)

### 1. Visualizations

#### Sequence Diagram: The Governance Loop
```mermaid
sequenceDiagram
    participant User
    participant SDK as SDK Client
    participant Loop as RunLoop
    participant Sup as Supervisor
    participant Eng as PolicyEngine
    participant Act as Inner Action

    User->>SDK: Action("MyFunc").Execute(Payload)
    SDK->>Loop: ExecuteByName("MyFunc")
    loop Semantic State Machine
        Loop->>Sup: Execute(Envelope)

        rect rgb(240, 240, 240)
            note right of Sup: Phase 1: Pre-Check
            Sup->>Eng: Assess(Envelope)
            Eng-->>Sup: OK / Blocked
        end

        alt Blocked
            Sup-->>Loop: Error (AlignmentError)
            Loop->>Loop: Handle Retry/Backoff
        else Allowed
            Sup->>Act: Execute(Envelope)
            Act-->>Sup: Result

            rect rgb(240, 240, 240)
                note right of Sup: Phase 2: Post-Check
                Sup->>Eng: Reflect(Result)
            end

            rect rgb(240, 240, 240)
                note right of Sup: Phase 3: Steering
                Sup->>Eng: EvaluateSteering(Result)
                Eng-->>Sup: Decision (Proceed/Retry/Route)
            end

            Sup-->>Loop: Result + Decision
        end

        Loop->>Loop: Process Decision
        note right of Loop: If Route -> Next Action<br/>If Retry -> Same Action
    end
    Loop-->>User: Final Result
```

#### Data Flow: The Fact Funnel
```mermaid
graph LR
    Input[JSON/Struct Input] -->|Reflection| Facts[Datalog Facts]
    Facts -->|Injected| Runtime[Mangle Runtime]
    Policy[Policy Rules (.dl)] -->|Loaded| Runtime
    Runtime -->|Solver| Outcome[Authorization Decision]
    Outcome -->|Control| Supervisor
```

### 2. Execution Narrative
1.  **Entry**: The user calls `client.Action("name").Execute(ctx, env)`.
2.  **Orchestration**: The `SDK` delegates to `RunLoop`. The loop initializes context, memory, and history.
3.  **Supervision**: The `Supervisor` intercepts the call. It asks the `Engine` to `Assess` the input.
4.  **Reasoning**: The `Engine` converts the input into Datalog facts (e.g., `value("req", "amount", 1000)`). It runs queries like `halt("req", Reason)`.
5.  **Execution**: If allowed, the `Supervisor` invokes the underlying `Adapter` (e.g., calls an LLM or runs a function).
6.  **Reflection**: The output is passed back to the `Engine` for `Reflect` (validation).
7.  **Steering**: The `Engine` evaluates `route(Target)` or `retry(Hint)` rules.
8.  **Loop**: The `RunLoop` acts on the steering decision. If `ROUTE`, it jumps to the next action. If `RETRY`, it re-executes with feedback.

## 4. SOURCE CODE DUMP (The Kernel)

### `internal/engine/resources/std.dl`
```datalog
% --- Manglekit Standard Library (v2.0) ---
% Auto-loaded on engine startup.

% ==========================================
% 1. DATA REFLECTION (DO NOT REMOVE)
% These predicates allow the engine to read JSON inputs and Graph data.
% ==========================================

% JSON Primitives (Flattened)
Decl json_str(Parent, Key, Val).
Decl json_num(Parent, Key, Val).
Decl json_bool(Parent, Key, Val).
Decl json_link(Parent, Key, Child).
Decl json_null(Parent, Key).

% Knowledge Graph Primitives (N-Quads/Triples)
Decl quad(S, P, O, G).
Decl triple(S, P, O).
triple(S, P, O) :- quad(S, P, O, _).

% ==========================================
% 2. SYSTEM CONTROL (Standard Vocabulary)
% ==========================================

% Arity 1: Global/Single-Context (JSON)
Decl deny(Reason).
Decl halt(Reason).
Decl route(NextStep).
Decl retry(Feedback).

% Arity 2: Entity-Specific (Graph/Batch)
% Allows pinning the decision to a specific node (Entity).
Decl deny(Entity, Reason).
Decl halt(Entity, Reason).
Decl route(Entity, NextStep).
Decl retry(Entity, Feedback).

% Config is always Key-Value (Arity 2) or Entity-Key-Value (Arity 3)
Decl config(Key, Value).
Decl config(Entity, Key, Value).

% Semantic Alias for backward compatibility
halt(Reason) :- deny(Reason).
halt(Entity, Reason) :- deny(Entity, Reason).

% ==========================================
% 3. CONTEXT INJECTION
% These predicates are injected by the runtime environment.
% ==========================================

Decl attempt(N).            % Current retry count (0, 1, 2...).
Decl meta(Key, Value).      % Envelope metadata (e.g., user_id, session_id).
Decl label(Tag).            % Security taint labels (e.g., "pii", "unsafe").

% Telemetry Predicate (Arity 2: Entity, RuleID)
Decl violation_rule(Entity, RuleID).
```

### `internal/engine/reflection.go`
```go
package engine

import (
	"fmt"
	"reflect"
	"strings"
)

// ToFacts converts a Go data structure into Mangle Datalog facts.
// It is the entry point for turning runtime objects into logic predicates.
func ToFacts(id string, input any) ([]string, error) {
	if input == nil {
		return nil, nil
	}
	var facts []string
	v := reflect.ValueOf(input)

	// Track visited pointers to prevent infinite recursion (Cycles)
	visited := make(map[uintptr]bool)

	if err := toFactsRecursive(id, "", v, &facts, visited); err != nil {
		return nil, err
	}
	return facts, nil
}

// LabelsToFacts converts a slice of security label strings into Mangle Datalog facts.
func LabelsToFacts(entityID string, labels []string) ([]string, error) {
	var facts []string
	if len(labels) > 0 {
		facts = make([]string, 0, len(labels))
	}

	for _, l := range labels {
		var sb strings.Builder
		sb.WriteString("label(\"")
		sb.WriteString(escapeString(l))
		sb.WriteString("\")")
		facts = append(facts, sb.String())
	}
	return facts, nil
}

// escapeString escapes special characters to ensure the resulting string
// is a valid Mangle string constant.
func escapeString(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch b {
		case '\\', '"':
			sb.WriteByte('\\')
			sb.WriteByte(b)
		case '\n':
			sb.WriteString("\\n")
		case '\r':
			sb.WriteString("\\r")
		case '\t':
			sb.WriteString("\\t")
		default:
			if b < 32 {
				sb.WriteByte(' ')
			} else {
				sb.WriteByte(b)
			}
		}
	}
	return sb.String()
}

func toFactsRecursive(id, path string, v reflect.Value, facts *[]string, visited map[uintptr]bool, args ...string) error {
	if !v.IsValid() {
		return nil
	}

	// 1. Dereference Interfaces
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	// 2. Cycle Detection (Ptr, Map, Slice) & Dereference Ptr
	k := v.Kind()
	if k == reflect.Ptr || k == reflect.Map || k == reflect.Slice {
		if v.IsNil() {
			return nil
		}
		ptr := v.Pointer()
		if visited[ptr] {
			return nil // Cycle detected
		}
		visited[ptr] = true
		defer delete(visited, ptr) // Stack-based tracking
	}

	if k == reflect.Ptr {
		v = v.Elem()
	}

	// 3. Switch on Kind
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			structField := t.Field(i)

			if !structField.IsExported() {
				continue
			}

			tag := structField.Tag.Get("mangle")
			if tag == "-" {
				continue
			}

			jsonTag := structField.Tag.Get("json")
			if jsonTag == "-" || strings.HasPrefix(jsonTag, "-,") {
				continue
			}

			fieldName := tag
			if fieldName == "" {
				if jsonTag != "" {
					parts := strings.Split(jsonTag, ",")
					fieldName = parts[0]
				}
			}
			if fieldName == "" {
				fieldName = strings.ToLower(structField.Name)
			}

			// Handle Embedded (Anonymous) Fields
			newPath := path
			isAnonymousUntagged := structField.Anonymous && tag == "" && structField.Tag.Get("json") == ""

			if !isAnonymousUntagged {
				if newPath != "" {
					newPath = newPath + "_" + fieldName
				} else {
					newPath = fieldName
				}
			}

			if err := toFactsRecursive(id, newPath, field, facts, visited, args...); err != nil {
				return err
			}
		}

	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			keyStr := fmt.Sprintf("%v", key.Interface())
			newArgs := make([]string, len(args)+1)
			copy(newArgs, args)
			newArgs[len(args)] = keyStr

			if err := toFactsRecursive(id, path, val, facts, visited, newArgs...); err != nil {
				return err
			}
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			idxStr := fmt.Sprintf("%d", i)
			newArgs := make([]string, len(args)+1)
			copy(newArgs, args)
			newArgs[len(args)] = idxStr

			if err := toFactsRecursive(id, path, v.Index(i), facts, visited, newArgs...); err != nil {
				return err
			}
		}

	default:
		generatePrimitiveFact(id, path, v, facts, args...)
	}
	return nil
}

func generatePrimitiveFact(id, path string, v reflect.Value, facts *[]string, args ...string) {
	var strVal string
	var isNumeric bool

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		strVal = fmt.Sprintf("%d", v.Int())
		isNumeric = true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		strVal = fmt.Sprintf("%d", v.Uint())
		isNumeric = true
	case reflect.Float32, reflect.Float64:
		strVal = fmt.Sprintf("%g", v.Float())
		isNumeric = true
	case reflect.Bool:
		strVal = fmt.Sprintf("%v", v.Bool())
	default:
		strVal = fmt.Sprintf("%v", v.Interface())
	}

	predicate := path
	if predicate == "" {
		predicate = "value"
	}

	safeID := escapeString(id)

	var sb strings.Builder
	sb.WriteString(predicate)
	sb.WriteByte('(')
	sb.WriteByte('"')
	sb.WriteString(safeID)
	sb.WriteByte('"')

	for _, arg := range args {
		sb.WriteString(", \"")
		sb.WriteString(escapeString(arg))
		sb.WriteByte('"')
	}

	sb.WriteString(", ")
	if isNumeric {
		sb.WriteString(strVal)
	} else {
		sb.WriteByte('"')
		sb.WriteString(escapeString(strVal))
		sb.WriteByte('"')
	}
	sb.WriteByte(')')

	*facts = append(*facts, sb.String())
}
```

### `internal/engine/flattener.go`
```go
package engine

import (
	"fmt"
	"reflect"
	"strconv"
)

// Flatten converts ANY dynamic structure (maps, slices of any type) into graph facts.
func Flatten(rootID string, input any) ([]string, error) {
	var facts []string
	if input == nil {
		return facts, nil
	}

	counter := 0
	visited := make(map[uintptr]bool)

	if err := flattenRecursive(rootID, reflect.ValueOf(input), &facts, &counter, visited); err != nil {
		return nil, err
	}
	return facts, nil
}

func flattenRecursive(nodeID string, v reflect.Value, facts *[]string, counter *int, visited map[uintptr]bool) error {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		if v.Kind() == reflect.Ptr {
			ptr := v.Pointer()
			if visited[ptr] {
				return nil
			}
			visited[ptr] = true
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()
			keyStr := fmt.Sprintf("%v", key.Interface())
			safeKey := escapeString(keyStr)

			effVal := val
			for effVal.Kind() == reflect.Interface || effVal.Kind() == reflect.Ptr {
				if effVal.IsNil() {
					break
				}
				effVal = effVal.Elem()
			}

			if isComplexKind(effVal.Kind()) {
				*counter++
				childID := fmt.Sprintf("node_%d", *counter)
				fact := fmt.Sprintf("json_link(\"%s\", \"%s\", \"%s\")", escapeString(nodeID), safeKey, childID)
				*facts = append(*facts, fact)

				if err := flattenRecursive(childID, val, facts, counter, visited); err != nil {
					return err
				}
			} else {
				addPrimitiveReflect(nodeID, safeKey, effVal, facts)
			}
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			val := v.Index(i)
			keyStr := strconv.Itoa(i)

			effVal := val
			for effVal.Kind() == reflect.Interface || effVal.Kind() == reflect.Ptr {
				if effVal.IsNil() {
					break
				}
				effVal = effVal.Elem()
			}

			if isComplexKind(effVal.Kind()) {
				*counter++
				childID := fmt.Sprintf("node_%d", *counter)
				fact := fmt.Sprintf("json_link(\"%s\", \"%s\", \"%s\")", escapeString(nodeID), keyStr, childID)
				*facts = append(*facts, fact)

				if err := flattenRecursive(childID, val, facts, counter, visited); err != nil {
					return err
				}
			} else {
				addPrimitiveReflect(nodeID, keyStr, effVal, facts)
			}
		}
	}
	return nil
}

func isComplexKind(k reflect.Kind) bool {
	return k == reflect.Map || k == reflect.Slice || k == reflect.Array
}

func addPrimitiveReflect(nodeID, key string, v reflect.Value, facts *[]string) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	nodeID = escapeString(nodeID)

	switch v.Kind() {
	case reflect.String:
		fact := fmt.Sprintf("json_str(\"%s\", \"%s\", \"%s\")", nodeID, key, escapeString(v.String()))
		*facts = append(*facts, fact)

	case reflect.Bool:
		sVal := "false"
		if v.Bool() {
			sVal = "true"
		}
		fact := fmt.Sprintf("json_bool(\"%s\", \"%s\", \"%s\")", nodeID, key, sVal)
		*facts = append(*facts, fact)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fact := fmt.Sprintf("json_num(\"%s\", \"%s\", %d)", nodeID, key, v.Int())
		*facts = append(*facts, fact)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fact := fmt.Sprintf("json_num(\"%s\", \"%s\", %d)", nodeID, key, v.Uint())
		*facts = append(*facts, fact)

	case reflect.Float32, reflect.Float64:
		fact := fmt.Sprintf("json_num(\"%s\", \"%s\", %g)", nodeID, key, v.Float())
		*facts = append(*facts, fact)

	default:
		sVal := fmt.Sprintf("%v", v.Interface())
		fact := fmt.Sprintf("json_str(\"%s\", \"%s\", \"%s\")", nodeID, key, escapeString(sVal))
		*facts = append(*facts, fact)
	}
}
```

### `internal/supervisor/supervisor.go`
```go
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/telemetry"

	"go.opentelemetry.io/otel"
)

// SupervisedAction is a decorator that wraps any `core.Action` to enforce governance blueprints.
type SupervisedAction struct {
	inner       core.Action
	engine      core.Evaluator
	tracer      core.Tracer
	failureMode string
}

func NewSupervisedAction(action core.Action, eng core.Evaluator, failureMode string) *SupervisedAction {
	return &SupervisedAction{
		inner:       action,
		engine:      eng,
		tracer:      &core.NopTracer{},
		failureMode: failureMode,
	}
}

func NewSupervisedActionWithTracer(action core.Action, eng core.Evaluator, tracer core.Tracer, failureMode string) *SupervisedAction {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	return &SupervisedAction{
		inner:       action,
		engine:      eng,
		tracer:      tracer,
		failureMode: failureMode,
	}
}

// Execute runs the supervised action, orchestrating the full governance lifecycle.
func (g *SupervisedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	tracer := g.tracer
	if tracer == nil {
		tracer = telemetry.NewOTelTracer(otel.Tracer("manglekit"))
	}
	meta := g.inner.Metadata()

	ctx, span := tracer.Start(ctx, fmt.Sprintf("Action.%s", meta.Name))
	defer span.End()

	span.SetAttributes(map[string]any{
		"mangle.action_name": meta.Name,
		"mangle.action_type": string(meta.Type),
		"mangle.input_id":    input.ID.String(),
	})

	result, err := g.executeInternal(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		if g.isAlignmentIssue(err) {
			span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeHalt})
			var alignErr *core.AlignmentError
			if errors.As(err, &alignErr) {
				attrs := map[string]any{core.KeyFeedback: alignErr.Message}
				if alignErr.RuleID != "" {
					attrs[core.AttrRuleID] = alignErr.RuleID
				}
				span.SetAttributes(attrs)
			} else {
				span.SetAttributes(map[string]any{core.KeyFeedback: err.Error()})
			}
		} else {
			span.SetAttributes(map[string]any{core.AttrOutcome: "ERROR"})
		}
		return core.Envelope{}, err
	}

	decision := result.Metadata[core.KeyDecision]
	switch decision {
	case core.DecisionRetry:
		span.SetAttributes(map[string]any{core.AttrOutcome: "retry"})
		if hint, ok := result.Metadata[core.KeyFeedback]; ok {
			if s, ok := hint.(string); ok {
				span.SetAttributes(map[string]any{core.KeyFeedback: s})
			}
		}
	case core.DecisionRoute:
		span.SetAttributes(map[string]any{core.AttrOutcome: "route"})
		if target, ok := result.Metadata[core.KeyNextStep]; ok {
			if s, ok := target.(string); ok {
				span.SetAttributes(map[string]any{core.AttrActionName: s})
			}
		}
	default:
		span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})
	}

	if attemptVal, ok := input.Metadata["retry_count"]; ok {
		if s, ok := attemptVal.(string); ok {
			if n, err := strconv.Atoi(s); err == nil {
				span.SetAttributes(map[string]any{core.AttrAttempt: n})
			}
		} else if n, ok := attemptVal.(int); ok {
			span.SetAttributes(map[string]any{core.AttrAttempt: n})
		}
	}

	span.SetAttributes(map[string]any{
		"mangle.output_id": result.ID.String(),
	})
	return result, nil
}

func (g *SupervisedAction) isAlignmentIssue(err error) bool {
	return core.IsAlignmentError(err)
}

func (g *SupervisedAction) Metadata() core.ActionMetadata {
	return g.inner.Metadata()
}

func (g *SupervisedAction) shouldBlock(err error) bool {
	if err == nil {
		return false
	}
	if core.IsAlignmentError(err) {
		return true
	}
	if core.IsInputError(err) {
		return true
	}
	if g.failureMode == "open" {
		return false
	}
	return true
}

func (g *SupervisedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	ctx = core.ContextWithLogger(ctx, g.engine.Logger())
	logger := core.LoggerFromContext(ctx)
	meta := g.inner.Metadata()

	logger.Info("Action started", "action", meta.Name, "input_id", input.ID.String())

	// Phase 1: Pre-Check (Assessment)
	if err := g.performAssessment(ctx, logger, meta, input); err != nil {
		return core.Envelope{}, err
	}

	// Phase 2: Dynamic Configuration Injection
	g.injectDynamicConfig(ctx, logger, &input)

	// Phase 3: Execute Inner Action
	result, err := g.executeAction(ctx, logger, meta, input)
	if err != nil {
		return core.Envelope{}, err
	}

	// Phase 4: Post-Check (Reflection)
	validatedResult, err := g.performReflection(ctx, logger, meta, result)
	if err != nil {
		return core.Envelope{}, err
	}

	// Phase 5: Steering
	validatedResult = g.applySteering(ctx, logger, meta, validatedResult)

	logger.Info("Action completed", "action", meta.Name, "result", "success")
	return validatedResult, nil
}

func (g *SupervisedAction) performAssessment(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) error {
	if err := g.engine.Assess(ctx, meta, input); err != nil {
		if g.shouldBlock(err) {
			msg := "assessment failed"
			if core.IsInputError(err) {
				msg = "assessment blocked due to invalid input"
			}
			logger.Warn(msg, core.AttrActionName, meta.Name, "error", err.Error())
			return fmt.Errorf("%s: %w", msg, err)
		}
		logger.Warn("engine assessment failed but Fail-Open active. Proceeding.", "error", err)
	}
	return nil
}

func (g *SupervisedAction) injectDynamicConfig(ctx context.Context, logger core.Logger, input *core.Envelope) {
	config, err := g.engine.GetActionConfig(ctx, *input)
	if err != nil {
		logger.Warn("failed to retrieve action config", "error", err)
		return
	}

	if len(config) == 0 {
		return
	}

	if input.Metadata == nil {
		input.Metadata = make(map[string]any)
	}
	for k, v := range config {
		input.Metadata[core.PrefixPromptConfig+k] = v
	}
}

func (g *SupervisedAction) executeAction(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) (core.Envelope, error) {
	childCtx := core.WithParentID(ctx, input.ID.String())
	result, err := g.inner.Execute(childCtx, input)
	if err != nil {
		logger.Error("action execution failed", core.AttrActionName, meta.Name, "error", err.Error())
		return core.Envelope{}, fmt.Errorf("action execution failed: %w", err)
	}

	if len(input.SecurityLabels) > 0 {
		result.MergeLabels(input.SecurityLabels)
	}

	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata["derived_from"] = input.ID.String()

	return result, nil
}

func (g *SupervisedAction) performReflection(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) (core.Envelope, error) {
	validatedResult, err := g.engine.Reflect(ctx, meta, result)
	if err != nil {
		if g.shouldBlock(err) {
			msg := "reflection failed"
			if core.IsInputError(err) {
				msg = "reflection blocked due to invalid input"
			}
			logger.Warn(msg, "action", meta.Name, "error", err.Error())
			return core.Envelope{}, fmt.Errorf("%s: %w", msg, err)
		}
		logger.Warn("engine reflection failed but Fail-Open active. Proceeding.", "error", err)
		validatedResult = result
	}
	return validatedResult, nil
}

func (g *SupervisedAction) applySteering(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) core.Envelope {
	decision, steeringMeta, err := g.engine.EvaluateSteering(ctx, result)
	if err != nil {
		logger.Warn("steering evaluation failed", "action", meta.Name, "error", err.Error())
		decision = ""
		steeringMeta = nil
	}

	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata[core.KeyDecision] = decision
	for k, v := range steeringMeta {
		result.Metadata[k] = v
	}

	return result
}
```

### `internal/engine/solver.go`
```go
package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine/resources"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/parse"
)

var ErrSolutionFound = errors.New("solution found")

type PolicyEngine struct {
	tracer  core.Tracer
	logger  core.Logger
	runtime *MangleRuntime
}

func New() (*PolicyEngine, error) {
	pe := &PolicyEngine{
		tracer:  &core.NopTracer{},
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}
	if err := pe.runtime.AddPolicy(resources.StdLib()); err != nil {
		return nil, fmt.Errorf("manglekit: failed to load std.dl: %w", err)
	}
	return pe, nil
}

func NewWithObservability(tracer core.Tracer, logger core.Logger) (*PolicyEngine, error) {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	if logger == nil {
		logger = core.NopLogger{}
	}
	pe := &PolicyEngine{
		tracer:  tracer,
		logger:  logger,
		runtime: NewMangleRuntime(),
	}
	if err := pe.runtime.AddPolicy(resources.GetPlannerRules()); err != nil {
		if logger != nil {
			logger.Error("failed to load planner core schema", "error", err)
		}
	}
	if err := pe.runtime.AddPolicy(resources.StdLib()); err != nil {
		if logger != nil {
			logger.Error("failed to load standard library", "error", err)
		}
		return nil, fmt.Errorf("manglekit: failed to load std.dl: %w", err)
	}
	return pe, nil
}

func (e *PolicyEngine) Logger() core.Logger {
	if e.logger == nil {
		return core.NopLogger{}
	}
	return e.logger
}

func (e *PolicyEngine) LoadFacts(facts []string) error {
	return e.runtime.LoadFacts(facts)
}

func (e *PolicyEngine) RegisterAction(meta core.ActionMetadata) error {
	var facts []string
	safeName := escapeString(meta.Name)

	facts = append(facts, fmt.Sprintf("action(\"%s\")", safeName))

	if meta.InputType != "" {
		facts = append(facts, fmt.Sprintf("has_input(\"%s\", \"%s\")", safeName, escapeString(meta.InputType)))
	}

	if meta.OutputType != "" {
		facts = append(facts, fmt.Sprintf("has_output(\"%s\", \"%s\")", safeName, escapeString(meta.OutputType)))
	}

	return e.LoadFacts(facts)
}

func (e *PolicyEngine) LoadPolicy(ctx context.Context, policy string) error {
	if policy == "" {
		return nil
	}
	if err := e.runtime.AddPolicy(policy); err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}
	if e.logger != nil {
		e.logger.Debug("policy loaded from string", "length", len(policy))
	}
	return nil
}

func (e *PolicyEngine) AssessPlan(ctx context.Context, input core.Envelope) (core.Decision, error) {
	err := e.Assess(ctx, core.ActionMetadata{}, input)
	if err != nil {
		var alignErr *core.AlignmentError
		if errors.As(err, &alignErr) {
			return core.Decision{
				Outcome: core.DecisionHalt,
				Reasons: []string{alignErr.Message},
				Meta:    map[string]string{"rule_id": alignErr.RuleID},
			}, nil
		}
		return core.Decision{Outcome: core.DecisionHalt, Reasons: []string{err.Error()}}, err
	}
	return core.Decision{Outcome: core.DecisionProceed}, nil
}

func (e *PolicyEngine) GetActionConfig(ctx context.Context, input core.Envelope) (map[string]string, error) {
	config := make(map[string]string)
	if e.runtime == nil || e.runtime.programInfo == nil {
		return config, nil
	}

	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts for config", "error", err)
		}
		return config, nil
	}

	for _, factStr := range input.Facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			continue
		}
		facts = append(facts, atom)
	}

	for k, v := range input.Metadata {
		safeK := escapeString(k)
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}
		if k == "retry_count" {
			attemptFact := fmt.Sprintf("attempt(%s)", vStr)
			if atom, err := parse.Atom(attemptFact); err == nil {
				facts = append(facts, atom)
			}
		}
	}

	err = e.runtime.QueryWithSolutions(facts, "config(Key, Value)", func(solution map[string]any) error {
		key, kOk := solution["Key"].(string)
		val, vOk := solution["Value"].(string)
		if kOk && vOk {
			config[key] = val
		}
		return nil
	})
	return config, nil
}

func (e *PolicyEngine) Assess(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.tracer == nil {
		return e.assessInternal(ctx, actionMeta, input)
	}
	ctx, span := e.tracer.Start(ctx, core.SpanPreCheck)
	defer span.End()
	if len(input.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: input.SecurityLabels})
	}
	err := e.assessInternal(ctx, actionMeta, input)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})
	}
	return err
}

func (e *PolicyEngine) assessInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	var extraFacts []ast.Atom
	if actionMeta.Name != "" {
		safeName := core.EntityInput
		safeOp := actionMeta.Name
		opFactStr := fmt.Sprintf("action_operation(\"%s\", \"%s\")", escapeString(safeName), escapeString(safeOp))
		opAtom, err := parse.Atom(opFactStr)
		if err == nil {
			extraFacts = append(extraFacts, opAtom)
		}
	}
	return e.evaluateGate(ctx, actionMeta.Name, core.EntityInput, input, extraFacts...)
}

func (e *PolicyEngine) Reflect(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.tracer == nil {
		return e.reflectInternal(ctx, actionMeta, output)
	}
	ctx, span := e.tracer.Start(ctx, core.SpanPostCheck)
	defer span.End()
	if len(output.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: output.SecurityLabels})
	}
	result, err := e.reflectInternal(ctx, actionMeta, output)
	if err != nil {
		span.RecordError(err)
		return core.Envelope{}, err
	}
	span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})
	return result, nil
}

func (e *PolicyEngine) reflectInternal(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	err := e.evaluateGate(ctx, actionMeta.Name, core.EntityOutput, output)
	if err != nil {
		return core.Envelope{}, err
	}
	return output, nil
}

func (e *PolicyEngine) evaluateGate(ctx context.Context, actionName string, entityID string, env core.Envelope, extraFacts ...ast.Atom) error {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return nil
	}

	facts, err := toMangleFacts(entityID, env.Payload, env.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert payload to facts", "error", err)
		}
		return &core.InputError{Err: fmt.Errorf("fact conversion error: %w", err)}
	}

	facts = append(facts, extraFacts...)

	labelFacts, err := LabelsToFacts(entityID, env.SecurityLabels)
	if err != nil {
		return &core.InputError{Err: fmt.Errorf("label conversion error: %w", err)}
	}
	for _, f := range labelFacts {
		atom, err := parse.Atom(f)
		if err == nil {
			facts = append(facts, atom)
		}
	}

	for _, f := range env.Facts {
		atom, err := parse.Atom(f)
		if err == nil {
			facts = append(facts, atom)
		}
	}

	for k, v := range env.Metadata {
		safeK := escapeString(k)
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}
		if k == "retry_count" {
			attemptFact := fmt.Sprintf("attempt(%s)", vStr)
			if atom, err := parse.Atom(attemptFact); err == nil {
				facts = append(facts, atom)
			}
		}
	}

	var violationMsg, ruleID string
	var blocked bool

	queryHalt := fmt.Sprintf("%s(\"%s\", Reason)", core.PredHalt, entityID)
	err = e.runtime.QueryWithSolutions(facts, queryHalt, func(solution map[string]any) error {
		if reason, ok := solution["Reason"].(string); ok {
			violationMsg = reason
			blocked = true
			return ErrSolutionFound
		}
		return nil
	})
	if errors.Is(err, ErrSolutionFound) {
		err = nil
	}

	if blocked {
		e.runtime.QueryWithSolutions(facts, "violation_rule(ID)", func(solution map[string]any) error {
			if id, ok := solution["ID"].(string); ok {
				ruleID = id
				return ErrSolutionFound
			}
			return nil
		})
		if e.logger != nil {
			e.logger.Debug("gate violation detected (halt)", "action", actionName, "msg", violationMsg, "rule_id", ruleID)
		}
		return &core.AlignmentError{Message: violationMsg, RuleID: ruleID}
	}

	// Legacy Deny
	queryDeny := fmt.Sprintf("%s(\"%s\")", core.PredHalt, entityID)
	denied, err := e.runtime.ExecuteQuery(facts, queryDeny)
	if denied {
		e.runtime.QueryWithSolutions(facts, fmt.Sprintf("%s(Msg)", core.PredViolation), func(solution map[string]any) error {
			if msg, ok := solution["Msg"].(string); ok {
				violationMsg = msg
				return ErrSolutionFound
			}
			return nil
		})
		if e.logger != nil {
			e.logger.Debug("gate violation detected (deny)", "action", actionName, "msg", violationMsg)
		}
		return &core.AlignmentError{Message: violationMsg}
	}

	return nil
}

func (e *PolicyEngine) CheckRequirement(ctx context.Context, input core.Envelope, reqName string) (bool, error) {
	if e.runtime == nil {
		return false, nil
	}
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		return false, fmt.Errorf("fact conversion failed: %w", err)
	}
	query := fmt.Sprintf(`requires("%s", "%s")`, core.EntityInput, reqName)
	return e.ExecuteQuery(ctx, facts, query)
}

func (e *PolicyEngine) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	decision := core.DecisionProceed
	metadata := make(map[string]string)
	if e.runtime == nil || e.runtime.programInfo == nil {
		return decision, metadata, nil
	}

	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		return decision, metadata, fmt.Errorf("failed to convert input to facts: %w", err)
	}

	for _, factStr := range input.Facts {
		atom, err := parse.Atom(factStr)
		if err == nil {
			facts = append(facts, atom)
		}
	}

	for k, v := range input.Metadata {
		safeK := escapeString(k)
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}
		if k == "retry_count" {
			attemptFact := fmt.Sprintf("attempt(%s)", vStr)
			if atom, err := parse.Atom(attemptFact); err == nil {
				facts = append(facts, atom)
			}
		}
	}

	err = e.runtime.QueryWithSolutions(facts, fmt.Sprintf("%s(Hint)", core.PredRetry), func(solution map[string]any) error {
		if hint, ok := solution["Hint"].(string); ok {
			decision = core.DecisionRetry
			metadata[core.KeyFeedback] = hint
			return ErrSolutionFound
		}
		return nil
	})
	if errors.Is(err, ErrSolutionFound) {
		return decision, metadata, nil
	}

	err = e.runtime.QueryWithSolutions(facts, fmt.Sprintf("%s(Target)", core.PredRoute), func(solution map[string]any) error {
		if target, ok := solution["Target"].(string); ok {
			decision = core.DecisionRoute
			metadata[core.KeyNextStep] = target
			return ErrSolutionFound
		}
		return nil
	})

	return decision, metadata, nil
}

func (e *PolicyEngine) ExecuteQuery(ctx context.Context, facts []ast.Atom, queryStr string) (bool, error) {
	if e.tracer == nil {
		return e.runtime.ExecuteQuery(facts, queryStr)
	}
	ctx, span := e.tracer.Start(ctx, "Datalog.ExecuteQuery")
	defer span.End()
	span.SetAttributes(map[string]any{"datalog.query": queryStr})
	res, err := e.runtime.ExecuteQuery(facts, queryStr)
	if err != nil {
		span.RecordError(err)
		return false, err
	}
	span.SetAttributes(map[string]any{"datalog.result": res})
	return res, nil
}

func (e *PolicyEngine) Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error) {
	var results []map[string]string
	if e.runtime == nil {
		return nil, fmt.Errorf("runtime not initialized")
	}

	var atomFacts []ast.Atom
	for _, f := range facts {
		atom, err := parse.Atom(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", f, err)
		}
		atomFacts = append(atomFacts, atom)
	}

	err := e.runtime.QueryWithSolutions(atomFacts, queryStr, func(solution map[string]any) error {
		strMap := make(map[string]string)
		for k, v := range solution {
			if s, ok := v.(string); ok {
				strMap[k] = s
			} else {
				strMap[k] = fmt.Sprintf("%v", v)
			}
		}
		results = append(results, strMap)
		return nil
	})
	return results, err
}

func toMangleFacts(entityID string, input any, contentType core.ContentType) ([]ast.Atom, error) {
	if input == nil {
		return nil, nil
	}
	var atoms []ast.Atom
	var facts []string
	var err error

	if contentType == core.TypeJSON {
		facts, err = Flatten(entityID, input)
	} else {
		facts, err = ToFacts(entityID, input)
	}

	if err != nil {
		return nil, err
	}

	for _, factStr := range facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", factStr, err)
		}
		atoms = append(atoms, atom)
	}
	return atoms, nil
}
```

### `internal/statehelper/statehelper.go`
```go
package statehelper

import (
	"context"
	"encoding/json"

	"github.com/duynguyendang/manglekit/core"
)

type ConversationManager struct{}

func NewConversationManager() *ConversationManager {
	return &ConversationManager{}
}

func (cm *ConversationManager) LoadHistory(ctx context.Context, sessionID string, sp core.StateProvider, logger core.Logger) *core.ConversationHistory {
	history := &core.ConversationHistory{}
	if sp != nil && sessionID != "" {
		rawState, err := sp.Get(ctx, sessionID)
		if err != nil {
			logger.Warn("Failed to retrieve state", "sessionID", sessionID, "error", err)
		}
		if rawState != nil {
			if stateBytes, ok := rawState.([]byte); ok {
				if err := json.Unmarshal(stateBytes, history); err != nil {
					logger.Warn("Failed to unmarshal state", "sessionID", sessionID, "error", err)
				}
			}
		}
	}
	return history
}

func (cm *ConversationManager) UpdateAndSaveHistory(ctx context.Context, sessionID string, sp core.StateProvider, logger core.Logger, history *core.ConversationHistory, q core.Query, a core.Answer) {
	if sp != nil && sessionID != "" && history != nil {
		history.Messages = append(history.Messages, core.Message{Role: "user", Content: q.Text})
		history.Messages = append(history.Messages, core.Message{Role: "model", Content: a.Text})

		updatedStateBytes, err := json.Marshal(history)
		if err == nil {
			if err := sp.Set(ctx, sessionID, updatedStateBytes); err != nil {
				logger.Warn("Failed to save state", "sessionID", sessionID, "error", err)
			}
		}
	}
}
```

### `sdk/client.go`
```go
package sdk

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/trace"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/supervisor"
)

const (
	TracerName = "github.com/duynguyendang/manglekit/sdk"
	FailModeOpen   = "open"
	FailModeClosed = "closed"
)

type Client struct {
	engine core.Evaluator
	tracer core.Tracer
	otelTracer trace.Tracer
	logger core.Logger
	agentMemory core.AgentMemory
	registry map[string]core.Action
	failureMode string
	blueprintPath string
	shutdownFunc func(context.Context) error
	llm core.TextGenerator
}

func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	c := &Client{
		logger: logger.NewDefault(),
		agentMemory: NewHybridMemory(core.NopStore{}, core.NopVectorStore{}, core.NopEmbedder{}),
		registry:    make(map[string]core.Action),
		failureMode: FailModeClosed,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if err := ensureDependencies(c); err != nil {
		return nil, err
	}
	return c, nil
}

func NewClientFromFile(ctx context.Context, configPath string, opts ...ClientOption) (*Client, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return NewClientFromConfig(ctx, cfg, opts...)
}

func NewClientFromConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error) {
	newOpts := append([]ClientOption{WithConfig(cfg)}, opts...)
	return NewClient(ctx, newOpts...)
}

func (c *Client) Supervise(action core.Action) core.Action {
	if c.tracer != nil {
		return supervisor.NewSupervisedActionWithTracer(action, c.engine, c.tracer, c.failureMode)
	}
	return supervisor.NewSupervisedAction(action, c.engine, c.failureMode)
}

func (c *Client) RegisterAction(name string, action core.Action) {
	c.registry[name] = action
	if c.engine != nil {
		if err := c.engine.RegisterAction(action.Metadata()); err != nil {
			c.logger.Warn("failed to register action metadata", "action", name, "error", err)
		}
	}
}

func (c *Client) Shutdown(ctx context.Context) error {
	if c.shutdownFunc != nil {
		return c.shutdownFunc(ctx)
	}
	return nil
}
```

### `sdk/loop.go`
```go
package sdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/core"
	engine_memory "github.com/duynguyendang/manglekit/internal/engine/memory"
)

const (
	DefaultMaxSteps   = 10
	DefaultMaxRetries = 3
	BackoffBase       = 100 * time.Millisecond
)

func (c *Client) ExecuteByName(ctx context.Context, actionName string, input any, opts ...ExecuteOption) (core.Envelope, error) {
	params := ExecutionParams{
		MemoryMode: core.MemoryModeNone,
	}
	for _, opt := range opts {
		opt(&params)
	}
	return c.runLoopInternal(ctx, actionName, input, params)
}

func (c *Client) runLoopInternal(ctx context.Context, startAction string, payload any, params ExecutionParams) (core.Envelope, error) {
	ctx = core.ContextWithLogger(ctx, c.logger)

	switch params.MemoryMode {
	case core.MemoryModePersist:
		params.Store = c.agentMemory
		if params.SessionID != "" {
			var err error
			params.CurrentHistory, err = params.Store.Read(ctx, params.SessionID)
			if err != nil && c.logger != nil {
				c.logger.Warn("RunLoop failed to hydrate history", "error", err)
			}
		}
	case core.MemoryModeTransient:
		params.Store = &engine_memory.VolatileStore{}
	default:
		params.Store = &core.NopStore{}
	}

	currentAction := startAction
	currentPayload := payload

	for step := 0; step < DefaultMaxSteps; step++ {
		if err := ctx.Err(); err != nil {
			return core.Envelope{}, err
		}

		c.logger.Info("RunLoop step", "step", step, "action", currentAction)

		result, err := c.ExecuteSingleStep(ctx, currentAction, currentPayload, &params)
		if err != nil {
			return core.Envelope{}, err
		}

		decision := result.Metadata[core.KeyDecision]
		if decision == core.DecisionRoute {
			next, ok := result.Metadata[core.KeyNextStep].(string)
			if !ok || next == "" {
				return core.Envelope{}, fmt.Errorf("route decision missing next_step")
			}
			currentAction = next
			currentPayload = result.Payload
			continue
		}

		if decision == core.DecisionRetry {
			continue
		}

		return result, nil
	}
	return core.Envelope{}, fmt.Errorf("max steps exceeded")
}

func (c *Client) ExecuteSingleStep(ctx context.Context, actionName string, payload any, params *ExecutionParams) (core.Envelope, error) {
	action, ok := c.registry[actionName]
	if !ok {
		return core.Envelope{}, fmt.Errorf("action not found: %s", actionName)
	}

	env := core.NewEnvelope(payload)
	env.ContentType = action.Metadata().InputContentType
	c.injectContext(ctx, &env, payload, params)

	result, err := action.Execute(ctx, env)
	if err != nil {
		return c.handleExecutionError(ctx, err, payload, params)
	}
	params.LastFeedback = ""

	c.updateHistory(ctx, payload, result, params)

	return c.handleDecision(ctx, actionName, result, payload, params)
}

func (c *Client) injectContext(ctx context.Context, env *core.Envelope, payload any, params *ExecutionParams) {
	if len(params.FeedbackHistory) > 0 {
		env.Metadata[core.KeyPrevFeedback] = strings.Join(params.FeedbackHistory, "; ")
	}
	if params.LastFeedback != "" {
		env.SetFeedback(params.LastFeedback)
		env.Metadata["mangle_feedback"] = params.LastFeedback
	}
	if len(params.CurrentHistory) > 0 && params.MemoryMode != core.MemoryModeNone {
		env.SetHistory(params.CurrentHistory)
	}
	c.recallContext(ctx, payload, env)
	for k, v := range params.Metadata {
		env.Metadata[k] = v
	}
	for k, v := range core.ContextFacts(ctx) {
		env.Metadata[k] = v
	}
}

func (c *Client) handleExecutionError(ctx context.Context, err error, payload any, params *ExecutionParams) (core.Envelope, error) {
	var alignErr *core.AlignmentError
	if !errors.As(err, &alignErr) {
		return core.Envelope{}, err
	}

	if params.RetryCount >= DefaultMaxRetries {
		return core.Envelope{}, fmt.Errorf("max retries exceeded: %w", err)
	}

	params.RetryCount++
	params.LastFeedback = alignErr.Message
	c.logger.Warn("RunLoop: Blueprint Alignment Issue", "feedback", params.LastFeedback, "attempt", params.RetryCount)

	if err := c.backoff(ctx, params.RetryCount); err != nil {
		return core.Envelope{}, err
	}

	res := core.NewEnvelope(payload)
	res.Metadata[core.KeyDecision] = core.DecisionRetry
	return res, nil
}

func (c *Client) updateHistory(ctx context.Context, payload any, result core.Envelope, params *ExecutionParams) {
	if params.MemoryMode == core.MemoryModeNone {
		return
	}
	newExchange := []core.Message{
		{Role: "user", Content: safelyStringify(payload)},
		{Role: "assistant", Content: safelyStringify(result.Payload)},
	}
	params.CurrentHistory = append(params.CurrentHistory, newExchange...)

	if params.Store != nil && params.SessionID != "" && params.MemoryMode == core.MemoryModePersist {
		if err := params.Store.Append(ctx, params.SessionID, params.CurrentHistory); err != nil && c.logger != nil {
			c.logger.Warn("RunLoop failed to persist history", "error", err)
		}
	}
}

func (c *Client) handleDecision(ctx context.Context, actionName string, result core.Envelope, payload any, params *ExecutionParams) (core.Envelope, error) {
	decision := result.Metadata[core.KeyDecision]

	if decision == "" || decision == core.DecisionProceed {
		c.asyncMemorize(payload, result.Payload)
	}

	switch decision {
	case core.DecisionRetry:
		return c.handleRetryDecision(ctx, actionName, result, params)
	case core.DecisionRoute:
		params.RetryCount = 0
		params.FeedbackHistory = nil
		return result, nil
	case core.DecisionProceed, "":
		return result, nil
	case core.DecisionHalt:
		return core.Envelope{}, c.buildHaltError(result)
	}
	return result, nil
}

func (c *Client) handleRetryDecision(ctx context.Context, actionName string, result core.Envelope, params *ExecutionParams) (core.Envelope, error) {
	if params.RetryCount >= DefaultMaxRetries {
		return core.Envelope{}, fmt.Errorf("max retries exceeded for action %s", actionName)
	}

	params.RetryCount++
	hint := result.GetFeedback()
	params.LastFeedback = hint
	params.FeedbackHistory = append(params.FeedbackHistory, hint)

	if err := c.backoff(ctx, params.RetryCount); err != nil {
		return core.Envelope{}, err
	}
	return result, nil
}

func (c *Client) buildHaltError(result core.Envelope) error {
	reason := result.Metadata["reason"]
	if reason == "" {
		reason = result.Metadata["violation_msg"]
	}
	if reason == "" {
		reason = "blueprint violation"
	}
	return fmt.Errorf("action halted by blueprint: %s", reason)
}

func (c *Client) backoff(ctx context.Context, retryCount int) error {
	sleepDuration := time.Duration(retryCount) * BackoffBase
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(sleepDuration):
		return nil
	}
}

func safelyStringify(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func (c *Client) recallContext(ctx context.Context, payload any, env *core.Envelope) {
	if c.agentMemory == nil {
		return
	}
	if c.engine != nil {
		needed, err := c.engine.CheckRequirement(ctx, *env, "memory")
		if err != nil || !needed {
			return
		}
	}
	inputStr := safelyStringify(payload)
	contextData, err := c.agentMemory.Recall(ctx, inputStr)
	if err == nil && contextData != "" {
		env.SetMeta(core.KeyContext, contextData)
	}
}

func (c *Client) asyncMemorize(input any, output any) {
	if c.agentMemory == nil {
		return
	}
	go func(q, a string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.agentMemory.Memorize(ctx, q, a)
	}(safelyStringify(input), safelyStringify(output))
}
```

### `sdk/proxy.go`
```go
package sdk

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
)

func (c *Client) Action(name string) core.Action {
	return &actionProxy{
		client: c,
		name:   name,
	}
}

type actionProxy struct {
	client *Client
	name   string
}

func (p *actionProxy) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	return p.client.ExecuteByName(ctx, p.name, env.Payload, WithMetadataMap(env.Metadata))
}

func (p *actionProxy) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: p.name,
		Type: "proxy",
	}
}
```

### `core/types.go`
```go
package core

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

const (
	KeyDecision     = "manglekit.decision"
	KeyFeedback     = "manglekit.feedback"
	KeyPrevFeedback = "prev_feedback"
	KeyNextStep     = "manglekit.next_step"
	KeyRiskScore = "manglekit.risk_score"
	KeyLatencyMs = "manglekit.latency_ms"
	KeyTraceID   = "manglekit.trace_id"
	KeyModel     = "manglekit.model"
	KeyHistory   = "manglekit_history"
	KeyContext   = "manglekit.context"
	KeySummary   = "manglekit.summary"
	PrefixPromptConfig = "prompt."
)

const (
	DecisionProceed = "PROCEED"
	DecisionHalt    = "HALT"
	DecisionRetry   = "RETRY"
	DecisionRoute   = "ROUTE"
)

const (
	EntityInput   = "Req"
	EntityOutput  = "Output"
	PredHalt      = "halt"
	PredRetry     = "retry"
	PredRoute     = "route"
	PredViolation = "violation_msg"
)

const (
	SpanPreCheck  = "Datalog.Assess"
	SpanPostCheck = "Datalog.Reflect"
	SpanMemory    = "Mangle.Recall"
	AttrPolicyName   = "policy.name"
	AttrOutcome      = "outcome"
	AttrLabels       = "mangle.labels"
	AttrActionName   = "action.name"
	AttrRuleID       = "mangle.rule_id"
	AttrAttempt      = "mangle.attempt"
)

const (
	OutcomeProceed = "PROCEED"
	OutcomeHalt    = "HALT"
)

type ContentType string

const (
	TypeStruct ContentType = "STRUCT"
	TypeJSON ContentType = "JSON"
)

type Envelope struct {
	ID uuid.UUID `json:"id"`
	Payload any `json:"data"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Error error `json:"error,omitempty"`
	SecurityLabels []string `json:"security_labels,omitempty"`
	Facts []string `json:"facts,omitempty"`
	ContentType ContentType `json:"content_type,omitempty"`
}

func NewEnvelope(payload any) Envelope {
	return Envelope{
		ID:             uuid.New(),
		Payload:        payload,
		Metadata:       make(map[string]any),
		SecurityLabels: []string{},
		ContentType:    TypeStruct,
	}
}

func (e *Envelope) SetMeta(k string, v any) {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[k] = v
}

func (e *Envelope) GetMeta(k string) string {
	if e.Metadata == nil {
		return ""
	}
	v, ok := e.Metadata[k]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func (e *Envelope) SetFeedback(msg string) {
	e.SetMeta(KeyFeedback, msg)
}

func (e *Envelope) GetFeedback() string {
	return e.GetMeta(KeyFeedback)
}

func (e *Envelope) AddLabel(label string) {
	if !e.HasLabel(label) {
		e.SecurityLabels = append(e.SecurityLabels, label)
	}
}

func (e *Envelope) HasLabel(label string) bool {
	for _, l := range e.SecurityLabels {
		if l == label {
			return true
		}
	}
	return false
}

func (e *Envelope) MergeLabels(other []string) {
	for _, l := range other {
		e.AddLabel(l)
	}
}

func (e *Envelope) SetHistory(msgs []Message) {
	b, err := json.Marshal(msgs)
	if err == nil {
		e.SetMeta(KeyHistory, string(b))
	}
}

type Decision struct {
	Outcome string
	Target  string
	Reasons []string
	Meta    map[string]string
}

type ActionMetadata struct {
	Name string
	Type string
	InputContentType ContentType
	InputType string
	OutputType string
	IsDynamic bool
}

type Message struct {
	Role string `json:"role"`
	Content string `json:"content"`
}

type ConversationHistory struct {
	Messages []Message `json:"messages"`
}

type Query struct {
	Text string         `json:"text"`
	Meta map[string]any `json:"meta,omitempty"`
}

type Answer struct {
	Text string         `json:"text"`
	Meta map[string]any `json:"meta,omitempty"`
}
```

### `core/governance.go`
```go
package core

import "context"

type Evaluator interface {
	AssessPlan(ctx context.Context, input Envelope) (Decision, error)
	Assess(ctx context.Context, actionMeta ActionMetadata, input Envelope) error
	Reflect(ctx context.Context, actionMeta ActionMetadata, output Envelope) (Envelope, error)
	EvaluateSteering(ctx context.Context, input Envelope) (string, map[string]string, error)
	GetActionConfig(ctx context.Context, input Envelope) (map[string]string, error)
	CheckRequirement(ctx context.Context, input Envelope, reqName string) (bool, error)
	LoadPolicy(ctx context.Context, source string) error
	LoadFacts(facts []string) error
	RegisterAction(meta ActionMetadata) error
	Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error)
	Logger() Logger
}

type PreProcessor interface {
	Process(ctx context.Context, input Envelope) (map[string]any, error)
}

type RiskEngine interface {
	CalculateRisk(ctx context.Context, input Envelope) (float64, error)
}

type ResourceMonitor interface {
	CountTokens(ctx context.Context, text string) (int, error)
	CheckBudget(ctx context.Context, key string, cost int) (bool, error)
}
```

### `adapters/func/wrapper.go`
```go
package function

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
)

type ToolFunc[In any, Out any] func(context.Context, In) (Out, error)

type Wrapper[In any, Out any] struct {
	name        string
	fn          ToolFunc[In, Out]
	contentType core.ContentType
	inputType   string
	outputType  string
}

func New[In any, Out any](name string, fn ToolFunc[In, Out]) *Wrapper[In, Out] {
	inType := reflect.TypeOf((*In)(nil)).Elem()
	if inType.Kind() == reflect.Ptr {
		inType = inType.Elem()
	}
	inName := inType.Name()

	outType := reflect.TypeOf((*Out)(nil)).Elem()
	if outType.Kind() == reflect.Ptr {
		outType = outType.Elem()
	}
	outName := outType.Name()

	return &Wrapper[In, Out]{
		name:        name,
		fn:          fn,
		contentType: core.TypeStruct,
		inputType:   inName,
		outputType:  outName,
	}
}

func (w *Wrapper[In, Out]) SetContentType(ct core.ContentType) {
	w.contentType = ct
}

func (w *Wrapper[In, Out]) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	in, ok := input.Payload.(In)
	if !ok {
		return core.Envelope{}, fmt.Errorf("%w: invalid input type, expected %T but got %T", core.ErrSystemError, *new(In), input.Payload)
	}

	out, err := w.fn(ctx, in)
	if err != nil {
		return core.Envelope{}, err
	}

	return core.NewEnvelope(out), nil
}

func (w *Wrapper[In, Out]) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name:             w.name,
		Type:             "function",
		InputContentType: w.contentType,
		InputType:        w.inputType,
		OutputType:       w.outputType,
		IsDynamic:        false,
	}
}
```

## 6. CHANGELOG
*   [2025-12-17]: Kernel Resync. Refocused on Logic Core. Added sdk/proxy.go and refined Adapter summaries.
