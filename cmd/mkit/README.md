# mkit: The Manglekit CLI

`mkit` is the command-line interface for **Manglekit**, the Neuro-Symbolic AI Governance framework. It provides a suite of tools to generate policies, evaluate logic, inspect data structures, and manage knowledge graphs.

## Installation

```bash
go install github.com/duynguyendang/manglekit/cmd/mkit@latest
```

## Commands

### 1. `mkit eval`
Evaluate a Datalog query against a policy blueprint, input data (JSON), and optional knowledge graph facts.

**Usage:**
```bash
mkit eval -p <policy.dl> -q <query> [flags]
```

**Flags:**
- `-p, --policy` (Required): Path to the Datalog policy file (`.dl`).
- `-q, --query` (Required): The Datalog query to execute (e.g., `deny(S, R)?`).
- `-d, --data`: Path to a JSON input file to be loaded as facts.
- `-k, --knowledge` (or `--facts`): Path to a Knowledge Graph file (`.nq`, `.nt`, `.ttl`).

**Example:**
```bash
mkit eval -p policy.dl -d input.json -q "deny(S, R)?"
```

---

### 2. `mkit generate rule`
Generate a Datalog rule from a natural language description using an LLM. This command uses a Neuro-Symbolic Feedback Loop (Teacher-Student Protocol) to ensure valid syntax.

**Usage:**
```bash
mkit generate rule --model <model_name> --prompt <description> [flags]
```

**Flags:**
- `--model` (Required): Model name to use (e.g., `gemini-2.0-flash`, `gpt-4o`).
- `--provider`: LLM Provider (`google` or `openai`). Default: `google`.
- `--prompt` (or `--desc`): The natural language description of the policy.
- `--vocab` (or `--facts`): Domain vocabulary (predicates) to guide generation.
- `--sample`: Path to a sample file (`.json` or `.nq`) to infer schema/structure.
- `--icl`: Path to a custom Datalog example file for In-Context Learning.

**Example:**
```bash
mkit generate rule \
  --model gemini-2.0-flash \
  --prompt "Deny requests if the user is not an admin and the amount > 1000" \
  --sample request.json
```

---

### 3. `mkit inspect struct`
Inspect a JSON structure and visualize how Manglekit converts it into Datalog facts. Useful for debugging schema mapping.

**Usage:**
```bash
mkit inspect struct --json <file_or_string>
```

**Flags:**
- `--json` (Required): Path to a JSON file or a raw JSON string.

**Example:**
```bash
mkit inspect struct --json request.json
```

---

### 4. `mkit kg convert`
Convert between different Knowledge Graph formats (e.g., Turtle to N-Quads).

**Usage:**
```bash
mkit kg convert -i <input_file> -o <output_file> [flags]
```

**Flags:**
- `-i, --input` (Required): Input file path.
- `-o, --output` (Required): Output file path.
- `--from`: Input format (auto-detected if empty).
- `--to`: Output format (`nquads`, `ntriples`, `jsonld`). Default: `nquads`.

**Example:**
```bash
mkit kg convert -i data.ttl -o data.nq
```

---

### 5. `mkit serve`
Start the Manglekit HTTP Server to expose the SDK via an API.

**Usage:**
```bash
mkit serve [flags]
```

**Flags:**
- `-p, --port`: Port to listen on (default: `8080`).
- `-f, --policy`: Path to the Datalog blueprint/policy file to enforce.
- `-m, --mcp`: Path to an MCP configuration file.

**Example:**
```bash
mkit serve --port 9090 --policy firewall.dl
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GOOGLE_API_KEY` | Required for Google (Gemini) models in `generate rule`. |
| `OPENAI_API_KEY` | Required for OpenAI models in `generate rule`. |
