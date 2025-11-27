package registry

import (
	"sync"

	"github.com/duynguyendang/manglekit/v1"
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

// Global returns the process-wide shared Registry instance.
//
// ✓ THREAD SAFE: This function is safe to call from any goroutine.
//
// Thread-Safety Guarantee:
// The returned *Registry is protected by its internal sync.RWMutex (added in registry.go).
// All Registry methods (Get, Register, GetHandler, RegisterHandler) use appropriate
// locks (RLock for reads, Lock for writes), ensuring concurrent access is safe.
//
// Usage Pattern:
//
//	// Safe to call from any goroutine
//	registry := registry.Global()
//	factory, err := registry.Get(ctx, kind, name)  // Uses RLock internally
//	registry.RegisterHandler(handler)              // Uses Lock internally
//
// Initialization:
// The global registry is initialized in init() and contains all standard providers
// registered via their init() functions (which call registry.Global().Register(...)).
func Global() *manglekit.Registry {
	return globalRegistry
}
