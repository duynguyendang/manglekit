package sdk

// WithMetadataMap injects a map of custom key-value pairs into the execution envelope's metadata.
func WithMetadataMap(meta map[string]string) ExecuteOption {
	return func(p *ExecutionParams) {
		if p.Metadata == nil {
			p.Metadata = make(map[string]string)
		}
		for k, v := range meta {
			p.Metadata[k] = v
		}
	}
}
