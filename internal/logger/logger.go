package logger

import (
	"go.uber.org/zap"
)

// New creates a new zap logger.
func New() (*zap.Logger, error) {
	// For now, we'll use a simple production logger.
	// This can be extended to support different configurations (e.g., development).
	return zap.NewProduction()
}
