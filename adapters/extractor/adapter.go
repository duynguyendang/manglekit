package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/util/schema"
)

// ExtractorAction extracts structured data from text using an LLM.
type ExtractorAction struct {
	name      string
	llm       core.Action
	target    reflect.Type // Store reflect.Type to create new instances
	schemaStr string
}

const promptTemplate = `SYSTEM: You are a precise Data Extraction Agent.
Your goal is to extract information from the user text to match the following JSON Schema:
{{ .SchemaStr }}

Return ONLY the raw JSON object. No markdown, no explanations.

USER: {{ .InputText }}`

// New creates a new ExtractorAction.
// name: The name of this action.
// llm: The LLM Action to use for extraction.
// targetStruct: An instance of the struct (or pointer to struct) to extract into.
// Returns an error if schema generation fails.
func New(name string, llm core.Action, targetStruct any) (*ExtractorAction, error) {
	s, err := schema.Generate(targetStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema for target struct: %w", err)
	}

	return &ExtractorAction{
		name:      name,
		llm:       llm,
		target:    reflect.TypeOf(targetStruct),
		schemaStr: s,
	}, nil
}

// Execute performs the extraction.
// Input Payload: string (the text to extract from).
// Output Payload: struct (the extracted struct).
func (e *ExtractorAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	inputText, ok := input.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("input payload must be string, got %T", input.Payload)
	}

	// Construct Prompt
	tmpl, err := template.New("prompt").Parse(promptTemplate)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var promptBuilder strings.Builder
	data := struct {
		SchemaStr string
		InputText string
	}{
		SchemaStr: e.schemaStr,
		InputText: inputText,
	}

	if err := tmpl.Execute(&promptBuilder, data); err != nil {
		return core.Envelope{}, fmt.Errorf("failed to execute prompt template: %w", err)
	}

	promptEnv := core.NewEnvelope(promptBuilder.String())
	// Propagate metadata if necessary (e.g. tracing IDs)
	// For now we just create a new envelope.

	// Call LLM
	respEnv, err := e.llm.Execute(ctx, promptEnv)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("llm execution failed: %w", err)
	}

	respText, ok := respEnv.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("llm response payload is not string, got %T", respEnv.Payload)
	}

	// Parse JSON
	// We need a pointer to a new instance for json.Unmarshal
	// If e.target is `Order` (struct type), reflect.New(e.target) creates `*Order`.
	// If e.target is `*Order` (ptr type), reflect.New(e.target) creates `**Order`.
	// Usually targetStruct is `Order{}`, so e.target is `Order`.
	ptr := reflect.New(e.target).Interface()

	if err := json.Unmarshal([]byte(respText), ptr); err != nil {
		return core.Envelope{}, fmt.Errorf("%w: extraction failed: %v", core.ErrSystemError, err)
	}

	// Return the value, not the pointer, to match the "struct definition" concept usually.
	// However, the prompt says: "Return a new Envelope containing the Struct as the payload."
	// If I pass `Order{}` to New, I expect `Order` in Payload.
	// reflect.New returns pointer to zero value.
	result := reflect.ValueOf(ptr).Elem().Interface()

	return core.NewEnvelope(result), nil
}

// Metadata returns the metadata for this extractor action.
func (e *ExtractorAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: e.name,
		Type: "extractor",
	}
}
