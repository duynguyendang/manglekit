package ai

import (
	"context"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/tracing"
	"github.com/firebase/genkit/go/genkit"
)

var (
	globalGenkit *genkit.Genkit
	initOnce     sync.Once
)

// GetGenkit returns the global Genkit instance, initializing it if necessary.
// This allows other packages to register models/tools.
func GetGenkit(ctx context.Context) *genkit.Genkit {
	initOnce.Do(func() {
		// Initialize without plugins initially
		globalGenkit = genkit.Init(ctx)
	})
	return globalGenkit
}

// NewGenkitAction creates a new core.Action backed by a Genkit model.
// It ensures the Genkit runtime is initialized and looks up the model by name.
func NewGenkitAction(ctx context.Context, modelName string) (core.Action, error) {
	g := GetGenkit(ctx)

	// Lookup model using genkit.LookupModel
	// Note: We use the full name (e.g., "openai/gpt-4o") directly.
	model := genkit.LookupModel(g, modelName)
	if model == nil {
		// Try parsing if lookup failed directly (in case some registry logic differs)
		// But usually full name is key.

		// Fallback debug info
		return nil, fmt.Errorf("genkit model not found: %s", modelName)
	}

	// Create adapter
	adapter := NewGenkitAdapter(model, g)

	// Wrap in LLMAction
	// We use modelName as the action name
	// Note: We might want a cleaner name? Using modelName is fine.
	return NewLLMAction(modelName, adapter)
}

// GenkitTextGenerator is the exported interface for adapters backed by a
// Genkit model. It extends core.TextGenerator with access to the underlying
// Genkit model and runtime for native features (structured output, middleware).
type GenkitTextGenerator interface {
	core.TextGenerator
	GenkitModel() ai.Model
	GenkitInstance() *genkit.Genkit
}

// genkitAdapter adapts the Firebase Genkit ai.Model interface to the Manglekit core.TextGenerator interface.
type genkitAdapter struct {
	model ai.Model
	gk    *genkit.Genkit
}

// NewGenkitAdapter creates a new adapter from a pre-initialized Genkit model.
//
// Parameters:
//   - model: The Genkit model instance.
//   - gk: The Genkit runtime instance.
//
// Returns:
//   - A core.TextGenerator implementation (also satisfies GenkitTextGenerator).
func NewGenkitAdapter(model ai.Model, gk *genkit.Genkit) core.TextGenerator {
	return &genkitAdapter{
		model: model,
		gk:    gk,
	}
}

// GenkitModel returns the underlying Genkit model.
func (g *genkitAdapter) GenkitModel() ai.Model { return g.model }

// GenkitInstance returns the underlying Genkit runtime.
func (g *genkitAdapter) GenkitInstance() *genkit.Genkit { return g.gk }

// Complete generates text using the underlying Genkit model.
func (g *genkitAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	resp, err := g.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// Generate implements the core.TextGenerator interface using Genkit.
// Supports genkit 1.7+ middleware via WithMiddleware, WithRetry, WithFallback, etc.
func (g *genkitAdapter) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	// Initialize Config
	cfg := &core.GenerationConfig{
		Temperature: 0.7, // Default
	}
	for _, opt := range opts {
		opt(cfg)
	}

	ctx = injectTelemetryLabels(ctx)

	var messages []*ai.Message

	// Dynamic Prompt Configuration
	facts := core.ContextFacts(ctx)
	systemPrompt := ""
	if facts != nil {
		if val, ok := facts[core.PrefixPromptConfig+"tone"]; ok {
			systemPrompt += "\n[INSTRUCTION]: Maintain a " + val + " tone."
		}
		if val, ok := facts[core.PrefixPromptConfig+"strategy"]; ok && val == "cot" {
			systemPrompt += "\n[STRATEGY]: Think step-by-step."
		}
	}

	if systemPrompt != "" {
		messages = append(messages, ai.NewSystemMessage(ai.NewTextPart(systemPrompt)))
	}

	messages = append(messages, ai.NewUserMessage(ai.NewTextPart(prompt)))

	// Build generation options
	genOpts := []ai.GenerateOption{
		ai.WithModel(g.model),
		ai.WithMessages(messages...),
	}

	// Build a single generation config with all values
	genConfig := ai.GenerationCommonConfig{}
	hasConfig := false

	if cfg.Temperature != 0 {
		genConfig.Temperature = cfg.Temperature
		hasConfig = true
	}
	if cfg.MaxTokens != 0 {
		genConfig.MaxOutputTokens = cfg.MaxTokens
		hasConfig = true
	}
	if cfg.TopP != 0 {
		genConfig.TopP = cfg.TopP
		hasConfig = true
	}
	if len(cfg.StopSequences) > 0 {
		genConfig.StopSequences = cfg.StopSequences
		hasConfig = true
	}

	// Apply config once if any values were set
	if hasConfig {
		genOpts = append(genOpts, ai.WithConfig(genConfig))
	}

	// Handle Output / JSON Mode
	if cfg.OutputType != nil || cfg.JSONMode {
		genOpts = append(genOpts, ai.WithOutputFormat("json"))
	}

	// Apply middleware options
	mwCfg := extractMiddlewareConfig(cfg)
	if mwCfg != nil {
		mwOpts := buildMiddlewareOptions(mwCfg)
		genOpts = append(genOpts, mwOpts...)
	}

	// Use high-level genkit.Generate to enable middleware
	resp, err := genkit.Generate(ctx, g.gk, genOpts...)
	if err != nil {
		return nil, err
	}

	// Extract token usage if available
	usage := make(map[string]int)
	if resp.Usage != nil {
		usage["prompt"] = int(resp.Usage.InputTokens)
		usage["completion"] = int(resp.Usage.OutputTokens)
		usage["total"] = int(resp.Usage.TotalTokens)
	}

	return &core.LLMResponse{
		Text:  resp.Text(),
		Usage: usage,
	}, nil
}

// Stream implements the core.TextGenerator interface.
// It uses Genkit's streaming callback to deliver incremental chunks.
func (g *genkitAdapter) Stream(ctx context.Context, prompt string) (<-chan core.StreamChunk, error) {
	ctx = injectTelemetryLabels(ctx)
	ch := make(chan core.StreamChunk, 8)

	streamCb := func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		if t := chunk.Text(); t != "" {
			select {
			case ch <- core.StreamChunk{Text: t}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}

	genOpts := []ai.GenerateOption{
		ai.WithModel(g.model),
		ai.WithMessages(ai.NewUserMessage(ai.NewTextPart(prompt))),
		ai.WithStreaming(streamCb),
	}

	go func() {
		defer close(ch)
		_, err := genkit.Generate(ctx, g.gk, genOpts...)
		if err != nil {
			select {
			case ch <- core.StreamChunk{Err: err}:
			case <-ctx.Done():
			}
		}
	}()

	return ch, nil
}

// injectTelemetryLabels propagates manglekit tracing metadata (TraceID, SessionID)
// into Genkit's telemetry label system so Dev UI spans are correlated with OODA frames.
func injectTelemetryLabels(ctx context.Context) context.Context {
	facts := core.ContextFacts(ctx)
	if facts == nil {
		return ctx
	}

	labels := tracing.TelemetryLabelsFromContext(ctx)
	if labels == nil {
		labels = make(map[string]string)
	}

	if v, ok := facts[core.KeyTraceID]; ok && v != "" {
		labels["manglekit.trace_id"] = v
	}
	if v, ok := facts["session_id"]; ok && v != "" {
		labels["manglekit.session_id"] = v
	}
	if v, ok := facts["parent_id"]; ok && v != "" {
		labels["manglekit.parent_id"] = v
	}

	if len(labels) > 0 {
		ctx = tracing.WithTelemetryLabels(ctx, labels)
	}
	return ctx
}
