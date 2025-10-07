# Review: `examples/04-chat-w-data`

This document captures structural refactoring opportunities for the Chat-with-Data example. It focuses on separating concerns, simplifying tests, and preparing the package for reuse in other flows.

## High-level structure

* **Observation:** `main.go` mixes demo scaffolding (document fixture, query fixture, console printing) with reusable policy wiring.
  * **Suggestion:** Move the reusable pieces into a small package (for example `examples/04-chat-w-data/app`). Export helpers such as `BuildUser(meta map[string]any) (policy.User, error)` and `FilterDocs(user policy.User, docs []core.Doc) []core.Citation`. The `main` package can then only orchestrate: load env, call helper, render output.
  * **Benefit:** Makes the policy pipeline usable from tests or other binaries and keeps `main()` minimal.

* **Observation:** The metadata extraction logic in `main.go` and `main_test.go` is duplicated.
  * **Suggestion:** Centralize the parsing in a helper (`policy.UserFromQuery(query core.Query)`). Return a typed error when attributes are missing.
  * **Benefit:** Avoids drift between demo code and tests, and enables reuse when the example is embedded in a larger app.

* **Observation:** Inline fixtures make it hard to evolve the example as test cases grow.
  * **Suggestion:** Move fixtures to top-level package-level vars or helper functions (`func sampleDocs() []core.Doc`). Consider loading from JSON or YAML in `testdata/` for readability.
  * **Benefit:** Reduces clutter in `main.go` and makes future table-driven tests straightforward.

## Policy package (`policy/policy.go`)

* **Observation:** Policy logic is cohesive but lacks clear separation between parsing and enforcement.
  * **Suggestion:** Expose the parsing step as a separate type (e.g. `type Parser interface { Parse(core.Doc) (Doc, error) }`) so callers can swap in a real parser. `ParseDoc` could return an error when key-value pairs are malformed instead of silently ignoring them.
  * **Benefit:** Makes the policy layer testable with malformed input and easier to embed in real ingestion flows.

* **Observation:** Policy enforcement is currently embedded directly in Go code.
  * **Suggestion:** Model the access policy with **Mangle** facts and rules instead of imperative Go conditionals. Define the facts that describe users, documents, and entitlements in `.mangle` files (or another rule bundle), then load and evaluate them through the Mangle engine from Go.
  * **Benefit:** Keeps policy logic declarative, enables hot-swapping rule bundles without recompiling, and matches the intended "rules-first" architecture of the framework.

* **Observation:** Sensitivity rules are currently hard-coded per document ID.
  * **Suggestion:** Accept a configuration map (maybe per department) or infer sensitivity from metadata. Document the fallback strategy in code comments to signal this is demo-only behavior.
  * **Benefit:** Prevents the example from encouraging ID-based ACLs and clarifies how to extend it.

* **Observation:** `VisibleColumns`/`MaskedColumns` functions are not used in the example and leak internal policy shape.
  * **Suggestion:** Either remove them from the public surface or showcase them in `main.go` by masking the response. If kept, add tests that assert masking behavior for privileged vs. non-privileged users.
  * **Benefit:** Ensures exported API matches demonstrated usage.

## Testing (`main_test.go`)

* **Observation:** Test recreates production logic instead of reusing helpers.
  * **Suggestion:** After extracting helpers, tests can invoke them directly. Consider table-driven tests that vary role/department/purpose and assert retrieved doc IDs plus masking output.
  * **Observation:** Assertions currently guard against nil slices before checking content.
  * **Suggestion:** Use `require.Len` to stop early on wrong counts and keep the test tighter.

## Additional opportunities

* Add a README or docstring explaining how to plug the policy into the broader sandwich pipeline (pre-retrieve -> filter -> mask -> LLM). This context helps users graduate from the demo to production.
* Consider wiring a lightweight retriever interface so that the policy filter feeds into actual retrieval rather than reusing the static slice.

