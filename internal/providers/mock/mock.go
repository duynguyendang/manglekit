package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/google/mangle/ast"
)

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
		&key:   &val,
		&keyB: &valB,
	}), nil
}

// Retriever is a mock retriever.
type Retriever struct {
	pairs map[string]string
}

// NewRetriever creates a new mock retriever.
func NewRetriever(pairs map[string]string) *Retriever {
	return &Retriever{pairs}
}

func (r *Retriever) Retrieve(ctx context.Context, req retrieve.Request) (retrieve.Result, error) {
	if text, ok := r.pairs[req.Query]; ok {
		return retrieve.Result{Docs: []core.Doc{{Text: text}}}, nil
	}
	return retrieve.Result{}, nil
}

// RetrieverOptions is the options for the mock retriever.
type RetrieverOptions struct {
	Pairs map[string]string `json:"pairs"`
}

// Reranker is a mock reranker.
type Reranker struct {
	passthrough map[string]bool
}

// NewReranker creates a new mock reranker.
func NewReranker(passthrough map[string]bool) *Reranker {
	return &Reranker{passthrough}
}

// Rerank reranks the documents.
func (r *Reranker) Rerank(ctx context.Context, req rerank.Request) ([]rerank.ScoredDoc, error) {
	if r.passthrough[req.Query] {
		var scoredDocs []rerank.ScoredDoc
		for _, doc := range req.Docs {
			scoredDocs = append(scoredDocs, rerank.ScoredDoc{Doc: doc})
		}
		return scoredDocs, nil
	}
	return nil, nil
}

// RerankerOptions is the options for the mock reranker.
type RerankerOptions struct {
	Passthrough map[string]bool `json:"passthrough"`
}

// LLM is a mock LLM.
type LLM struct {
	model string
}

// NewLLM creates a new mock LLM.
func NewLLM(model string) *LLM {
	return &LLM{model}
}

// Complete generates a response.
func (l *LLM) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	return llm.Response{Text: fmt.Sprintf("model: %s query: %s", l.model, req.Prompt)}, nil
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
	Model string `json:"model"`
}

// Tool is a mock tool.
type Tool struct {
	Name       string
	Fn         func(Params) (Object, error)
	lastParams Params
	mu         sync.Mutex
}

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

func (t *Tool) GetName() string {
	return t.Name
}

func (t *Tool) GetDescription() string {
	return "mock tool"
}

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
