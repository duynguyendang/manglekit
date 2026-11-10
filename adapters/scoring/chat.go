package scoring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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
//
// Truncation is handled with bounds, not with a guess. Every token outside the
// returned top-K has probability <= p_K (the list is sorted descending), so for
// a missing label set M:
//
//	D = Σ_observed p + Σ_M p  <=  Σ_observed p + |M| · p_K
//	=> normalized(p_top) >= p_top / (Σ_observed p + |M| · p_K)   [sound LCB]
//
// Two consequences worth stating: the observed argmax is also the TRUE argmax
// (any missing label has p <= p_K <= p_top), so truncation never changes the
// winner — only how much we may trust the margin. And Distribution is set to
// the LOWER bounds, so a caller that ignores Truncated under-estimates
// confidence instead of over-estimating it. Upper carries the survivor-
// normalized view for reporting only.
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
	pK := math.Inf(1) // smallest logprob in the returned list, as a probability
	for _, t := range content.TopLogProbs {
		observe(t.Token, t.LogProb)
		p := math.Exp(t.LogProb)
		if p < pK {
			pK = p
		}
	}

	observed := make([]float64, len(labels))
	var missing []string
	sumObserved := 0.0
	for i, lb := range labels {
		lp, ok := seen[strings.TrimSpace(lb)]
		if !ok {
			missing = append(missing, lb)
			continue
		}
		observed[i] = math.Exp(lp)
		sumObserved += observed[i]
	}
	if sumObserved <= 0 {
		return nil, fmt.Errorf("%w: %s (none of the options appeared among the %d candidates returned)",
			ErrLabelsUnobserved, strings.Join(labels, ", "), len(content.TopLogProbs))
	}
	if len(missing) == 0 {
		d, err := Build(labels, options, observed)
		if err != nil {
			return nil, err
		}
		d.Source = sourceChatLogProbs
		return d, nil
	}

	// Truncated: bound the unknown mass and report both views.
	if math.IsInf(pK, 1) {
		pK = 0
	}
	denomUpperBound := sumObserved + float64(len(missing))*pK

	lower := make(map[string]float64, len(options))
	upper := make(map[string]float64, len(options))
	best, bestP := 0, -1.0
	for i, o := range options {
		u := observed[i] / sumObserved
		l := 0.0
		if observed[i] > 0 && denomUpperBound > 0 {
			l = observed[i] / denomUpperBound
		}
		upper[o] = u
		lower[o] = l
		if observed[i] > bestP {
			bestP, best = observed[i], i
		}
	}

	d, err := Build(labels, options, observed)
	if err != nil {
		return nil, err
	}
	// Overwrite with the conservative view: Distribution = lower bounds.
	d.Distribution = lower
	d.Lower = lower
	d.Upper = upper
	d.Choice = options[best]
	d.Margin = lower[options[best]] - secondLargest(lower)
	d.Entropy = entropyOf(lower)
	d.Truncated = true
	d.Missing = missing
	d.Source = sourceChatLogProbsTruncated
	return d, nil
}

// sourceChatLogProbs and its truncated variant are the score_source values a
// policy can gate on.
const (
	sourceChatLogProbs          = "chat_logprobs"
	sourceChatLogProbsTruncated = "chat_logprobs_truncated"
)

func secondLargest(m map[string]float64) float64 {
	top, second := -1.0, 0.0
	for _, v := range m {
		switch {
		case v > top:
			top, second = v, top
			if second < 0 {
				second = 0
			}
		case v > second:
			second = v
		}
	}
	if second < 0 {
		return 0
	}
	return second
}

func entropyOf(m map[string]float64) float64 {
	var h float64
	for _, v := range m {
		if v > 0 {
			h -= v * math.Log(v)
		}
	}
	return h
}
