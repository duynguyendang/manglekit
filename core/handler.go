// Package core provides the central interfaces and types for the Manglekit SDK.
package core

import (
	"context"
)

// ComponentHandler defines the build logic for a specific component kind.
// Implementations are registered with the registry and are responsible for
// resolving dependencies for, building, and assigning a component of a
// specific kind to the Resolved struct.
type ComponentHandler interface {
	// Kind returns the component kind this handler is responsible for.
	Kind() Kind

	// BuildComponent contains the full logic for building one component. It
	// resolves dependencies using the builderDI, calls the factory to construct
	// the component, assigns it to the correct field in the resolved struct,
	// and returns any resource closer that needs to be tracked.
	BuildComponent(
		ctx context.Context,
		// builderDI is the dependency injection interface provided by the builder.
		// Implementations should type-assert this to a DI interface that exposes
		// methods for retrieving other built components by name.
		builderDI any,
		// factory is the component's constructor function, retrieved from the registry.
		// Implementations should type-assert this to a known factory type.
		factory any,
		// resolved is the target struct where the built component will be stored.
		resolved *Resolved,
		// cfg is the component's specific options struct.
		cfg ProviderOptions,
		// name is the registered name of the component to be built.
		name string,
	) (ResourceCloser, error)
}
