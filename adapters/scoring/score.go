package scoring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ScoreConfig configures an OpenAI-compatible /v1/score endpoint (SGLang
// native API).
type ScoreConfig struct {
	BaseURL    string // e.g. "http://localhost:30000" (no /v1 suffix)
	Model      string
	HTTPClient *http.Client
	// Tokenizer resolves labels to token ids. Required unless LabelTokenIDs
	// supplies them directly.
	Tokenizer TokenIDResolver
	// LabelTokenIDs supplies label -> token id up front, for deployments
	// without a /tokenize endpoint. The caller asserts they are single tokens.
	LabelTokenIDs map[string]int
	// SpacePrefix scores " A" instead of "A" when the chat template glues a
	// space to the answer token — the case where the two are different ids.
	SpacePrefix bool
}

// ScoreServer scores through a /v1/score endpoint that accepts explicit
// label_token_ids, i.e. the exact path the fixed-choice recipe describes: the
// server restricts the softmax to the labels you name, so no candidate can be
// silently truncated the way a hosted top_logprobs list can.
type ScoreServer struct {
	BaseURL       string
	Model         string
	Client        *http.Client
	Tok           TokenIDResolver
	LabelTokenIDs map[string]int
	SpacePrefix   bool

	idMu    sync.Mutex
	idCache map[string][]int
}

// NewScoreServer validates cfg and builds the backend. Either Tokenizer or
// LabelTokenIDs must be able to resolve every label to one token id.
func NewScoreServer(cfg ScoreConfig) (*ScoreServer, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("scoring: ScoreConfig.BaseURL is required")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("scoring: ScoreConfig.Model is required")
	}
	if cfg.Tokenizer == nil && len(cfg.LabelTokenIDs) == 0 {
		return nil, errors.New("scoring: ScoreConfig needs a Tokenizer or LabelTokenIDs to resolve labels to token ids")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	ids := make(map[string]int, len(cfg.LabelTokenIDs))
	for k, v := range cfg.LabelTokenIDs {
		ids[k] = v
	}
	return &ScoreServer{
		BaseURL:       strings.TrimRight(cfg.BaseURL, "/"),
		Model:         cfg.Model,
		Client:        client,
		Tok:           cfg.Tokenizer,
		LabelTokenIDs: ids,
		SpacePrefix:   cfg.SpacePrefix,
		idCache:       map[string][]int{},
	}, nil
}

var _ Scorer = (*ScoreServer)(nil)

type scoreRequest struct {
	Model         string   `json:"model"`
	Query         string   `json:"query"`
	Items         []string `json:"items"`
	LabelTokenIDs []int    `json:"label_token_ids"`
	ApplySoftmax  bool     `json:"apply_softmax"`
}

type scoreResponse struct {
	// Scores is one list of probabilities per item, in label_token_ids order.
	Scores [][]float64 `json:"scores"`
}

// Score implements Scorer. One request per prompt: the server reads the
// vocabulary distribution at the answer position and returns the probability
// of exactly the label tokens.
func (s *ScoreServer) Score(ctx context.Context, prompt string, labels, options []string) (*Decision, error) {
	if len(labels) != len(options) {
		return nil, fmt.Errorf("scoring: %d labels but %d options", len(labels), len(options))
	}
	ids, err := s.resolveLabelIDs(ctx, labels)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(scoreRequest{
		Model:         s.Model,
		Query:         prompt,
		Items:         []string{""},
		LabelTokenIDs: ids,
		ApplySoftmax:  true,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/v1/score", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scoring: /v1/score: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<22))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scoring: /v1/score: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var sr scoreResponse
	if err := json.Unmarshal(raw, &sr); err != nil {
		return nil, fmt.Errorf("scoring: /v1/score decode: %w", err)
	}
	if len(sr.Scores) == 0 {
		return nil, fmt.Errorf("%w: /v1/score returned no scores", ErrLabelsUnobserved)
	}
	if len(sr.Scores[0]) != len(ids) {
		return nil, fmt.Errorf("scoring: /v1/score returned %d probabilities for %d labels", len(sr.Scores[0]), len(ids))
	}
	// apply_softmax already normalizes over the labels; Build renormalizes as
	// a no-op and derives margin/entropy from the same numbers.
	return Build(labels, options, sr.Scores[0])
}

// resolveLabelIDs maps labels to single token ids, in label order. Explicit
// LabelTokenIDs win (for servers without /tokenize); otherwise the tokenizer
// endpoint is consulted — which is also what enforces the one-token rule.
// Results are cached per label set: the mapping is constant for a running
// server, and the benchmark calls this once per case.
func (s *ScoreServer) resolveLabelIDs(ctx context.Context, labels []string) ([]int, error) {
	// SpacePrefix is part of the key: it changes which surface is tokenized,
	// so "A" and " A" must not share a cache entry.
	prefix := ""
	if s.SpacePrefix {
		prefix = " "
	}
	key := prefix + strings.Join(labels, "\x00")
	s.idMu.Lock()
	defer s.idMu.Unlock()
	if cached, ok := s.idCache[key]; ok {
		return cached, nil
	}

	if len(s.LabelTokenIDs) > 0 {
		ids := make([]int, 0, len(labels))
		for _, lb := range labels {
			id, ok := s.LabelTokenIDs[lb]
			if !ok {
				return nil, fmt.Errorf("scoring: no token id supplied for label %q", lb)
			}
			ids = append(ids, id)
		}
		s.idCache[key] = ids
		return ids, nil
	}

	surfaces := s.tokenSurfaces(labels)
	if err := ValidateSingleTokens(ctx, s.Tok, surfaces); err != nil {
		return nil, err
	}
	ids := make([]int, len(labels))
	for i, text := range surfaces {
		id, err := s.Tok.SingleTokenID(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("scoring: token id for %q: %w", text, err)
		}
		ids[i] = id
	}
	s.idCache[key] = ids
	return ids, nil
}

// tokenSurfaces applies the SpacePrefix convention: which exact text is sent
// to the tokenizer. The one-token rule MUST be checked against this text, not
// the bare label — that difference is precisely the " A" vs "A" trap, since a
// provider may tokenize the two into different counts.
func (s *ScoreServer) tokenSurfaces(labels []string) []string {
	out := make([]string, len(labels))
	for i, lb := range labels {
		if s.SpacePrefix {
			out[i] = " " + lb
		} else {
			out[i] = lb
		}
	}
	return out
}

// HTTPTokenizer talks to an OpenAI-compatible /tokenize endpoint (SGLang and
// vLLM both expose one) to count tokens and resolve single-token ids.
type HTTPTokenizer struct {
	BaseURL string
	Client  *http.Client
}

type tokenizeRequest struct {
	Text             string `json:"text"`
	AddSpecialTokens bool   `json:"add_special_tokens"`
}

type tokenizeResponse struct {
	Tokens  []int    `json:"tokens"`
	Count   int      `json:"count"`
	Content []string `json:"content"` // some servers return pieces instead of ids
}

// NewHTTPTokenizer builds a tokenizer client for base (no /v1 suffix needed).
func NewHTTPTokenizer(baseURL string, client *http.Client) *HTTPTokenizer {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &HTTPTokenizer{BaseURL: strings.TrimRight(baseURL, "/"), Client: client}
}

// TokenLen returns the token count of text.
func (t *HTTPTokenizer) TokenLen(ctx context.Context, text string) (int, error) {
	tk, err := t.call(ctx, text)
	if err != nil {
		return 0, err
	}
	switch {
	case len(tk.Tokens) > 0:
		return len(tk.Tokens), nil
	case tk.Count > 0:
		return tk.Count, nil
	case len(tk.Content) > 0:
		return len(tk.Content), nil
	}
	return 0, errors.New("scoring: /tokenize returned nothing")
}

// SingleTokenID returns the token id when text occupies exactly one token.
func (t *HTTPTokenizer) SingleTokenID(ctx context.Context, text string) (int, error) {
	tk, err := t.call(ctx, text)
	if err != nil {
		return 0, err
	}
	if len(tk.Tokens) != 1 {
		return 0, fmt.Errorf("text %q is %d tokens, need exactly 1", text, len(tk.Tokens))
	}
	return tk.Tokens[0], nil
}

func (t *HTTPTokenizer) call(ctx context.Context, text string) (*tokenizeResponse, error) {
	body, err := json.Marshal(tokenizeRequest{Text: text})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.BaseURL+"/tokenize", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scoring: /tokenize: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<22))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scoring: /tokenize: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var tk tokenizeResponse
	if err := json.Unmarshal(raw, &tk); err != nil {
		return nil, fmt.Errorf("scoring: /tokenize decode: %w", err)
	}
	return &tk, nil
}

var _ TokenIDResolver = (*HTTPTokenizer)(nil)
