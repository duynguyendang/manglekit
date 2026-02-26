package sdk

import (
	"context"

	"github.com/duynguyendang/manglekit-wip/core"
)

// Action returns a handle to a registered action that implements the core.Action interface.
// This allows registered actions (like LLMs) to be injected into other components
// that expect a core.Action dependency.
func (c *Client) Action(name string) core.Action {
	return &actionProxy{
		client: c,
		name:   name,
	}
}

// actionProxy adapts the Client.ExecuteByName functionality to the core.Action interface.
type actionProxy struct {
	client *Client
	name   string
}

func (p *actionProxy) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	// We use ExecuteByName to leverage the full client capabilities (retries, steering, etc.)
	// We pass the envelope's payload and metadata.
	return p.client.ExecuteByName(ctx, p.name, env.Payload, WithMetadataMap(env.Metadata))
}

func (p *actionProxy) Metadata() core.ActionMetadata {
	// Return the proxy signature info
	return core.ActionMetadata{
		Name: p.name,
		Type: "proxy",
	}
}
