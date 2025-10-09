package genkitshim

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	realgenkit "github.com/firebase/genkit/go/genkit"
)

// Genkit re-exports the core Genkit runtime type so callers can interact with the
// underlying registry returned by Init.
type Genkit = realgenkit.Genkit

// DefineTool delegates to the upstream Genkit DefineTool helper.
func DefineTool[In, Out any](g *realgenkit.Genkit, name, description string, fn ai.ToolFunc[In, Out]) ai.Tool {
	return realgenkit.DefineTool(g, name, description, fn)
}

// DefineFlow delegates to the upstream Genkit DefineFlow helper.
func DefineFlow[In, Out any](g *realgenkit.Genkit, name string, fn core.Func[In, Out]) *core.Flow[In, Out, struct{}] {
	return realgenkit.DefineFlow(g, name, fn)
}

// Init wraps genkit.Init so callers can configure the Genkit runtime.
func Init(ctx context.Context, opts ...realgenkit.GenkitOption) *realgenkit.Genkit {
	return realgenkit.Init(ctx, opts...)
}

// RunTool executes a registered tool inside a traced Genkit step and casts the
// output to the requested type. This helper mirrors the ergonomics of
// genkit.Run but focuses on tool execution to keep flow code declarative.
func RunTool[In, Out any](ctx context.Context, stepName string, tool ai.Tool, input In) (Out, error) {
	return realgenkit.Run[Out](ctx, stepName, func() (Out, error) {
		raw, err := tool.RunRaw(ctx, input)
		if err != nil {
			var zero Out
			return zero, err
		}
		typed, ok := raw.(Out)
		if !ok {
			var zero Out
			return zero, fmt.Errorf("genkit: tool %q returned %T, expected %T", tool.Name(), raw, zero)
		}
		return typed, nil
	})
}
