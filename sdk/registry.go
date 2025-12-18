package sdk

import (
	"context"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
)

// ProviderFactory defines the constructor signature for creating an action from config.
// Strict Mode: Returns a ClientOption to configure the client directly.
type ProviderFactory func(opts map[string]any) (ClientOption, error)

var (
	registryMu       sync.RWMutex
	providerRegistry = make(map[string]ProviderFactory)
)

// RegisterProvider allows external packages (plugins) to register their factories.
// Example: sdk.RegisterProvider("google", GoogleFactory)
func RegisterProvider(name string, factory ProviderFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	providerRegistry[name] = factory
}

// Provider retrieves a registered factory.
func Provider(name string) (ProviderFactory, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	factory, ok := providerRegistry[name]
	if !ok {
		return nil, fmt.Errorf("provider '%s' is not registered", name)
	}
	return factory, nil
}

// WithProviderConfig creates a ClientOption that initializes a provider from configuration.
// It looks up the factory, creates the ClientOption, and applies it.
func WithProviderConfig(name string, cfg config.ActionConfig) ClientOption {
	return func(c *Client) error {
		registryMu.RLock()
		factory, ok := providerRegistry[cfg.Provider]
		registryMu.RUnlock()

		if !ok {
			// If fail_on_startup is true, we error out.
			if cfg.FailOnStartup {
				return fmt.Errorf("provider factory '%s' not found for action '%s'", cfg.Provider, name)
			}
			// Otherwise log warning and skip
			c.logger.Warn("provider factory not found", "provider", cfg.Provider, "action", name)
			return nil
		}

		// Create the option using the factory.
		opt, err := factory(cfg.Options)
		if err != nil {
			if cfg.FailOnStartup {
				return fmt.Errorf("failed to create option for action '%s': %w", name, err)
			}
			c.logger.Warn("failed to create provider option", "name", name, "error", err)
			return nil
		}

		// Apply the option to the client
		if err := opt(c); err != nil {
			if cfg.FailOnStartup {
				return fmt.Errorf("failed to apply provider option for action '%s': %w", name, err)
			}
			c.logger.Warn("failed to apply provider option", "name", name, "error", err)
			return nil
		}

		return nil
	}
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

// MemoryProvider retrieves a registered memory factory.
func MemoryProvider(name string) (MemoryFactory, error) {
	memoryRegistryMu.RLock()
	defer memoryRegistryMu.RUnlock()
	factory, ok := memoryRegistry[name]
	if !ok {
		return nil, fmt.Errorf("memory provider '%s' is not registered", name)
	}
	return factory, nil
}
