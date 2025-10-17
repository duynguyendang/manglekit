package state

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/internal/providers/state/redis"
	"github.com/duynguyendang/manglekit/sdk"
)

func init() {
	Register(sdk.GlobalRegistry())
}

// Register registers all state providers.
func Register(r *manglekit.Registry) {
	inmemory.Register(r)
	redis.Register(r)
}