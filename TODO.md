# Manglekit TODO List

This file tracks outstanding tasks based on LLD gaps and roadmap. Tasks are prioritized by phase for MVP completion.

- [ ] Implement ingestion flow: Create ingest/ package with Ingestor interface for document chunking (fixed-size + overlap), embedding, and indexing. Add async goroutines for batch processing.
- [ ] Add /v1/ingest endpoint to cmd/agent/main.go: Handle multipart uploads, queue jobs, return job_id for status tracking.
- [x] Implement FromEnv in config.go: Parse MKT_* environment variables (e.g., MKT_LLM_NAME, MKT_LLM_PARAMS as JSON) and build via Builder.
- [ ] Add security middleware to HTTP service: JWT parsing (claims to Query.Meta), input sanitization (html.Escape), context timeouts, and rate limiting (semaphore).
- [ ] Enhance Mangle rules: Extend RuleResult with Explanations array; implement PII redaction in post-mutate (regex-based); add BoltDB facts store for persistent ontologies.
- [ ] Add LLM prompt templating: Support Go text/template in Client.Complete for customizable prompts (e.g., include citations format).
- [ ] Implement LLM caching: Add sync.Map for prompt+context hashes to cache responses; integrate in providers with TTL.
- [ ] Integrate Genkit: Create GenkitOrchestrator alternative; load YAML flows for intent routing while preserving Sandwich rules.
- [ ] Expand testing: Add unit tests for ingestion/HTTP (mocks for Index/queue); E2E for edges (policy deny, no-evidence, token overflow with truncation); aim >90% coverage.
- [ ] Add multi-modal support (phase 4): Extend Doc for images/audio; pluggable embedders (e.g., CLIP); update Retrieve for hybrid text/multi-modal search.