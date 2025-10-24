package registry

import (
	"sync"

	"github.com/duynguyendang/manglekit"
)

var (
	globalRegistry *manglekit.Registry
	mu             sync.Mutex
)

func init() {
	globalRegistry = manglekit.NewRegistry()
}

func resetLocked() {
	mu.Lock()
	defer mu.Unlock()
	globalRegistry = manglekit.NewRegistry()
}

func Global() *manglekit.Registry {
	return globalRegistry
}
