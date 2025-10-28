package openai_llm

import (
"github.com/duynguyendang/manglekit/core"
"github.com/duynguyendang/manglekit/internal/providers/llm" // Assumes generic handler is here
)

// Options struct for YAML unmarshaling
type Options struct {
core.ComponentOptions `mapstructure:",squash"`
Model string `yaml:"model"`
APIKey string `yaml:"api_key"`
}

// Register populates the global Registry
func Register() {
core.MustRegister(&core.Registration{
Kind: core.KindLLM, //
Type: "openai",
OptionsType: &Options{},
NewHandler: llm.NewHandler, // Generic Kind-level handler
NewFactory: func() core.Factory { return &Factory{} }, // Specific provider factory
})
}
