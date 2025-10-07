package llm

// OpenAIOptions provides typed configuration for OpenAI language models.
type OpenAIOptions struct {
	// APIKey is the API key for authenticating with the OpenAI service.
	APIKey string `json:"apiKey,omitempty"`
	// Model is the identifier for the specific OpenAI model to be used (e.g., "gpt-4-turbo").
	Model string `json:"model,omitempty"`
	// PromptTemplate is an optional custom Go template string for formatting the
	// final prompt. If empty, a default template will be used.
	PromptTemplate string `json:"promptTemplate,omitempty"`
}

// GoogleOptions provides typed configuration for Google language models.
type GoogleOptions struct {
	// Model is the identifier for the specific Google model to be used (e.g., "gemini-1.5-flash").
	Model string `json:"model"`
	// PromptTemplate is an optional custom Go template string for formatting the
	// final prompt. If empty, a default template will be used.
	PromptTemplate string `json:"promptTemplate"`
}