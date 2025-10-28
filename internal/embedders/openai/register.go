package openai_embedder

import (
"github.com/duynguyendang/manglekit/core"
"github.com/duynguyendang/manglekit/internal/embedders" // Assumes generic handler is here
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
Kind: core.KindEmbedder, //
Type: "openai",
OptionsType: &Options{},
NewHandler: embedders.NewHandler,
NewFactory: func() core.Factory { return &Factory{} },
})
}
