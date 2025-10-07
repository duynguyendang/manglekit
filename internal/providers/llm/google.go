package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

func init() {
	manglekit.RegisterLLM("google", NewGoogle)
}

// googleClient implements the llm.Client interface for Google's generative models,
// using the Genkit framework as an abstraction layer.
type googleClient struct {
	model          ai.Model
	promptTemplate string
	promptBuilder  *llm.PromptBuilder
}

// NewGoogle creates a new Google llm.Client. It is the constructor function
// registered with the MangleKit registry for the "google" LLM provider. It
// requires a configured Genkit instance, which is handled by the builder.
//
// opts provides configuration such as the model name and an optional prompt template.
// g is the initialized Genkit instance that provides access to the model.
// It returns a configured llm.Client or an error if dependencies are missing.
func NewGoogle(opts llm.GoogleOptions, g *genkit.Genkit) (llm.Client, error) {
	if g == nil {
		return nil, fmt.Errorf("genkit client is required for google llm")
	}
	if opts.Model == "" {
		return nil, fmt.Errorf("Model is required for google llm")
	}

	return &googleClient{
		model:          googlegenai.GoogleAIModel(g, opts.Model),
		promptTemplate: opts.PromptTemplate,
		promptBuilder:  llm.NewPromptBuilder(llm.DefaultRAGTemplate),
	}, nil
}

// Complete generates a response from the configured Google model. It first uses
// the PromptBuilder to construct the final prompt from the request's context and
// data, then calls the model via the Genkit framework.
// This method satisfies the llm.Client interface.
//
// req is the request containing the prompt, context, and other parameters.
// It returns an llm.Response with the generated text and token usage, or an
// error if prompt building or model generation fails.
func (c *googleClient) Complete(req llm.Request) (llm.Response, error) {
	// Prepare the data for the template.
	data := map[string]any{
		"Context": req.Context,
		"Query":   req.Prompt,
	}
	// Merge any dynamic data from the request, overwriting defaults if needed.
	for k, v := range req.Data {
		data[k] = v
	}

	prompt, err := c.promptBuilder.Build(c.promptTemplate, data)
	if err != nil {
		return llm.Response{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	var finalAnswer strings.Builder
	var usage map[string]int

	genkitReq := ai.NewModelRequest(nil, ai.NewUserMessage(ai.NewTextPart(prompt)))
	ctx := context.Background()

	res, err := c.model.Generate(ctx, genkitReq, nil)
	if err != nil {
		return llm.Response{}, fmt.Errorf("failed to generate response from google: %w", err)
	}

	if res.Message != nil {
		finalAnswer.WriteString(res.Message.Text())
	}

	if res.Usage != nil {
		usage = map[string]int{
			"prompt":     res.Usage.InputTokens,
			"completion": res.Usage.OutputTokens,
		}
	}

	return llm.Response{
		Text:  finalAnswer.String(),
		Usage: usage,
	}, nil
}