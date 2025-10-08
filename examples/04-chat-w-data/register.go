package main

import "github.com/duynguyendang/manglekit"

// newCustomConverter is the constructor for our query-to-fact converter.
func newCustomConverter(params map[string]any) (any, error) {
	return &CustomFactConverter{}, nil
}

// init registers our custom converter with the Manglekit component registry.
func init() {
	manglekit.Register("custom-converter", newCustomConverter)
}