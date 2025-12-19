# OpenRouter / LocalAI Demo
This example demonstrates how to connect Manglekit to an OpenAI-compatible provider (like OpenRouter, vLLM, or LocalAI) by specifying a custom Base URL.

## Prerequisites
* Go 1.22+
* An OpenRouter API key (or LocalAI setup)

## Usage

1. Set your environment variables:
   ```bash
   export OPENAI_API_KEY="sk-or-v1-..."
   export OPENAI_BASE_URL="https://openrouter.ai/api/v1"
   ```

2. Run the example:
   ```bash
   go run examples/openrouter_demo/main.go
   ```
