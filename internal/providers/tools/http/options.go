package http

import "github.com/duynguyendang/manglekit/core"

const (
	ProviderName = "http"
)

type Options struct {
	Endpoint string            `yaml:"endpoint"`
	Method   string            `yaml:"method"`
	Headers  map[string]string `yaml:"headers"`
}

func (o Options) ProviderName() string {
	return ProviderName
}

func (o Options) ProviderKind() core.Kind {
	return core.KindTool
}
