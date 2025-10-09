package manglekit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"
)

// componentCfg is a generic configuration for a named component with parameters.
type componentCfg struct {
	Name   string         `yaml:"name"`
	Params map[string]any `yaml:"params"`
}

// ToolConfig defines the configuration for a single, named tool that can be
// used within the declarative workflow. A tool is a configured instance of a
// provider (e.g., a specific retriever or LLM with its own settings).
type ToolConfig struct {
	// Provider is the registered name of the component provider (e.g., "openai", "bm25").
	Provider string `yaml:"provider"`
	// Params is a map of parameters used to configure the tool instance.
	Params map[string]any `yaml:"params"`
}

// OrchestratorConfig defines which orchestrator to use (e.g., "sandwich" or
// "declarative") and its specific settings.
type OrchestratorConfig struct {
	// Type specifies the orchestrator to use. If omitted, it defaults to "sandwich".
	Type string `yaml:"type"`
	// FlowName is the name of the flow to execute. This is required for the
	// "declarative" orchestrator, as it specifies the entry point for the Datalog query.
	FlowName string `yaml:"flowName,omitempty"`
}

// Config is the top-level struct for loading a MangleKit orchestrator's
// configuration from a YAML file. It defines the components to be used for each
// stage of the pipeline ("sandwich" mode) or the set of available tools
// ("declarative" mode), along with their parameters.
type Config struct {
	// Orchestrator selects and configures the workflow engine.
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	// Tools defines a map of named components that can be referenced as dependencies
	// in declarative flows. The map key is the tool's name.
	Tools map[string]ToolConfig `yaml:"tools"`
	// Providers holds global configurations for provider families, such as API keys,
	// which can be shared across multiple components.
	Providers ProviderConfigs `yaml:"providers"`
	// Embedder specifies the text embedding component for the "sandwich" orchestrator.
	Embedder componentCfg `yaml:"embedder"`
	// Retriever specifies the document retrieval component for the "sandwich" orchestrator.
	Retriever componentCfg `yaml:"retriever"`
	// VectorStore specifies the vector database component for the "sandwich" orchestrator.
	VectorStore componentCfg `yaml:"vectorStore"`
	// Reranker specifies the document reranking component for the "sandwich" orchestrator.
	Reranker componentCfg `yaml:"reranker"`
	// Rules specifies the rules engine component for the "sandwich" orchestrator.
	Rules componentCfg `yaml:"rules"`
	// LLM specifies the language model component for the "sandwich" orchestrator.
	LLM componentCfg `yaml:"llm"`
	// TopK is the default number of documents to retrieve in the "sandwich" orchestrator.
	TopK int `yaml:"topK"`
	// MaxTokens is the default maximum number of tokens for the LLM response in the
	// "sandwich" orchestrator.
	MaxTokens int `yaml:"maxTokens"`
	// FallbackThreshold is the confidence score below which a fallback is triggered
	// in the "sandwich" orchestrator.
	FallbackThreshold float64 `yaml:"fallbackThreshold"`
}

// ProviderConfigs holds the global configurations for different provider families.
// These settings, such as API keys or default models, can be shared across
// multiple components that belong to the same family.
type ProviderConfigs struct {
	// Google holds global settings for all Google-based providers (Gemini, Google Embedders).
	Google *GoogleConfig `yaml:"google,omitempty"`
	// OpenAI holds global settings for all OpenAI-based providers.
	OpenAI *OpenAIConfig `yaml:"openai,omitempty"`
	// Groq holds global settings for Groq, which uses an OpenAI-compatible API.
	Groq *OpenAICompatibleConfig `yaml:"groq,omitempty"`
	// OpenAICompatible holds global settings for other providers with OpenAI-like APIs.
	OpenAICompatible *OpenAICompatibleConfig `yaml:"openaiCompatible,omitempty"`
	// Mangle holds global settings for the Mangle rules engine.
	Mangle *MangleConfig `yaml:"mangle,omitempty"`
}

// MangleConfig holds global or default configuration for the Mangle rules engine provider.
type MangleConfig struct {
	// RulePaths is a list of file paths or glob patterns for Mangle rule files (.dlog).
	RulePaths []string `yaml:"path"`
	// PreProcess is a list of Mangle transformer names to run in the 'pre' stage.
	PreProcess []string `yaml:"preProcess"`
	// PostProcess is a list of Mangle transformer names to run in the 'post' stage.
	PostProcess []string `yaml:"postProcess"`
	// DefaultConverters specifies whether to include the built-in fact converters.
	DefaultConverters bool `yaml:"defaultConverters"`
}

// GoogleConfig holds global configuration for Google providers (e.g., Genkit, Gemini).
type GoogleConfig struct {
	// APIKey is the API key for Google AI services. If not provided, the builder
	// will check the `GOOGLE_API_KEY` environment variable.
	APIKey string `yaml:"apiKey,omitempty"`
	// Model is the default model identifier to use if a component does not specify one.
	Model string `yaml:"model,omitempty"`
	// PromptTemplate is a default prompt template string to use for LLM components.
	PromptTemplate string `yaml:"promptTemplate,omitempty"`
}

// OpenAIConfig holds global configuration specific to OpenAI providers.
type OpenAIConfig struct {
	// APIKey is the API key for OpenAI services. If not provided, the builder
	// will check the `OPENAI_API_KEY` environment variable.
	APIKey string `yaml:"apiKey,omitempty"`
	// Model is the default model identifier to use if a component does not specify one.
	Model string `yaml:"model,omitempty"`
	// Dimensions is the default embedding vector dimension for supported models.
	Dimensions int `yaml:"dimensions,omitempty"`
	// PromptTemplate is a default prompt template string to use for LLM components.
	PromptTemplate string `yaml:"promptTemplate,omitempty"`
}

// OpenAICompatibleConfig holds configuration for providers that use an
// OpenAI-compatible API, such as Groq or local models served via a similar interface.
type OpenAICompatibleConfig struct {
	// APIKey is the API key for the service. If not provided, the builder may
	// check a provider-specific environment variable (e.g., `GROQ_API_KEY`).
	APIKey string `yaml:"apiKey,omitempty"`
	// BaseURL is the base URL of the API endpoint (e.g., "https://api.groq.com/openai/v1").
	BaseURL string `yaml:"baseURL,omitempty"`
	// Model is the default model identifier to use for this provider.
	Model string `yaml:"model,omitempty"`
	// Dimensions is the default embedding vector dimension for this provider.
	Dimensions int `yaml:"dimensions,omitempty"`
}

// NewBuilderFromYAML reads a YAML configuration file, parses it, and returns a
// pre-configured Builder instance. This is the recommended and most powerful way
// to initialize MangleKit from a static configuration.
//
// The function automatically handles several important details:
//   - It expands environment variables within the YAML file using standard shell
//     syntax (e.g., `$VAR` or `${VAR}`), which is useful for injecting secrets like API keys.
//   - It uses a generic, reflection-based system to find and resolve all file paths
//     within the configuration that are tagged with `path:"resolve"`. This allows you
//     to use relative paths in your config file, which are resolved relative to the
//     config file's location.
//
// path is the file system path to the YAML configuration file.
// It returns a pre-configured, ready-to-use BuilderAPI instance, or an error
// if the file cannot be read, parsed, or if the configuration is invalid.
func NewBuilderFromYAML(path string) (BuilderAPI, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	expanded := []byte(os.ExpandEnv(string(b)))

	var cfg Config
	if err := yaml.Unmarshal(expanded, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	configDir := filepath.Dir(path)

	baseBuilder := NewBuilder()
	if impl, ok := baseBuilder.(*builder); ok {
		impl.configDir = configDir
	}

	builder := baseBuilder.
		WithConfig(&cfg). // Pass the full config to the builder
		WithTopK(cfg.TopK).
		WithMaxTokens(cfg.MaxTokens).
		WithFallbackThreshold(cfg.FallbackThreshold)

	// Helper function to create and configure a component from the YAML config.
	configureComponent := func(name string, params map[string]any, setter func(any) BuilderAPI) error {
		if name == "" {
			return nil
		}
		// Look up the Go type for the component's options struct.
		optsType, ok := nameToOptionsType[name]
		if !ok {
			return fmt.Errorf("no options type registered for component name %q", name)
		}

		// Create a new instance of the options struct (it's a pointer).
		optsPtr := reflect.New(optsType.Elem()).Interface()

		// Use JSON marshaling as a trick to convert map[string]any to the struct.
		// This is a common Go idiom that avoids a heavy dependency like mapstructure.
		jsonParams, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params for %q: %w", name, err)
		}
		if err := json.Unmarshal(jsonParams, optsPtr); err != nil {
			return fmt.Errorf("failed to unmarshal params into options struct for %q: %w", name, err)
		}

		// Now that the options struct is populated, resolve any paths within it.
		if err := resolvePathsInStruct(optsPtr, configDir); err != nil {
			return fmt.Errorf("failed to resolve paths for component %q: %w", name, err)
		}

		// Call the appropriate With... method on the builder.
		setter(optsPtr)
		return nil
	}

	if err := configureComponent(cfg.Embedder.Name, cfg.Embedder.Params, builder.WithEmbedder); err != nil {
		return nil, err
	}
	if err := configureComponent(cfg.Retriever.Name, cfg.Retriever.Params, builder.WithRetriever); err != nil {
		return nil, err
	}
	if err := configureComponent(cfg.VectorStore.Name, cfg.VectorStore.Params, builder.WithVectorStore); err != nil {
		return nil, err
	}
	if err := configureComponent(cfg.Reranker.Name, cfg.Reranker.Params, builder.WithReranker); err != nil {
		return nil, err
	}
	if err := configureComponent(cfg.Rules.Name, cfg.Rules.Params, builder.WithRules); err != nil {
		return nil, err
	}
	if err := configureComponent(cfg.LLM.Name, cfg.LLM.Params, builder.WithLLM); err != nil {
		return nil, err
	}

	return builder, nil
}

// NewBuilderFromEnv creates a new Builder instance configured entirely from
// environment variables. This is useful for containerized or CI/CD environments
// where file-based configuration is inconvenient.
//
// The function looks for variables following the pattern:
//   - `MKT_LLM_NAME`: The name of the LLM provider (e.g., "google").
//   - `MKT_LLM_PARAMS`: A JSON string of parameters for the LLM.
//   - `MKT_RETRIEVER_NAME`: The name of the retriever provider.
//   - `MKT_RETRIEVER_PARAMS`: A JSON string of parameters for the retriever.
//
// ...and so on for "EMBEDDER", "RERANKER", "RULES", and "VECTORSTORE".
// It also reads `MKT_TOPK`, `MKT_MAX_TOKENS`, and `MKT_FALLBACK_THRESHOLD`.
func NewBuilderFromEnv() (BuilderAPI, error) {
	baseBuilder := NewBuilder()

	// Since there's no file, the config dir is the current working directory.
	// This is important for resolving any relative paths in the JSON params.
	configDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	if impl, ok := baseBuilder.(*builder); ok {
		impl.configDir = configDir
	}

	// Helper to read an env var and configure a component
	configureComponentFromEnv := func(
		componentName string,
		setter func(any) BuilderAPI,
	) error {
		name := os.Getenv(fmt.Sprintf("MKT_%s_NAME", componentName))
		if name == "" {
			return nil
		}

		paramsJSON := os.Getenv(fmt.Sprintf("MKT_%s_PARAMS", componentName))
		var params map[string]any
		if paramsJSON != "" {
			if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
				return fmt.Errorf("invalid JSON for %s params: %w", componentName, err)
			}
		}

		optsType, ok := nameToOptionsType[name]
		if !ok {
			return fmt.Errorf("no options type registered for component name %q", name)
		}
		optsPtr := reflect.New(optsType.Elem()).Interface()

		jsonParams, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params for %q: %w", name, err)
		}
		if err := json.Unmarshal(jsonParams, optsPtr); err != nil {
			return fmt.Errorf("failed to unmarshal params for %q: %w", name, err)
		}
		if err := resolvePathsInStruct(optsPtr, configDir); err != nil {
			return fmt.Errorf("failed to resolve paths for %q: %w", name, err)
		}

		setter(optsPtr)
		return nil
	}

	// Configure each component
	if err := configureComponentFromEnv("LLM", baseBuilder.WithLLM); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("EMBEDDER", baseBuilder.WithEmbedder); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("RETRIEVER", baseBuilder.WithRetriever); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("VECTORSTORE", baseBuilder.WithVectorStore); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("RERANKER", baseBuilder.WithReranker); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("RULES", baseBuilder.WithRules); err != nil {
		return nil, err
	}

	// Configure top-level options
	if topKStr := os.Getenv("MKT_TOPK"); topKStr != "" {
		var topK int
		if _, err := fmt.Sscanf(topKStr, "%d", &topK); err == nil {
			baseBuilder.WithTopK(topK)
		}
	}
	if maxTokensStr := os.Getenv("MKT_MAX_TOKENS"); maxTokensStr != "" {
		var maxTokens int
		if _, err := fmt.Sscanf(maxTokensStr, "%d", &maxTokens); err == nil {
			baseBuilder.WithMaxTokens(maxTokens)
		}
	}
	if fallbackStr := os.Getenv("MKT_FALLBACK_THRESHOLD"); fallbackStr != "" {
		var fallback float64
		if _, err := fmt.Sscanf(fallbackStr, "%f", &fallback); err == nil {
			baseBuilder.WithFallbackThreshold(fallback)
		}
	}

	return baseBuilder, nil
}

// resolvePathsInStruct recursively traverses a struct, slice, or pointer and
// resolves any string or []string fields tagged with `path:"resolve"`.
//
// The path is resolved relative to the provided baseDir. The function modifies
// the fields in place using reflection. It's designed to work on the unmarshaled
// YAML config struct.
func resolvePathsInStruct(data any, baseDir string) error {
	v := reflect.ValueOf(data)

	// Dereference pointers and interfaces to get the underlying value.
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i)
			if !fieldVal.CanSet() {
				continue
			}

			// Check for the `path:"resolve"` tag on the struct field.
			if v.Type().Field(i).Tag.Get("path") == "resolve" {
				switch fieldVal.Kind() {
				case reflect.String:
					// Resolve a single string path.
					path := fieldVal.String()
					if path != "" && !filepath.IsAbs(path) {
						fieldVal.SetString(filepath.Join(baseDir, path))
					}
				case reflect.Slice:
					// Resolve a slice of string paths.
					if fieldVal.Type().Elem().Kind() == reflect.String {
						for j := 0; j < fieldVal.Len(); j++ {
							path := fieldVal.Index(j).String()
							if path != "" && !filepath.IsAbs(path) {
								fieldVal.Index(j).SetString(filepath.Join(baseDir, path))
							}
						}
					}
				}
			}

			// Recursively call on the field to handle nested structs, pointers, or slices.
			if err := resolvePathsInStruct(fieldVal.Addr().Interface(), baseDir); err != nil {
				return err
			}
		}
	case reflect.Slice:
		// If the top-level item is a slice, iterate over its elements.
		for i := 0; i < v.Len(); i++ {
			if err := resolvePathsInStruct(v.Index(i).Addr().Interface(), baseDir); err != nil {
				return err
			}
		}
	}

	return nil
}
