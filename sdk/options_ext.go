package sdk

import "fmt"

// WithMetadataMap injects a map of custom key-value pairs into the execution envelope's metadata.
func WithMetadataMap(meta map[string]any) ExecuteOption {
	return func(p *ExecutionParams) {
		if p.Metadata == nil {
			p.Metadata = make(map[string]string)
		}
		for k, v := range meta {
			if s, ok := v.(string); ok {
				p.Metadata[k] = s
			} else {
				p.Metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}
}
