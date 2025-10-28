package dense

import (
"github.com/duynguyendang/manglekit/core"
"github.com/duynguyendang/manglekit/internal/providers/retrievers" // Assumes generic handler
)

// Options struct declares dependencies by name
type Options struct {
core.ComponentOptions `mapstructure:",squash"`
VectorStore string `yaml:"vectorstore"`
Embedder string `yaml:"embedder"`
TopK int `yaml:"top_k"`
}

// Register populates the global Registry
func Register() {
core.MustRegister(&core.Registration{
Kind: core.KindRetriever, //
Type: "dense",
OptionsType: &Options{},
NewHandler: retrievers.NewHandler,
NewFactory: func() core.Factory { return &Factory{} },
})
}
