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

// ToolConfig defines the configuration for a single tool in the declarative workflow.
// A tool is a configured instance of a provider (e.g., a specific retriever or LLM).
type ToolConfig struct {
	Provider string         `yaml:"provider"`
	Params   map[string]any `yaml:"params"`
}

// OrchestratorConfig defines which orchestrator to use and its specific settings.
type OrchestratorConfig struct {
	// Type specifies the orchestrator to use, e.g., "sandwich" or "declarative".
	// Defaults to "sandwich" if not provided.
	Type string `yaml:"type"`
	// FlowName is the name of the flow to execute, required for the "declarative" orchestrator.
	FlowName string `yaml:"flowName,omitempty"`
}

// Config is the top-level struct for loading a MangleKit orchestrator's
// configuration from a YAML file. It defines the components to be used for each
// stage of the pipeline and their parameters.
type Config struct {
	// Orchestrator selects and configures the workflow engine.
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	// Tools defines a map of named components that can be referenced in declarative flows.
	Tools map[string]ToolConfig `yaml:"tools"`
	// Providers holds global configurations for provider families, like API keys.
	Providers ProviderConfigs `yaml:"providers"`
	// Embedder specifies the text embedding component.
	Embedder componentCfg `yaml:"embedder"`
	// Retriever specifies the document retrieval component.
	Retriever componentCfg `yaml:"retriever"`
	// VectorStore specifies the vector database component.
	VectorStore componentCfg `yaml:"vectorStore"`
	// Reranker specifies the document reranking component.
	Reranker componentCfg `yaml:"reranker"`
	// Rules specifies the rules engine component.
	Rules componentCfg `yaml:"rules"`
	// LLM specifies the language model component.
	LLM componentCfg `yaml:"llm"`
	// TopK is the default number of documents to retrieve.
	TopK int `yaml:"topK"`
	// MaxTokens is the default maximum number of tokens for the LLM response.
	MaxTokens int `yaml:"maxTokens"`
	// FallbackThreshold is the confidence score below which a fallback is triggered.
	FallbackThreshold float64 `yaml:"fallbackThreshold"`
}

// ProviderConfigs holds the global configurations for different provider families,
// such as API keys or default models. These can be referenced by the components.
type ProviderConfigs struct {
	Google           *GoogleConfig           `yaml:"google,omitempty"`
	OpenAI           *OpenAIConfig           `yaml:"openai,omitempty"`
	Groq             *OpenAICompatibleConfig `yaml:"groq,omitempty"`
	OpenAICompatible *OpenAICompatibleConfig `yaml:"openaiCompatible,omitempty"`
	Mangle           *MangleConfig           `yaml:"mangle,omitempty"`
}

// MangleConfig holds configuration specific to the Mangle rules engine provider.
type MangleConfig struct {
	// RulePaths is a list of file paths or glob patterns for Mangle rule files (.dlog).
	RulePaths []string `yaml:"path"`
	// PreProcess is a list of transformer names to run in the 'pre' stage.
	PreProcess []string `yaml:"preProcess"`
	// PostProcess is a list of transformer names to run in the 'post' stage.
	PostProcess []string `yaml:"postProcess"`
	// DefaultConverters specifies whether to include the built-in fact converters.
	DefaultConverters bool `yaml:"defaultConverters"`
}

// GoogleConfig holds global configuration for Google providers (Genkit, Gemini).
type GoogleConfig struct {
	// APIKey is the API key for Google AI services.
	APIKey string `yaml:"apiKey,omitempty"`
	// Model is the default model identifier to use.
	Model string `yaml:"model,omitempty"`
	// PromptTemplate is a default prompt template string.
	PromptTemplate string `yaml:"promptTemplate,omitempty"`
}

// OpenAIConfig holds global configuration specific to OpenAI providers.
type OpenAIConfig struct {
	// APIKey is the API key for OpenAI services.
	APIKey string `yaml:"apiKey,omitempty"`
	// Model is the default model identifier to use.
	Model string `yaml:"model,omitempty"`
	// Dimensions is the desired embedding vector dimension (for supported models).
	Dimensions int `yaml:"dimensions,omitempty"`
	// PromptTemplate is a default prompt template string.
	PromptTemplate string `yaml:"promptTemplate,omitempty"`
}

// OpenAICompatibleConfig holds configuration for providers that use an
// OpenAI-compatible API, such as Groq.
type OpenAICompatibleConfig struct {
	// APIKey is the API key for the service.
	APIKey string `yaml:"apiKey,omitempty"`
	// BaseURL is the base URL of the API endpoint.
	BaseURL string `yaml:"baseURL,omitempty"`
	// Model is the model identifier to use.
	Model string `yaml:"model,omitempty"`
	// Dimensions is the desired embedding vector dimension.
	Dimensions int `yaml:"dimensions,omitempty"`
}

// NewBuilderFromYAML reads a YAML configuration file, parses it, and returns a
// pre-configured Builder instance. This is the recommended way to initialize
// MangleKit from a static configuration.
//
// The function supports environment variable expansion within the YAML file using
// standard shell syntax (e.g., `$VAR` or `${VAR}`).
//
// It uses a generic, reflection-based system to find and resolve all file paths
// within the configuration that are tagged with `path:"resolve"`.
//
// path is the file path to the YAML configuration file.
// It returns a pre-configured BuilderAPI instance or an error if the file
// cannot be read or parsed.
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

	// Resolve all tagged paths in the config struct relative to the config file's directory.
	configDir := filepath.Dir(path)
	if err := resolvePathsInStruct(&cfg, configDir); err != nil {
		return nil, fmt.Errorf("failed to resolve paths in config: %w", err)
	}

	builder := NewBuilder().
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
