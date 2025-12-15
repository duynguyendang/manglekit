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

// MemoryFactory defines the constructor for memory providers.
type MemoryFactory func(ctx context.Context, cfg config.MemoryConfig) (core.AgentMemory, error)

var (
	memoryRegistryMu sync.RWMutex
	memoryRegistry   = make(map[string]MemoryFactory)
)

// RegisterMemoryProvider allows plugins to register memory backends (e.g., "qdrant", "simple").
func RegisterMemoryProvider(name string, factory MemoryFactory) {
	memoryRegistryMu.Lock()
	defer memoryRegistryMu.Unlock()
	memoryRegistry[name] = factory
}

// GetMemoryProvider retrieves a registered memory factory.
func GetMemoryProvider(name string) (MemoryFactory, error) {
	memoryRegistryMu.RLock()
	defer memoryRegistryMu.RUnlock()
	factory, ok := memoryRegistry[name]
	if !ok {
		return nil, fmt.Errorf("memory provider '%s' is not registered", name)
	}
	return factory, nil
}
