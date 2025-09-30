---
title: Manglekit Overview
version: v1.0
tags: [golang, rag, ai]
---

# Manglekit

Manglekit is a framework for building Retrieval-Augmented Generation (RAG) applications in Go.
It uses a "Sandwich Pattern" to wrap a RAG pipeline with Mangle, a declarative rules engine.

## Core Concepts

- **Mangle-Pre**: Normalizes queries and applies pre-retrieval policies.
- **Retrieval**: Fetches relevant documents.
- **Mangle-Post**: Filters and redacts retrieved content before generation.
- **Generation**: Synthesizes an answer using an LLM.