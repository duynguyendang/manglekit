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
	"time"
)

// ChatConfig configures the hosted, key-only backend: an OpenAI-compatible
// chat endpoint that returns token log-probabilities.
type ChatConfig struct {
	// BaseURL is the API root INCLUDING the version segment, e.g.
	// "https://api.openai.com/v1" or "http://localhost:8080/v1".
	BaseURL string
	// APIKey is sent as a bearer token; empty means no Authorization header
	// (useful against a local gateway).
	APIKey string
	// Model is the served model name.
	Model string
	// TopLogProbs is how many candidate tokens to request at the answer
	// position. Providers cap this (commonly 20); too small a value makes
	// labels unobservable, which the policy then escalates rather than
	// guessing. Defaults to 20.
	TopLogProbs int
	// HTTPClient overrides transport settings (timeouts, proxies, tests).
	HTTPClient *http.Client
}

// ChatScorer scores fixed-choice options through a hosted chat API by asking
// for ONE token with logprobs enabled, then reading the option labels out of
// the returned top_logprobs and renormalizing over exactly those.
//
// It needs no scoring server, which is what makes it the practical default for
// hosted models — at one honest cost: top_logprobs is a TRUNCATED view of the
// vocabulary, so a label outside the returned set is unobservable. That is
// reported as ErrLabelsUnobserved rather than silently normalized away.
//
// Token usage from the response is NOT retained: measurement (latency,
// tokens-per-case, calibration, permutation studies) is application-layer and
// lives with the caller, not in the kernel.
type ChatScorer struct {
	BaseURL     string
	APIKey      string
	Model       string
	TopLogProbs int
	Client      *http.Client
}

// NewChatScorer validates cfg and builds a ChatScorer.
func NewChatScorer(cfg ChatConfig) (*ChatScorer, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("scoring: ChatConfig.BaseURL is required")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("scoring: ChatConfig.Model is required")
	}
	top := cfg.TopLogProbs
	if top == 0 {
		top = 20
	}
	if top < 2 {
		return nil, fmt.Errorf("scoring: ChatConfig.TopLogProbs=%d is too small to rank options", top)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	return &ChatScorer{
		BaseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		APIKey:      cfg.APIKey,
		Model:       cfg.Model,
		TopLogProbs: top,
		Client:      client,
	}, nil
}

var _ Scorer = (*ChatScorer)(nil)

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	LogProbs    bool          `json:"logprobs"`
	TopLogProbs int           `json:"top_logprobs,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatTopLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
}

type chatContentLogProb struct {
	Token       string           `json:"token"`
	LogProb     float64          `json:"logprob"`
	TopLogProbs []chatTopLogProb `json:"top_logprobs"`
}

type chatResponse struct {
	Choices []struct {
		Message  chatMessage `json:"message"`
		LogProbs *struct {
			Content []chatContentLogProb `json:"content"`
		} `json:"logprobs"`
	} `json:"choices"`
}

// Score implements Scorer.
func (s *ChatScorer) Score(ctx context.Context, prompt string, labels, options []string) (*Decision, error) {
	if len(labels) != len(options) {
		return nil, fmt.Errorf("scoring: %d labels but %d options", len(labels), len(options))
	}
	if s.TopLogProbs < len(labels) {
		return nil, fmt.Errorf("scoring: TopLogProbs=%d cannot rank %d options", s.TopLogProbs, len(labels))
	}

	body, err := json.Marshal(chatRequest{
		Model:       s.Model,
		Messages:    []chatMessage{{Role: "user", Content: prompt}},
		MaxTokens:   1,
		Temperature: 0,
		LogProbs:    true,
		TopLogProbs: s.TopLogProbs,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.APIKey)
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scoring: chat/completions: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<22))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scoring: chat/completions: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("scoring: chat/completions decode: %w", err)
	}
	if len(cr.Choices) == 0 {
		return nil, errors.New("scoring: chat/completions returned no choices")
	}
	lp := cr.Choices[0].LogProbs
	if lp == nil || len(lp.Content) == 0 {
		return nil, fmt.Errorf("%w: the endpoint returned no logprobs (does this provider support them?)", ErrLabelsUnobserved)
	}
	return decisionFromAnswerToken(lp.Content[0], labels, options)
}

// decisionFromAnswerToken reads the option labels out of the answer position's
// logprob set. Token surfaces are trimmed so " A" and "A" resolve alike;
// non-option candidates in the list are DISCARDED, never folded into the total.
func decisionFromAnswerToken(content chatContentLogProb, labels, options []string) (*Decision, error) {
	seen := map[string]float64{}
	observe := func(token string, logProb float64) {
		k := strings.TrimSpace(token)
		if k == "" {
			return
		}
		if v, ok := seen[k]; !ok || logProb > v {
			seen[k] = logProb
		}
	}
	observe(content.Token, content.LogProb)
	for _, t := range content.TopLogProbs {
		observe(t.Token, t.LogProb)
	}

	logs := make([]float64, len(labels))
	var missing []string
	for i, lb := range labels {
		lp, ok := seen[strings.TrimSpace(lb)]
		if !ok {
			missing = append(missing, lb)
			continue
		}
		logs[i] = lp
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s (not among the %d candidates returned; raise TopLogProbs or use a /v1/score endpoint)",
			ErrLabelsUnobserved, strings.Join(missing, ", "), len(content.TopLogProbs))
	}
	// exp(logprob) then renormalize over exactly these labels — the restricted
	// softmax, done here because the API only hands back log-probabilities.
	return Build(labels, options, Softmax(logs))
}
