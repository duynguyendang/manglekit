package ooda

import (
	"context"
	"fmt"
	"sync"
)

// ToolFunc is the function signature for executable tools.
// Each tool receives a context and arguments, and returns a result or error.
type ToolFunc func(ctx context.Context, args map[string]interface{}) (string, error)

// Registry maps action names to executable Go functions.
// This decouples Manglekit's logical decisions from their Go implementations.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]ToolFunc
}

// NewRegistry creates a new empty action registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolFunc),
	}
}

// Register adds a new tool to the registry.
// Returns an error if a tool with the same name is already registered.
func (r *Registry) Register(name string, fn ToolFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool '%s' already registered", name)
	}

	r.tools[name] = fn
	return nil
}

// MustRegister registers a tool and panics on error.
// Useful for initialization code.
func (r *Registry) MustRegister(name string, fn ToolFunc) {
	if err := r.Register(name, fn); err != nil {
		panic(err)
	}
}

// Unregister removes a tool from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Get retrieves a tool by name.
// Returns nil if not found.
func (r *Registry) Get(name string) ToolFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// Has checks if a tool is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.tools[name]
	return exists
}

// List returns all registered tool names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute looks up and executes a tool by name.
// Returns an error if the tool is not found.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	fn := r.Get(name)
	if fn == nil {
		return "", fmt.Errorf("unknown action: %s (registered: %v)", name, r.List())
	}
	return fn(ctx, args)
}

// Dispatcher executes actions from the registry based on Decision.
type Dispatcher struct {
	registry *Registry
	fallback ToolFunc // Called when action not found
}

// NewDispatcher creates a new dispatcher with the given registry.
func NewDispatcher(registry *Registry) *Dispatcher {
	return &Dispatcher{
		registry: registry,
	}
}

// WithFallback sets a fallback function to be called when an action is not found.
func (d *Dispatcher) WithFallback(fn ToolFunc) *Dispatcher {
	d.fallback = fn
	return d
}

// SafeStop is a mandatory safety fallback tool that is called when the system
// enters an undefined or high-risk state.
var SafeStop ToolFunc = func(ctx context.Context, args map[string]interface{}) (string, error) {
	return "STOPPED: SafeStop invoked due to undefined action or high-risk state", nil
}

// Dispatch executes an action based on the action name and arguments.
// If the action is not found, it logs a "Sovereign/Capability Mismatch" and calls SafeStop.
func (d *Dispatcher) Dispatch(ctx context.Context, actionName string, args map[string]interface{}) (string, error) {
	// Try to find the tool
	fn := d.registry.Get(actionName)
	if fn != nil {
		return fn(ctx, args)
	}

	// Sovereign Violation: Action not found in registry
	errMsg := fmt.Sprintf("SOVEREIGN VIOLATION: action '%s' not found in registry - capability mismatch", actionName)
	fmt.Printf("⚠️ %s\n", errMsg)
	fmt.Printf("   Available actions: %v\n", d.registry.List())
	fmt.Printf("   Requested args: %v\n", args)

	// Call SafeStop as mandatory safety fallback
	if SafeStop != nil {
		safeStopResult, safeStopErr := SafeStop(ctx, map[string]interface{}{
			"reason":         "sovereign_violation",
			"action":         actionName,
			"args":           args,
			"original_error": errMsg,
		})
		if safeStopErr != nil {
			return "", fmt.Errorf("%s; SafeStop failed: %w", errMsg, safeStopErr)
		}
		return safeStopResult, fmt.Errorf("%s", errMsg)
	}

	// No SafeStop - return error
	return "", fmt.Errorf("%s", errMsg)
}

// ActionEnvelope contains the action to be executed and its parameters.
type ActionEnvelope struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// NewActionEnvelope creates a new ActionEnvelope.
func NewActionEnvelope(name string, args map[string]interface{}) *ActionEnvelope {
	if args == nil {
		args = make(map[string]interface{})
	}
	return &ActionEnvelope{
		Name:      name,
		Arguments: args,
	}
}

// String returns a string representation of the action envelope.
func (e *ActionEnvelope) String() string {
	return fmt.Sprintf("ActionEnvelope{Name: %s, Arguments: %v}", e.Name, e.Arguments)
}
