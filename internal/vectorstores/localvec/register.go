package localvec

import (
"github.com/duynguyendang/manglekit/core"
"github.com/duynguyendang/manglekit/internal/vectorstores" // Assumes generic handler
)

// Options struct for YAML unmarshaling
type Options struct {
core.ComponentOptions `mapstructure:",squash"`
Path string `yaml:"path"`
}

// Register populates the global Registry
func Register() {
core.MustRegister(&core.Registration{
Kind: core.KindVectorStore, //
Type: "localvec",
OptionsType: &Options{},
NewHandler: vectorstores.NewHandler,
NewFactory: func() core.Factory { return &Factory{} },
})
}
