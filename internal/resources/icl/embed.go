package icl

import (
	_ "embed"
)

//go:embed golden.dl
var goldenRules string

// GetGoldenRules returns the embedded golden rules content.
func GetGoldenRules() string {
	return goldenRules
}
