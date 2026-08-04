# Datalog Engine Capabilities

> Moved from the README in v0.6. Reference docs:
> [predicates](../../../docs/context/datalog/predicates.md),
> [standard library](../../../docs/context/datalog/standard-library.md),
> [policy authoring](../../../docs/context/governance/datalog-policies.md).

The Manglekit Datalog engine (powered by [mangle-go](https://codeberg.org/TauCeti/mangle-go)) supports a rich set of predicates, functions, and operators for declarative logic.

### Comparisons (`:ge`, `:le`, `:gt`, `:lt`)

Compare values in rule bodies. Uses **integer** values (scale floats by 1000).

```prolog
% Quality gate: pass if score >= threshold
passes_gate(DocType) :-
    completeness_pct(DocType, Score),
    min_completeness_pct(DocType, Min),
    :ge(Score, Min).

% Overfitting detection: fail if effort > max
task_exceeds_max(Task) :-
    task_effort_pct(Task, E),
    max_effort_pct(Max),
    :gt(E, Max).
```

| Operator | Meaning | Example |
|----------|---------|---------|
| `:ge(A, B)` | A >= B | `:ge(870, 850)` → true |
| `:le(A, B)` | A <= B | `:le(100, 200)` → true |
| `:gt(A, B)` | A > B | `:gt(200, 150)` → true |
| `:lt(A, B)` | A < B | `:lt(700, 850)` → true |

> **Note**: mangle-go comparisons use `int64` only. For float comparisons, scale to integers (e.g., `0.85` → `850`).

### Negation (`!`)

Find cases where a condition is **not** met. All predicates must be defined in the program.

```prolog
% Find missing capabilities
has_capability("cloud", "deploy").
has_capability("cloud", "monitor").
needs_capability("cloud", "scale").
needs_capability("cloud", "deploy").

% "scale" is needed but not available
missing_capability(Project, Cap) :-
    needs_capability(Project, Cap),
    !has_capability(Project, Cap).
```

**Result**: `missing_capability("cloud", "scale")` — only `scale` is missing.

### Aggregation (`fn:count`, `fn:sum`, `fn:max`, `fn:min`)

Aggregate values using `|> do fn:group_by(...)` transforms.

```prolog
task_effort("M1.1", 3).
task_effort("M1.2", 5).
task_effort("M1.3", 2).

% Sum all efforts
total_effort(Total) :-
    task_effort(Task, E) |> do fn:group_by(Task), let Total = fn:sum(E).

% Find max effort
max_effort(MaxE) :-
    task_effort(Task, E) |> do fn:group_by(Task), let MaxE = fn:max(E).

% Count tasks
task_count(Count) :-
    task_effort(_, _) |> do fn:group_by(_), let Count = fn:count(_).
```

| Function | Description | Input | Output |
|----------|-------------|-------|--------|
| `fn:count(X)` | Count elements | list | int |
| `fn:sum(E)` | Sum values | list of int | int |
| `fn:max(E)` | Maximum value | list of int | int |
| `fn:min(E)` | Minimum value | list of int | int |
| `fn:avg(E)` | Average value | list of floats | float64 |

### Arithmetic (`fn:mult`, `fn:div`, `fn:plus`, `fn:minus`)

Arithmetic functions work within `|>` transforms.

```prolog
% PERT estimation: E = (O + 4*M + P) / 6
task_pert("M1.1", 2, 3, 5).

pert_estimate(Task, E) :-
    task_pert(Task, O, M, P),
    let E = fn:div(fn:plus(O, fn:plus(fn:mult(4, M), P)), 6).
```

| Function | Description | Example |
|----------|-------------|---------|
| `fn:plus(A, B)` | Integer addition | `fn:plus(3, 5)` → 8 |
| `fn:minus(A, B)` | Integer subtraction | `fn:minus(10, 3)` → 7 |
| `fn:mult(A, B)` | Integer multiplication | `fn:mult(4, 3)` → 12 |
| `fn:div(A, B)` | Integer division | `fn:div(19, 6)` → 3 |
| `fn:float:plus(A, B)` | Float addition | `fn:float:plus(1.5, 2.5)` → 4.0 |
| `fn:float:mult(A, B)` | Float multiplication | `fn:float:mult(10.0, 0.5)` → 5.0 |
| `fn:float:div(A, B)` | Float division | `fn:float:div(10.0, 3.0)` → 3.333... |

> **Note**: `let` bindings work inside `|>` transforms. Standalone `let X = fn:...` in rule bodies is not supported.

### String Operations

```prolog
% String contains
has_password(Text) :- fn:contains(Text, "password").

% String starts with
is_api_endpoint(URL) :- fn:starts_with(URL, "/api/").

% String concatenation
full_path(Base, Seg, Result) :- let Result = fn:string:concat(Base, Seg).
```

### Full Example: Quality Gate

```prolog
% Document metrics (integer-scaled: 0.87 → 870)
completeness_pct("BRD", 870).
consistency_pct("BRD", 920).
generic_pct("BRD", 100).

% Thresholds
min_completeness("BRD", 850).
min_consistency("BRD", 880).
max_generic("BRD", 200).

% Derived predicates
passes_completeness(D) :- completeness_pct(D, S), min_completeness(D, M), :ge(S, M).
passes_consistency(D) :- consistency_pct(D, S), min_consistency(D, M), :ge(S, M).
passes_generic(D) :- generic_pct(D, R), max_generic(D, M), :le(R, M).

passes_quality_gate(D) :-
    passes_completeness(D),
    passes_consistency(D),
    passes_generic(D).
```

---

