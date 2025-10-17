package llm

import "github.com/duynguyendang/manglekit/core"

// OpenAIOptions provides typed configuration for OpenAI and compatible language
// models (e.g., Groq). It specifies the model to use and how to authenticate.
type OpenAIOptions struct {
	// APIKey is the API key for authenticating with the OpenAI or a compatible service.
	// If not set here, it is often read from an environment variable (e.g., OPENAI_API_KEY).
	APIKey string `json:"apiKey,omitempty"`
	// Model is the identifier for the specific model to be used for completions,
	// for example, "gpt-4-turbo" or "llama3-8b-8192".
	Model string `json:"model,omitempty"`
	// PromptTemplate is an optional custom Go template string for formatting the
	// final prompt that is sent to the LLM. If this is empty, a default
	// prompt template will be used by the client.
	PromptTemplate string `json:"promptTemplate,omitempty"`
	// Temperature controls the randomness of the model's output.
	Temperature float32 `json:"temperature,omitempty"`
	// MaxOutputTokens is the maximum number of tokens to generate in the response.
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

func (o OpenAIOptions) ProviderName() string { return "openai" }
func (o OpenAIOptions) ProviderKind() core.Kind   { return core.KindLLM }

// GroqOptions is an alias for OpenAIOptions, but it identifies itself as "groq".
type GroqOptions struct {
	OpenAIOptions
}

func (o GroqOptions) ProviderName() string { return "groq" }

// GoogleOptions provides typed configuration for Google language models,
// such as those in the Gemini family.
type GoogleOptions struct {
	// Model is the identifier for the specific Google model to be used for
	// completions, for example, "gemini-1.5-flash".
	Model string `json:"model"`
	// PromptTemplate is an optional custom Go template string for formatting the
	// final prompt that is sent to the LLM. If this is empty, a default
	// prompt template will be used by the client.
	PromptTemplate string `json:"promptTemplate"`
	// Temperature controls the randomness of the model's output.
	Temperature float32 `json:"temperature,omitempty"`
	// MaxOutputTokens is the maximum number of tokens to generate in the response.
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

func (o GoogleOptions) ProviderName() string { return "google" }
func (o GoogleOptions) ProviderKind() core.Kind   { return core.KindLLM }
