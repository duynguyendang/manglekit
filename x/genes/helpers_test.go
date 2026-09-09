package genes

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// staticAction is a trivial supervised-capable core.Action for tests.
type staticAction struct{}

func (staticAction) Execute(_ context.Context, in core.Envelope) (core.Envelope, error) {
	out := core.Envelope{Payload: "ok: " + sprint(in.Payload)}
	return out, nil
}

func (staticAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "write_doc"}
}

func sprint(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmtSprint(v)
}

// newClient is a thin wrapper kept in one place for readability.
func newClient(ctx context.Context) (*sdk.Client, error) {
	return sdk.NewClient(ctx)
}

func fmtSprint(v any) string { return fmt.Sprint(v) }
