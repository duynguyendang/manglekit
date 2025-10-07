package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

// DefaultRAGTemplate provides a standard prompt structure for Retrieval-Augmented
// Generation (RAG). It instructs the model to answer a user's question based
// solely on the provided context.
const DefaultRAGTemplate = `System: You are a helpful AI assistant. Your task is to answer the user's question based ONLY on the provided context. If the context does not contain the answer, say "I do not have enough information to answer this question."

Context:
---
{{- range .Context }}
{{ . }}
---
{{- end }}

User Question: {{ .Query }}

Answer:`

// PromptBuilder is a thread-safe utility responsible for compiling and executing
// Go templates to generate final prompt strings. It improves performance by
// caching compiled templates in memory.
type PromptBuilder struct {
	templateCache      map[string]*template.Template
	mu                 sync.RWMutex
	defaultTemplateStr string
}

// NewPromptBuilder creates and returns a new PromptBuilder.
//
// defaultTemplate is the template string to be used as a fallback when a specific
// user template is not provided to the Build method.
func NewPromptBuilder(defaultTemplate string) *PromptBuilder {
	return &PromptBuilder{
		templateCache:      make(map[string]*template.Template),
		defaultTemplateStr: defaultTemplate,
	}
}

// Build generates a final prompt string by executing a Go template with the
// provided data. It uses the userTemplate if one is given; otherwise, it falls
// back to the default template configured on the PromptBuilder.
//
// For efficiency, successfully compiled templates are cached in a thread-safe
// manner to avoid repeated parsing. The builder also injects several useful
// functions into the template:
//   - `toJSON`: Marshals a value to a JSON string.
//   - `join`: Joins a string slice with a separator.
//   - `truncate`: Truncates a string to a maximum length.
//
// userTemplate is the template string to execute. If empty, the default is used.
// data is a map of data to be made available within the template.
// It returns the executed prompt as a string, or an error if template parsing
// or execution fails.
func (pb *PromptBuilder) Build(userTemplate string, data map[string]any) (string, error) {
	templateToUse := userTemplate
	if strings.TrimSpace(templateToUse) == "" {
		templateToUse = pb.defaultTemplateStr
	}

	// 1. Check cache with a read lock
	pb.mu.RLock()
	tmpl, found := pb.templateCache[templateToUse]
	pb.mu.RUnlock()

	if !found {
		// 2. If not found, acquire a write lock to compile and cache
		pb.mu.Lock()
		// Double-check, as another goroutine might have acquired the lock first
		tmpl, found = pb.templateCache[templateToUse]
		if !found {
			var err error
			funcMap := template.FuncMap{
				"toJSON": func(v any) (string, error) {
					b, err := json.Marshal(v)
					if err != nil {
						return "", err
					}
					return string(b), nil
				},
				"join": strings.Join,
				"truncate": func(s string, max int) string {
					if len(s) > max {
						return s[:max]
					}
					return s
				},
			}

			tmpl, err = template.New("rag").Funcs(funcMap).Parse(templateToUse)
			if err != nil {
				pb.mu.Unlock()
				return "", fmt.Errorf("failed to parse prompt template: %w", err)
			}
			pb.templateCache[templateToUse] = tmpl
		}
		pb.mu.Unlock()
	}

	// 3. Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template: %w", err)
	}

	return buf.String(), nil
}