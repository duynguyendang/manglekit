package core

import "context"

// TextGenerator is the clean interface for LLM text generation.
// Core does NOT know about Genkit or HTTP requests.
type TextGenerator interface {
	Complete(ctx context.Context, prompt string) (string, error)
}
