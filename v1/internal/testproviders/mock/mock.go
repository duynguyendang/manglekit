package mock

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/google/mangle/ast"
)

func Register(r *manglekit.Registry) {
	manglekit.Register(r, &RetrieverOptions{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *RetrieverOptions) (core.Retriever, error) {
			return NewRetriever(cfg.Pairs), nil
		},
	)
	manglekit.Register(r, &RerankerOptions{},
		func(ctx context.Context, deps diapi.RerankerDeps, cfg *RerankerOptions) (core.Reranker, error) {
			return NewReranker(cfg.Passthrough), nil
		},
	)
	manglekit.Register(r, &LLMOptions{},
		func(ctx context.Context, deps diapi.LLMDeps, cfg *LLMOptions) (core.LLMClient, error) {
			return NewLLM(cfg.Model, cfg.Response), nil
		},
	)
	manglekit.Register(r, &EmbedderOptions{},
		func(ctx context.Context, deps diapi.EmbedderDeps, cfg *EmbedderOptions) (ai.Embedder, error) {
			return &Embedder{}, nil
		},
	)
}

type Params map[string]any
type Object map[string]any

func ConstantsToParams(args []ast.Constant) (Params, error) {
	params := make(Params)
	if len(args) > 0 {
		params["a"] = args[0].Symbol
	}
	return params, nil
}

func ObjectToConstant(obj Object) (ast.Constant, error) {
	key, err := ast.Name("/a")
	if err != nil {
		return ast.Constant{}, err
	}
	val, err := ast.Name(obj["a"].(string))
	if err != nil {
		return ast.Constant{}, err
	}
	keyB, err := ast.Name("/b")
	if err != nil {
		return ast.Constant{}, err
	}
	valB, err := ast.Name(obj["b"].(string))
	if err != nil {
		return ast.Constant{}, err
	}
	return *ast.Struct(map[*ast.Constant]*ast.Constant{
		&key:  &val,
		&keyB: &valB,
	}), nil
}

// Retriever is a mock retriever.
type Retriever struct {
	pairs        map[string]string
	RetrieveFunc func(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error)
}

// NewRetriever creates a new mock retriever.
func NewRetriever(pairs map[string]string) *Retriever {
	return &Retriever{pairs: pairs}
}

func (r *Retriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	if r.RetrieveFunc != nil {
		return r.RetrieveFunc(ctx, req)
	}
	if text, ok := r.pairs[req.Query]; ok {
		return core.RetrieveResult{Docs: []core.Doc{{Text: text}}}, nil
	}
	return core.RetrieveResult{}, nil
}

// RetrieverOptions is the options for the mock retriever.
type RetrieverOptions struct {
	Pairs map[string]string `json:"pairs"`
}

func (o *RetrieverOptions) ProviderName() string    { return "mock-retriever" }
func (o *RetrieverOptions) ProviderKind() core.Kind { return core.KindRetriever }
func (o *RetrieverOptions) GetProviderOptions() any { return o }

// Reranker is a mock reranker.
type Reranker struct {
	passthrough map[string]bool
}

// NewReranker creates a new mock reranker.
func NewReranker(passthrough map[string]bool) *Reranker {
	return &Reranker{passthrough}
}

// Rerank reranks the documents.
func (r *Reranker) Rerank(ctx context.Context, req core.RerankRequest) ([]core.ScoredDoc, error) {
	if r.passthrough[req.Query] {
		var scoredDocs []core.ScoredDoc
		for _, doc := range req.Docs {
			scoredDocs = append(scoredDocs, core.ScoredDoc{Doc: doc})
		}
		return scoredDocs, nil
	}
	return nil, nil
}

// RerankerOptions is the options for the mock reranker.
type RerankerOptions struct {
	Passthrough map[string]bool `json:"passthrough"`
}

func (o *RerankerOptions) ProviderName() string    { return "mock-reranker" }
func (o *RerankerOptions) ProviderKind() core.Kind { return core.KindReranker }
func (o *RerankerOptions) GetProviderOptions() any { return o }

// LLM is a mock LLM.
type LLM struct {
	model        string
	response     string
	CompleteFunc func(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error)
}

// NewLLM creates a new mock LLM.
func NewLLM(model, response string) *LLM {
	return &LLM{model: model, response: response}
}

// Complete generates a response.
func (l *LLM) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	if l.CompleteFunc != nil {
		return l.CompleteFunc(ctx, req)
	}
	if l.response != "" {
		return core.LLMResponse{Text: l.response, Usage: make(map[string]int)}, nil
	}
	var fullPrompt strings.Builder
	fullPrompt.WriteString(req.Prompt)
	if len(req.Context) > 0 {
		fullPrompt.WriteString(" context: ")
		fullPrompt.WriteString(strings.Join(req.Context, " "))
	}
	return core.LLMResponse{
		Text:  fmt.Sprintf("model: %s prompt: %s", l.model, fullPrompt.String()),
		Usage: make(map[string]int),
	}, nil
}

// Model returns the model name.
func (l *LLM) Model() string {
	return l.model
}

func (l *LLM) GetName() string {
	return "mock-llm"
}

// LLMOptions is the options for the mock LLM.
type LLMOptions struct {
	Model    string `json:"model"`
	Response string `json:"response,omitempty"`
}

func (o *LLMOptions) ProviderName() string    { return "mock-llm" }
func (o *LLMOptions) ProviderKind() core.Kind { return core.KindLLM }
func (o *LLMOptions) GetProviderOptions() any { return o }

// EmbedderOptions is the options for the mock embedder.
type EmbedderOptions struct{}

func (o *EmbedderOptions) ProviderName() string    { return "mock-embedder" }
func (o *EmbedderOptions) ProviderKind() core.Kind { return core.KindEmbedder }
func (o *EmbedderOptions) GetProviderOptions() any { return o }

// Tool is a mock tool that can be used in tests.
type Tool struct {
	// Name is the name of the tool.
	Name string
	// Fn is the function that the tool executes.
	Fn func(Params) (Object, error)
	// lastParams is the last parameters that the tool was called with.
	lastParams Params
	// mu is a mutex to protect lastParams.
	mu sync.Mutex
}

// GetLastParams returns the last parameters that the tool was called with.
func (t *Tool) GetLastParams() Params {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Return a copy to avoid race conditions on the map.
	p := make(Params)
	for k, v := range t.lastParams {
		p[k] = v
	}
	return p
}

// GetName returns the name of the tool.
func (t *Tool) GetName() string {
	return t.Name
}

// GetDescription returns the description of the tool.
func (t *Tool) GetDescription() string {
	return "mock tool"
}

// GetFn returns the function that the tool executes.
func (t *Tool) GetFn() func(args []ast.Constant) (ast.Constant, error) {
	return func(args []ast.Constant) (ast.Constant, error) {
		params, err := ConstantsToParams(args)
		if err != nil {
			return ast.Constant{}, err
		}

		obj, err := t.Fn(params)
		if err != nil {
			return ast.Constant{}, err
		}

		// Store params after Fn is called so mutations (like adding __called) are captured.
		t.mu.Lock()
		t.lastParams = params
		t.mu.Unlock()

		return ObjectToConstant(obj)
	}
}

// Embedder is a mock embedder.
type Embedder struct{}

// Name returns the name of the embedder.
func (e *Embedder) Name() string {
	return "mock-embedder"
}

// Register registers the embedder.
func (e *Embedder) Register(r api.Registry) {}

// Embed embeds the documents.
func (e *Embedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	return &ai.EmbedResponse{}, nil
}

// RuleSet is a mock ruleset.
type RuleSet struct{}

// NewRuleSet creates a new mock ruleset.
func NewRuleSet() *RuleSet {
	return &RuleSet{}
}

// Evaluate evaluates the rules.
func (r *RuleSet) Evaluate(ctx context.Context, stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	return core.RuleResult{Allowed: true}, nil
}

// EvaluateFacts evaluates the rules using explicit facts.
func (r *RuleSet) EvaluateFacts(ctx context.Context, stage core.Stage, facts []ast.Atom, a *core.Answer) (core.RuleResult, error) {
	return core.RuleResult{Allowed: true}, nil
}

// ToolOptions is the options for the mock tool.
type ToolOptions struct{}

func (o *ToolOptions) ProviderName() string    { return "noop-tool" }
func (o *ToolOptions) ProviderKind() core.Kind { return core.Kind("tool") } // Not a real kind.
func (o *ToolOptions) GetProviderOptions() any { return o }

// NoopTool is a tool that does nothing.
type NoopTool struct{}

// Execute does nothing and returns nil.
func (t *NoopTool) Execute(ctx context.Context, execCtx *core.ExecutionContext) error {
	return nil
}

// NewTool creates a new mock tool.
func NewTool() core.Tool {
	return &NoopTool{}
}
