# Configuration Package - Infrastructure-as-Code Support

The `config` package provides lightweight, production-ready configuration management for Manglekit applications. It enables declarative infrastructure-as-code workflows through YAML configuration files with full environment variable expansion.

## Quick Start

### 1. Create a Configuration File

Create `mangle.yaml` in your application directory:

```yaml
# Policy configuration
policy:
  path: ${POLICY_PATH:-./policies/main.dl}
  evaluation_timeout: 30

# Observability settings
observability:
  enabled: true
  service_name: ${SERVICE_NAME:-my-service}
  log_level: ${LOG_LEVEL:-info}
  otlp_endpoint: ${OTLP_ENDPOINT:-}

# Pre-defined actions (reserved for future use)
actions:
  llm_google:
    type: llm
    provider: google
    options:
      model: gemini-pro
      temperature: 0.7
```

### 2. Load Configuration in Your Code

```go
package main

import (
	"context"
	"log"

	"github.com/duynguyendang/manglekit"
)

func main() {
	ctx := context.Background()

	// Load configuration from YAML file
	client, err := manglekit.NewClientFromConfig(ctx, "mangle.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Use the client...
	action := manglekit.ProtectFunc(client, "myAction", myFunction)
	// ...
}
```

## Configuration Schema

### Top-Level Config

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `policy` | PolicyConfig | No | Policy engine settings |
| `observability` | ObservabilityConfig | No | Logging and tracing configuration |
| `actions` | map[string]ActionConfig | No | Pre-defined action templates |

### PolicyConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `path` | string | "" | Path to Datalog policy file (.dl or .dlog) |
| `evaluation_timeout` | int | 0 | Policy evaluation timeout in seconds |

### ObservabilityConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | false | Enable/disable observability |
| `service_name` | string | "manglekit-app" | Service name for observability |
| `log_level` | string | "info" | Logging level (debug, info, warn, error) |
| `otlp_endpoint` | string | "" | OpenTelemetry Protocol endpoint |

### ActionConfig

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Action type (llm, retriever, embedder, etc.) |
| `provider` | string | Provider name (google, openai, anthropic, etc.) |
| `options` | map[string]interface{} | Provider-specific configuration |

## Environment Variable Expansion

The configuration loader supports POSIX-style environment variable expansion:

### Basic Expansion

```yaml
policy:
  path: ${POLICY_PATH}  # Requires POLICY_PATH to be set
```

Environment: `POLICY_PATH=/opt/policies/main.dl`  
Result: `path: /opt/policies/main.dl`

### Default Values

```yaml
policy:
  path: ${POLICY_PATH:-./policies/default.dl}  # Uses default if unset
```

If `POLICY_PATH` is unset:  
Result: `path: ./policies/default.dl`

### Common Patterns

```yaml
observability:
  service_name: ${SERVICE_NAME:-my-app}
  log_level: ${LOG_LEVEL:-info}
  otlp_endpoint: ${OTEL_EXPORTER_OTLP_ENDPOINT:-http://localhost:4317}
```

## Usage Examples

### Example 1: Production Setup

```yaml
# config/production.yaml
policy:
  path: /etc/manglekit/policies/production.dl
  evaluation_timeout: 60

observability:
  enabled: true
  service_name: governance-service
  log_level: info
  otlp_endpoint: http://otel-collector.monitoring:4317
```

```go
client, err := manglekit.NewClientFromConfig(ctx, "config/production.yaml")
```

### Example 2: Development with Defaults

```yaml
# mangle.yaml (minimal config)
observability:
  enabled: true
```

```go
// Uses defaults: 
// - policy.path: "" (no policy loaded)
// - service_name: "manglekit-app"
// - log_level: "info"
client, err := manglekit.NewClientFromConfig(ctx, "mangle.yaml")
```

### Example 3: Environment-Driven Config

```yaml
# config/template.yaml
policy:
  path: ${POLICY_PATH}
  evaluation_timeout: ${POLICY_TIMEOUT:-30}

observability:
  enabled: true
  service_name: ${APP_NAME}
  log_level: ${LOG_LEVEL:-info}
  otlp_endpoint: ${OTEL_ENDPOINT}
```

```bash
# Set environment variables
export POLICY_PATH=/opt/policies/main.dl
export APP_NAME=my-governance-service
export LOG_LEVEL=debug
export OTEL_ENDPOINT=http://localhost:4317

# Run application
go run ./main.go config/template.yaml
```

### Example 4: With Custom Options

```go
import "go.opentelemetry.io/otel"

tp := otel.GetTracerProvider()

client, err := manglekit.NewClientFromConfig(
	ctx, 
	"mangle.yaml",
	manglekit.WithTracerProvider(tp),
	manglekit.WithLogger(customLogger),
)
```

## API Reference

### config.Load

```go
func Load(path string) (*Config, error)
```

Loads a configuration file from the given path. Environment variables in the file are expanded using standard POSIX syntax.

**Returns:** Loaded Config struct with defaults applied, or error if file not found or YAML is invalid.

### config.ParseConfig

```go
func ParseConfig(data []byte) (*Config, error)
```

Parses YAML bytes into a Config struct. Useful for programmatic configuration.

**Returns:** Config struct with defaults applied.

### config.LoadFromReader

```go
func LoadFromReader(r io.Reader) (*Config, error)
```

Loads configuration from any io.Reader (files, HTTP responses, etc.).

**Returns:** Config struct with defaults applied.

### manglekit.NewClientFromConfig

```go
func NewClientFromConfig(
	ctx context.Context,
	configPath string,
	opts ...ClientOption,
) (*Client, error)
```

Creates a new Manglekit Client from a configuration file. This is the recommended way to initialize Manglekit in production.

**Features:**
- Loads configuration from YAML file
- Expands environment variables
- Initializes observability
- Loads policy from configured path
- Supports client customization via options

## Error Handling

The configuration loader provides detailed error messages for troubleshooting:

```go
client, err := manglekit.NewClientFromConfig(ctx, "mangle.yaml")
if err != nil {
	// Errors are wrapped with context:
	// "failed to load configuration: failed to read config file..."
	// "failed to load policy from ...: failed to load policy rules..."
	log.Fatal(err)
}
```

## Testing

The package includes comprehensive tests:

```bash
go test ./config -v
go test -v -run TestNewClientFromConfig .
```

**Test Coverage:**
- ✅ Valid YAML loading
- ✅ Environment variable expansion
- ✅ Default value application
- ✅ Error handling (missing files, invalid YAML)
- ✅ Reader-based loading
- ✅ Integration with Manglekit client

## Best Practices

### 1. Use Defaults for Development

For development, rely on defaults and only override what's necessary:

```yaml
# mangle.yaml
observability:
  enabled: true
```

### 2. Use Environment Variables for Secrets

Never commit secrets to configuration files:

```yaml
# ✅ Good
observability:
  service_name: ${SERVICE_NAME}

# ❌ Bad
observability:
  api_key: sk-1234567890  # Don't do this!
```

### 3. Template Configuration for Multiple Environments

```yaml
# config/base.yaml
observability:
  enabled: true

# config/production.yaml
include: config/base.yaml
observability:
  log_level: info
  otlp_endpoint: ${OTEL_ENDPOINT}

# config/development.yaml
include: config/base.yaml
observability:
  log_level: debug
```

### 4. Version Control Configuration

```bash
# Good - commit configuration structure
git add mangle.yaml config/production.yaml

# Good - ignore environment-specific values
echo ".env" >> .gitignore
git add .gitignore
```

### 5. Document Required Environment Variables

```bash
# .env.example
POLICY_PATH=/opt/policies/main.dl
SERVICE_NAME=my-service
LOG_LEVEL=info
OTEL_ENDPOINT=http://localhost:4317
```

## Troubleshooting

### Issue: "failed to read config file"

**Cause:** File not found or permission denied  
**Solution:** Check file path and permissions:
```bash
ls -la mangle.yaml
```

### Issue: "failed to unmarshal YAML"

**Cause:** Invalid YAML syntax  
**Solution:** Validate YAML structure:
```bash
cat mangle.yaml | python -m yaml.safe_load
```

### Issue: Environment variables not expanded

**Cause:** Variable not set or wrong syntax  
**Solution:** Check variable is exported:
```bash
export POLICY_PATH=/path/to/policy.dl
echo $POLICY_PATH  # Should print the path
```

### Issue: Policy loading fails

**Cause:** Policy file not found or invalid Datalog syntax  
**Solution:** Verify policy file exists and is valid Datalog:
```bash
ls -la ${POLICY_PATH}
cat ${POLICY_PATH}
```

## Contributing

To add features to the config package:

1. Update types in `config/config.go`
2. Update loader logic in `config/loader.go`
3. Add tests in `config/loader_test.go`
4. Update this documentation
5. Run tests: `go test ./config -v`

## License

Same as Manglekit framework.
