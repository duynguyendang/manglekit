package sdk

import "github.com/duynguyendang/manglekit"

var globalRegistry = manglekit.NewRegistry()

// GlobalRegistry returns the singleton instance of the Manglekit registry.
func GlobalRegistry() *manglekit.Registry {
	return globalRegistry
}