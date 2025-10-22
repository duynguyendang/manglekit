# Manglekit — Architecture & Code Smell Audit

**Date:** 2025-10-22
**Scope:** . (excluding examples/**, docs/**, legacy-examples/**)

## Summary
- Total findings: 1 (Errors: 1, Warnings: 0)
- Files affected: 1

## Findings
| Rule | Level | File | Line | Message |
|------|-------|------|------|---------|
| R13 | error | core/diapi/di.go | 4 | Core package dependency violation: core must not import internal/providers, pipeline, or the root builder. |

## Recommendations
- R13: Remove the import of `github.com/duynguyendang/manglekit/core` from `core/diapi/di.go`. The `core` package should not depend on itself or its subpackages in this manner. It seems like a circular dependency.
