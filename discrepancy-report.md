# Discrepancy Report

This report details the findings of the architectural audit performed on 2025-11-05.

## Summary

The audit revealed a critical discrepancy between the codebase and the architectural documentation. While most of the "Resolved" smells in `docs/code-review.md` were verified as fixed, the "Builder Leaking into Handler" smell (a violation of ADR 7 / R14) persists throughout the codebase, despite being documented as "Resolved". This indicates that the documentation is out of sync with the code and the codebase is not in a stable state.

## Detailed Findings

### Violation: Builder Leaking into Handler (ADR 7 / R14)

*   **Description:** Handlers are directly type-asserting the `any` builderDI interface to the concrete `diapi.Builder`, which violates the principle of Type-Safe DI. Handlers should instead resolve dependencies through the typed methods of the `diapi.Builder` interface and pass them to factories via specific `diapi.*Deps` structs.

*   **Locations:**
    *   `pipeline/declarative/handler.go:33:	builder, ok := builderDI.(diapi.Builder)`
    *   `pipeline/sandwich/handler.go:33:	b, ok := builderDI.(diapi.Builder)`
    *   `internal/providers/llm/handler.go:33:	b, ok := builderDI.(diapi.Builder)`
    *   `internal/providers/rerank/handler.go:33:	b, ok := builderDI.(diapi.Builder)`
    *   `internal/providers/retrievers/handler.go:33:	b, ok := builderDI.(diapi.Builder)`
    *   `internal/providers/state/handler.go:33:	b, ok := builderDI.(diapi.Builder)`
    *   `internal/providers/schemaparsers/handler.go:33:	b, ok := builderDI.(diapi.Builder)`
    *   `internal/providers/rules/handler.go:33:	b, ok := builderDI.(diapi.Builder)`

*   **Conflict:** The architectural documents (`CONTEXT.md`, `LLD.md`, `ADR.md`, and `code-review.md`) all incorrectly state that this violation has been "Resolved" as of 2025-11-05. The evidence from the codebase directly contradicts this claim.

### Other Audited Items

*   **"Resolved" Smells:** The audit confirmed that the "Polluted BuilderAPI," "Non-Deterministic Type Resolution," "Factory Signature Mismatch," and "Non-Deterministic Reranking Tie-Breaking" smells are indeed resolved as stated in the documentation.
*   **Filesystem Structure:** The filesystem structure of `internal/providers/` was found to be compliant with the architectural guidelines.

## Conclusion

The presence of the "Builder Leaking into Handler" violation across multiple core components means the system is not compliant with ADR 7. The documentation must be updated to reflect this reality, and the stability of the codebase should be re-classified as "unstable."
