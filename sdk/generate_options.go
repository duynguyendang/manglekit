package sdk

import "github.com/duynguyendang/manglekit/core"

// WithTemperature sets the sampling temperature.
func WithTemperature(t float64) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.Temperature = t
	}
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(n int) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.MaxTokens = n
	}
}

// WithTopP sets the nucleus sampling probability.
func WithTopP(p float64) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.TopP = p
	}
}

// WithStopSequences sets the stop sequences.
func WithStopSequences(seqs []string) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.StopSequences = seqs
	}
}

// WithModel sets the model to use.
func WithModel(m string) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.Model = m
	}
}

// WithJSONMode enables JSON mode.
func WithJSONMode(enabled bool) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.JSONMode = enabled
	}
}

// WithStructuredOutput sets the output type for structured generation.
func WithStructuredOutput(schema any) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.OutputType = schema
	}
}
