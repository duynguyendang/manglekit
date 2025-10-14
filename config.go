package manglekit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"

	"gopkg.in/yaml.v3"
)

// BuilderAPI defines the fluent interface for the MangleKit builder.
// It is used by the YAML and environment variable constructors to provide a
// consistent, chainable API for configuration.
type BuilderAPI interface {
	WithConfig(*Config) BuilderAPI
	WithRetriever(any) BuilderAPI
	WithVectorStore(any) BuilderAPI
	WithReranker(any) BuilderAPI
	WithRules(any) BuilderAPI
	WithLLM(any) BuilderAPI
	WithFlow(string) BuilderAPI
	WithEmbedder(any) BuilderAPI
	WithTopK(int) BuilderAPI
	WithMaxTokens(int) BuilderAPI
	WithObservability(core.Observability) BuilderAPI
	WithFallbackThreshold(float64) BuilderAPI
	Build(context.Context) (core.Orchestrator, error)
}

// componentCfg is a generic configuration for a named component with parameters.
// It is used in the "sandwich" orchestrator configuration to define which
// provider to use for a given role (e.g., retriever, llm) and its settings.
type componentCfg struct {
	// Name is the registered name of the provider for this component (e.g., "bm25", "google").
	Name string `yaml:"name"`
	// Params is a map of parameters used to configure the provider instance. The
	// structure of this map must match the provider's specific options struct.
	Params map[string]any `yaml:"params"`
}

// ToolConfig defines the configuration for a single, named tool that can be
// used within the declarative workflow. A tool is a configured instance of a
// provider (e.g., a specific retriever or LLM with its own settings) that can
// be referenced by name in Datalog rules.
type ToolConfig struct {
	// Provider is the registered name of the component provider (e.g., "openai", "bm25", "google-embedder").
	Provider string `yaml:"provider"`
	// Params is a map of parameters used to configure the tool instance. The
	// structure must match the provider's specific options struct. Dependencies
	// on other tools are specified here by providing the name of the dependency
	// tool as a string value (e.g., `embedder: "my_embedder_tool"`).
	Params map[string]any `yaml:"params"`
}

// OrchestratorConfig defines which orchestrator to use (e.g., "sandwich" or
// "declarative") and its specific settings.
type OrchestratorConfig struct {
	// Type specifies the orchestrator to use. Supported values are "sandwich" and
	// "declarative". If omitted, it defaults to "sandwich".
	Type string `yaml:"type"`
	// FlowName is the name of the flow to execute. This is required for the
	// "declarative" orchestrator, as it specifies the entry point for the Datalog query
	// that drives the workflow.
	FlowName string `yaml:"flowName,omitempty"`
}

// LoggingConfig defines runtime logging behavior for the SDK.
type LoggingConfig struct {
	// Level controls the minimum log level emitted by the zap adapter.
	Level string `yaml:"level"`
	// Format controls the encoder used by the zap adapter ("json" or "console").
	Format string `yaml:"format"`
}

// Config is the top-level struct for loading a MangleKit orchestrator's
// configuration from a YAML file. It supports two primary modes of operation,
// controlled by the `Orchestrator.Type` field.
//
// In "sandwich" mode, it uses the top-level component fields (`Retriever`, `LLM`, etc.)
// to define a fixed, linear pipeline.
//
// In "declarative" mode, it uses the `Tools` map to define a collection of
// available components that can be orchestrated dynamically by a rules engine.
type Config struct {
	// Orchestrator selects and configures the workflow engine.
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	// Logging defines global logging behavior for the orchestrator runtime.
	Logging *LoggingConfig `yaml:"logging,omitempty"`
	// Tools defines a map of named components that can be referenced as dependencies
	// in declarative flows. The map key is the tool's name. This is used only
	// by the "declarative" orchestrator.
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
// multiple components that belong to the same family, reducing duplication.
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
// @param path is the file system path to the YAML configuration file.
// @return A pre-configured, ready-to-use BuilderAPI instance, or an error
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
	baseBuilder.configDir = configDir

	if cfg.Logging != nil {
		logImpl, err := logger.New(logger.Config{Level: cfg.Logging.Level, Format: cfg.Logging.Format})
		if err != nil {
			return nil, fmt.Errorf("failed to configure logger: %w", err)
		}
		baseBuilder.WithObservability(core.Observability{Logger: logImpl})
	}

	builder := baseBuilder.
		WithConfig(&cfg). // Pass the full config to the builder
		WithTopK(cfg.TopK).
		WithMaxTokens(cfg.MaxTokens).
		WithFallbackThreshold(cfg.FallbackThreshold)

		// Helper function to create and configure a component from the YAML config.
	configureComponent := func(componentType, name string, params map[string]any, setter func(any) BuilderAPI) error {
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
		switch componentType {
		case "embedder":
			baseBuilder.embedderName = name
		case "retriever":
			baseBuilder.retrieverName = name
		case "vectorStore":
			baseBuilder.vectorStoreName = name
		case "reranker":
			baseBuilder.rerankerName = name
		case "rules":
			baseBuilder.rulesName = name
		case "llm":
			baseBuilder.llmName = name
		}
		return nil
	}

	if err := configureComponent("embedder", cfg.Embedder.Name, cfg.Embedder.Params, builder.WithEmbedder); err != nil {
		return nil, err
	}
	if err := configureComponent("retriever", cfg.Retriever.Name, cfg.Retriever.Params, builder.WithRetriever); err != nil {
		return nil, err
	}
	if err := configureComponent("vectorStore", cfg.VectorStore.Name, cfg.VectorStore.Params, builder.WithVectorStore); err != nil {
		return nil, err
	}
	if err := configureComponent("reranker", cfg.Reranker.Name, cfg.Reranker.Params, builder.WithReranker); err != nil {
		return nil, err
	}
	if err := configureComponent("rules", cfg.Rules.Name, cfg.Rules.Params, builder.WithRules); err != nil {
		return nil, err
	}
	if err := configureComponent("llm", cfg.LLM.Name, cfg.LLM.Params, builder.WithLLM); err != nil {
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
//
// @return A pre-configured BuilderAPI instance or an error if configuration is invalid.
func NewBuilderFromEnv() (BuilderAPI, error) {
	baseBuilder := NewBuilder()

	// Since there's no file, the config dir is the current working directory.
	// This is important for resolving any relative paths in the JSON params.
	configDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	baseBuilder.configDir = configDir

	level := os.Getenv("MKT_LOG_LEVEL")
	format := os.Getenv("MKT_LOG_FORMAT")
	if level != "" || format != "" {
		logImpl, err := logger.New(logger.Config{Level: level, Format: format})
		if err != nil {
			return nil, fmt.Errorf("failed to configure logger from environment: %w", err)
		}
		baseBuilder.WithObservability(core.Observability{Logger: logImpl})
	}

	// Helper to read an env var and configure a component
	configureComponentFromEnv := func(
		componentType string,
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
		switch componentType {
		case "embedder":
			baseBuilder.embedderName = name
		case "retriever":
			baseBuilder.retrieverName = name
		case "vectorStore":
			baseBuilder.vectorStoreName = name
		case "reranker":
			baseBuilder.rerankerName = name
		case "rules":
			baseBuilder.rulesName = name
		case "llm":
			baseBuilder.llmName = name
		}
		return nil
	}

	// Configure each component
	if err := configureComponentFromEnv("llm", "LLM", baseBuilder.WithLLM); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("embedder", "EMBEDDER", baseBuilder.WithEmbedder); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("retriever", "RETRIEVER", baseBuilder.WithRetriever); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("vectorStore", "VECTORSTORE", baseBuilder.WithVectorStore); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("reranker", "RERANKER", baseBuilder.WithReranker); err != nil {
		return nil, err
	}
	if err := configureComponentFromEnv("rules", "RULES", baseBuilder.WithRules); err != nil {
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
