//go:build testhooks

package noop

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
)

// NoopTool is a minimal tool implementation for testing the registry.
// It now correctly implements the core.Tool interface.
type NoopTool struct{
	ShouldError bool
}

func (n *NoopTool) Execute(ctx context.Context, execCtx *core.ExecutionContext) error {
	// For the test, we don't need to do anything complex.
	// We just need a method with the correct signature.
	if execCtx != nil && execCtx.Meta != nil {
		execCtx.Meta["noop_executed"] = true
	}
	return nil
}

// ensure it implements the core.Tool interface
var _ core.Tool = &NoopTool{}

// NoopOptions implements the ProviderOptions interface, allowing the NoopTool
// to be registered with the sdk.Register function.
type NoopOptions struct{}

func (o *NoopOptions) ProviderName() string { return "noop" }
// We'll register our test tool under the SchemaParser kind for this smoke test.
func (o *NoopOptions) ProviderKind() core.Kind { return core.KindSchemaParser }

// New creates a new NoopTool. The dependencies argument is ignored.
func New(ctx context.Context, deps any, cfg *NoopOptions) (core.Tool, error) {
	return &NoopTool{}, nil
}
