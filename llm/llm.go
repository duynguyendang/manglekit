package llm

// Request encapsulates the input for a large language model (LLM) completion request.
type Request struct {
	// Prompt is the template or instruction for the LLM.
	Prompt string
	// Context is a slice of textual passages (e.g., from a retriever) to be
	// included in the final prompt to ground the LLM's response.
	Context []string
	// MaxTokens is the maximum number of tokens to generate in the response.
	MaxTokens int
	// Data is a map of arbitrary key-value data that can be used to populate
	// a dynamic prompt template.
	Data map[string]any
}

// Response represents the output from an LLM completion.
type Response struct {
	// Text is the generated text content.
	Text string
	// Usage is a map containing token usage statistics, e.g.,
	// {"prompt": 100, "completion": 250}.
	Usage map[string]int
}

// Client defines the interface for a large language model client.
type Client interface {
	// Complete takes a Request and returns a Response from the LLM.
	//
	// req is the LLM completion request.
	// It returns a Response containing the generated text and usage data,
	// or an error if the operation fails.
	Complete(req Request) (Response, error)
}