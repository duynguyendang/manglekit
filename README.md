[![Go](https://img.shields.io/badge/Go-1.25%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is the **Zero-Trust Policy Gate for AI Agent Actions**.

> *"LLMs propose, symbolic logic approves."*

Modern AI applications require **deterministic safety and auditability** (financial limits, compliance checks, permission boundaries), but LLMs are inherently **probabilistic** (hallucinatory, jailbreakable, non-deterministic).

Manglekit solves this by placing every AI agent action behind a **fail-closed, auditable Datalog gate** before the action is dispatched.

1. **Deterministic Verification**: Policies written in Datalog ([Google Mangle](https://github.com/google/mangle)) verify actions against relational facts, multi-tier priorities (T0–T3), and integer arithmetic (`:gt`, `:lt`, `:ge`, `:le`).
2. **Zero-Trust Supervision**: A fail-closed gate interceptor evaluates incoming action requests before handler execution and verifies outputs afterward.
3. **Audit Provenance**: Every denied or allowed action produces a mathematically grounded **derivation tree** proving *which rule fired and why*.
4. **Polyglot Sidecar**: Run in-process in Go or as a lightweight HTTP/JSON sidecar (`mkit serve`) for Python, TypeScript, and multi-agent frameworks.

---

## Quick Start: Enterprise Financial Guard

Three steps: define a Datalog policy, register a supervised action, and execute it through the gate.

### 1. Define the policy (`policy.dl`)

Gate payment refunds: require manager approval for amounts over $1,000 (100,000 cents), and block unverified destination accounts:

```prolog
Decl amount_cents(Req, Cents).
Decl destination_account(Req, Account).
Decl account_verified(Account).
Decl manager_approval(Req, Approver).
Decl has_manager_approval(Req).

has_manager_approval(Req) :- manager_approval(Req, _).

% Trusted baseline accounts
account_verified("acc_verified_123").

% T1 Halt: refund exceeds threshold without manager approval
halt("Req", "refund exceeds limit without manager approval", "T1") :-
    action_operation("Req", "process_refund"),
    amount_cents("Req", Amount),
    :gt(Amount, 100000),
    !has_manager_approval("Req").

% T1 Halt: destination account is unverified (account takeover risk)
halt("Req", "refund destination account is unverified", "T1") :-
    action_operation("Req", "process_refund"),
    destination_account("Req", Acc),
    !account_verified(Acc).
```

### 2. Write the skill (`main.go`)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/duynguyendang/manglekit"
    "github.com/duynguyendang/manglekit/adapters/func"
    "github.com/duynguyendang/manglekit/core"
)

type RefundRequest struct {
    AmountCents int    `mangle:"amount_cents"`
    Destination string `mangle:"destination_account"`
}

func main() {
    ctx := context.Background()

    client, err := manglekit.NewClient(ctx, manglekit.WithPolicyPath("policy.dl"))
    if err != nil {
        log.Fatalf("client init: %v", err)
    }
    defer client.Shutdown(ctx)

    // Register a supervised action
    client.RegisterSupervised("process_refund", function.New("process_refund",
        func(ctx context.Context, in RefundRequest) (string, error) {
            return fmt.Sprintf("Disbursed %d cents to %s", in.AmountCents, in.Destination), nil
        }))

    // Execute an unapproved $1,500 refund
    _, err = client.ExecuteByName(ctx, "process_refund", RefundRequest{
        AmountCents: 150000,
        Destination: "acc_verified_123",
    })

    if err != nil {
        fmt.Println("🛑 Action Blocked by Policy Gate:")
        fmt.Println(err)

        // Query the explanation proof tree
        expl, _ := client.Explain(ctx, `halt("Req", Reason, Tier)`, []string{
            `action_operation("Req", "process_refund").`,
            `amount_cents("Req", 150000).`,
            `destination_account("Req", "acc_verified_123").`,
        })
        fmt.Printf("\n--- Derivation Proof Tree ---\n%s\n", expl.String())
    }
}
```

### 3. Run it

```bash
go run .
```

Output:
```text
🛑 Action Blocked by Policy Gate:
policy violation: blocked at tier T1 (rule: refund exceeds limit without manager approval) (action: process_refund): Supervisor Pre-Flight check failed. [matched: halt("Req", "refund exceeds limit without manager approval", "T1")]

--- Derivation Proof Tree ---
query: halt("Req", Reason, Tier)
halt("Req","refund exceeds limit without manager approval","T1")
  └─ via rule [T1]: halt("Req","refund exceeds limit without manager approval","T1") :- action_operation("Req","process_refund"), amount_cents("Req",Amount), :gt(Amount,100000), !has_manager_approval("Req").
  action_operation("Req","process_refund")
  └─ [base fact]
  amount_cents("Req",150000)
  └─ [base fact]
  150000 > 100000
  └─ [builtin]
  !has_manager_approval("Req")
  └─ [negation]
```

The handler was **never called**. The action was halted by the fail-closed supervisor before execution, and the audit trail contains the grounded arithmetic proof (`150000 > 100000`).

---

## Canonical Guard Templates

Full runnable implementations and contract tests are available in [`manglekit-examples/governance/enterprise_guards`](https://github.com/duynguyendang/manglekit-examples/tree/main/governance/enterprise_guards):

1. **Financial & Refund Guard**: Limits disbursements, prevents unverified destination rerouting, and enforces dual-authorization facts.
2. **Destructive Infrastructure Guard**: Blocks `terraform destroy`, `DROP TABLE`, and container termination unless the requester is a verified owner and possesses a valid break-glass token.
3. **Communication & OTP Guard**: Prevents outbound notification tools from transmitting authentication codes or PII to unverified phone numbers or unencrypted channels.
4. **Production Showcase — Mosaic**: High-throughput data-cleaning and regulatory compliance pipeline running MangleKit in production.

---

## System Architecture

MangleKit maintains a strict boundary between its **governance core** and **optional extensions**:

```text
manglekit/
├── core/               # Domain vocabulary, types, errors (leaf package)
├── sdk/                # User facade: Client, Supervise, ExecuteByName, Explain
├── internal/
│   ├── supervisor/     # Zero-Trust Gate: fail-closed pre-check and post-check
│   └── engine/         # Datalog solver: Mangle runtime, AST analysis, IDB cache
├── adapters/           # Modular bridges: HTTP, MCP tools, Genkit, resilience
└── x/                  # Optional extensions (core never imports x/)
    ├── ooda/           # OODA loop runtime (ADR-004)
    ├── oodaflow/       # Genkit flow bridge for OODA
    ├── east/           # Entropy and steering (ADR-002)
    └── genes/          # Signed crystallized rule packages (ADR-003)
```

### Trust Tiers

Policies in MangleKit are organized by four trust tiers:

| Tier | Name | Semantics |
|---|---|---|
| **T0** | **Axiom** | Hard safety boundary (cannot be overridden by any lower rule). Blocks immediately. |
| **T1** | **Governance** | Organizational rules, limits, and role-based constraints. Blocks immediately. |
| **T2** | **Playbook** | Advisory operational guidance. Surfaces warnings, logged in audit trail, non-blocking. |
| **T3** | **User Preference** | Advisory user habits. Logged in audit trail, non-blocking. |

---

## Polyglot Usage: Sidecar Mode

For Python, TypeScript, and foreign runtimes, MangleKit runs as a standalone sidecar service:

```bash
# Start the HTTP policy gate
mkit serve --port 8080 --policy policy.dl
```

Send tool-call intent to `/v1/assess` before executing it in your agent runtime. Decisions are returned with decision IDs, SHA-256 policy fingerprints, and complete derivation trees.

---

## Contributing & Development

```bash
make test       # go test -race -count=1 ./... -v
make coverage   # coverage report (gate >= 60%)
make build      # build mkit CLI
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for pull request guidelines.

## License

Apache 2.0
