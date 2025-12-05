package util

import (
    "context"
    "github.com/duynguyendang/manglekit/core"
)

// FuncWrapper adapts a functional handler to the core.Action interface
type FuncWrapper struct {
	ActionName     string
	MetaInputType  string
	MetaOutputType string
	Fn             func(context.Context, core.Envelope) (core.Envelope, error)
}

func (f *FuncWrapper) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name:             f.ActionName,
		Type:             "generic-function",
		InputContentType: core.TypeStruct,
		InputType:        f.MetaInputType,
		OutputType:       f.MetaOutputType,
		IsDynamic:        false,
	}
}

func (f *FuncWrapper) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
    return f.Fn(ctx, env)
}
