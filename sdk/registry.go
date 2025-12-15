package sdk

import (
	"context"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
)

// ProviderFactory defines the constructor signature for creating an action from config.
type ProviderFactory func(ctx context.Context, name string, cfg config.ActionConfig) (core.Action, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]ProviderFactory)
)

// RegisterProvider allows external packages (plugins) to register their factories.
// Example: sdk.RegisterProvider("google", GoogleFactory)
func RegisterProvider(name string, factory ProviderFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// GetProvider retrieves a registered factory.
func GetProvider(name string) (ProviderFactory, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("provider '%s' is not registered", name)
	}
	return factory, nil
}
