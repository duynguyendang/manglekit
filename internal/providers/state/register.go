package state

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/internal/providers/state/redis"
)

// Register registers all state providers and the state provider kind handler.
func Register(r *manglekit.Registry) {
	inmemory.Register(r)
	redis.Register(r)
	r.RegisterHandler(&Handler{})
}
