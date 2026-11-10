package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func chatStub(t *testing.T, content logProbContentForTest, probe func(map[string]any)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if probe != nil {
			probe(req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message":  map[string]any{"role": "assistant", "content": strings.TrimSpace(content.Token)},
				"logprobs": map[string]any{"content": []logProbContentForTest{content}},
			}},
		})
	}))
}

// logProbContentForTest mirrors the wire shape for test fixtures.
type logProbContentForTest = struct {
	Token       string  `json:"token"`
	LogProb     float64 `json:"logprob"`
	TopLogProbs []struct {
		Token   string  `json:"token"`
		LogProb float64 `json:"logprob"`
	} `json:"top_logprobs"`
}

func top(token string, logprob float64) struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
} {
	return struct {
		Token   string  `json:"token"`
		LogProb float64 `json:"logprob"`
	}{token, logprob}
}

// TestChatScorerRestrictedSoftmax is the Jev claim, server-side free: one chat
// call, labels read from top_logprobs, total = 100% over the options only.
func TestChatScorerRestrictedSoftmax(t *testing.T) {
	content := logProbContentForTest{
		Token:   " B",
		LogProb: -0.34,
		TopLogProbs: []struct {
			Token   string  `json:"token"`
			LogProb float64 `json:"logprob"`
		}{
			top(" B", -0.34), top(" A", -1.61), top(" C", -2.90),
			top(" The", -5.52), top(" I", -5.60), // non-options: must be discarded
		},
	}
	srv := chatStub(t, content, func(req map[string]any) {
		if req["logprobs"] != true {
			t.Errorf("must request logprobs: %v", req)
		}
		if req["max_tokens"] != float64(1) {
			t.Errorf("must cap max_tokens at 1, got %v", req["max_tokens"])
		}
		if req["temperature"] != float64(0) {
			t.Errorf("must pin temperature=0, got %v", req["temperature"])
		}
		if tp := req["top_logprobs"]; tp == nil || tp.(float64) < 3 {
			t.Errorf("must ask for top_logprobs >= 3, got %v", tp)
		}
	})
	defer srv.Close()

	s, err := NewChatScorer(ChatConfig{BaseURL: srv.URL, APIKey: "k", Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Score(context.Background(), "Route this", []string{"A", "B", "C"},
		[]string{"billing", "tech", "account"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Choice != "tech" {
		t.Fatalf("winner = %q, want tech", d.Choice)
	}
	// e^-.34/(e^-.34+e^-1.61+e^-2.90) = .7118/.9667 = .7363
	if p := d.Prob("tech"); math.Abs(p-0.7363) > 0.001 {
		t.Errorf("tech prob = %.4f, want 0.7363", p)
	}
	sum := 0.0
	for _, v := range d.Distribution {
		sum += v
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Errorf("must total 100%%, got %.6f", sum)
	}
	if len(d.Distribution) != 3 {
		t.Errorf("noise tokens leaked into the distribution: %v", d.Distribution)
	}
}

// TestChatScorerTruncatedYieldsSoundBounds: a missing label is no longer a
// dead end — it is a BOUND. Truncation cannot change the winner (any missing
// label has p <= p_K <= p_top), so the choice stands, while Distribution
// carries sound LOWER bounds and Upper carries the survivor-normalized view
// for reporting. Renormalizing over survivors and calling it confidence is
// exactly the failure mode this replaces.
func TestChatScorerTruncatedYieldsSoundBounds(t *testing.T) {
	content := logProbContentForTest{Token: "A", LogProb: -0.2, TopLogProbs: []struct {
		Token   string  `json:"token"`
		LogProb float64 `json:"logprob"`
	}{top("A", -0.2), top("B", -1.8), top("maybe", -2.5)}} // p_K = e^-2.5; C absent
	srv := chatStub(t, content, nil)
	defer srv.Close()

	s, err := NewChatScorer(ChatConfig{BaseURL: srv.URL, Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Score(context.Background(), "Route this", []string{"A", "B", "C"},
		[]string{"billing", "tech", "account"})
	if err != nil {
		t.Fatalf("truncation must yield bounds, not an error: %v", err)
	}
	if !d.Truncated {
		t.Fatal("Truncated must be set")
	}
	if len(d.Missing) != 1 || d.Missing[0] != "C" {
		t.Fatalf("Missing must name the unobserved label, got %v", d.Missing)
	}
	if d.Choice != "billing" {
		t.Fatalf("truncation cannot change the winner, got %q", d.Choice)
	}
	if d.Source != "chat_logprobs_truncated" {
		t.Errorf("score_source must record the truncation, got %q", d.Source)
	}

	sumLower, sumUpper := 0.0, 0.0
	for _, o := range []string{"billing", "tech", "account"} {
		sumLower += d.Distribution[o]
		sumUpper += d.Upper[o]
		if d.Distribution[o] > d.Upper[o]+1e-12 {
			t.Errorf("lower bound must not exceed the upper view for %s", o)
		}
	}
	if sumLower >= 1 {
		t.Errorf("lower bounds must not sum to 1 — unknown mass exists, got %.4f", sumLower)
	}
	if math.Abs(sumUpper-1) > 1e-9 {
		t.Errorf("upper view must sum to 1, got %.4f", sumUpper)
	}
	if d.Distribution["account"] != 0 {
		t.Errorf("the unobserved option must get lower bound 0, got %v", d.Distribution["account"])
	}

	// The policy-visible consequence: gating on the bound is fail-closed, and
	// the flag set still marks the distribution as truncated.
	th := Thresholds{Auto: 0.8, Margin: 0.2, Review: 0.7}
	env := Request("x", d, th)
	if env.Metadata["truncated_top_k"] != "true" {
		t.Fatalf("truncation must reach the policy as a fact/flag: %v", env.Metadata)
	}
	if !containsFact(env.Facts, `score_lcb_pm("billing"`) {
		t.Fatalf("lower bounds must be published for the policy to gate on: %v", env.Facts)
	}
	if !containsFact(env.Facts, `truncated_top_k("true")`) {
		t.Fatalf("truncation fact missing: %v", env.Facts)
	}
	if th.Flags(d)["below_auto"] != "true" {
		t.Errorf("a bounded top must not clear a 0.80 bar: %v", th.Flags(d))
	}
}

// TestChatScorerNothingObservedStillErrors: with NO option among the returned
// candidates there is nothing to bound, so it remains a hard error.
func TestChatScorerNothingObservedStillErrors(t *testing.T) {
	content := logProbContentForTest{Token: "The", LogProb: -0.1, TopLogProbs: []struct {
		Token   string  `json:"token"`
		LogProb float64 `json:"logprob"`
	}{top("The", -0.1), top("maybe", -2.5)}}
	srv := chatStub(t, content, nil)
	defer srv.Close()

	s, err := NewChatScorer(ChatConfig{BaseURL: srv.URL, Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Score(context.Background(), "Route this", []string{"A", "B", "C"},
		[]string{"billing", "tech", "account"})
	if !errors.Is(err, ErrLabelsUnobserved) {
		t.Fatalf("want ErrLabelsUnobserved, got %v", err)
	}

	// And the governed path for a nil decision: the flag set the policy sees.
	env := Request("x", nil, Thresholds{Auto: 0.8, Margin: 0.2, Review: 0.7})
	if env.Metadata["unobserved_distribution"] != "true" {
		t.Fatalf("escalation flag missing: %v", env.Metadata)
	}
}

func containsFact(facts []string, needle string) bool {
	for _, f := range facts {
		if strings.Contains(f, needle) {
			return true
		}
	}
	return false
}

func TestChatScorerNoLogprobsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{
			"message": map[string]any{"role": "assistant", "content": "A"},
		}}})
	}))
	defer srv.Close()

	s, err := NewChatScorer(ChatConfig{BaseURL: srv.URL, Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Score(context.Background(), "p", []string{"A"}, []string{"a"}); !errors.Is(err, ErrLabelsUnobserved) {
		t.Fatalf("a provider that ignores logprobs must be reported, got %v", err)
	}
}

func TestChatScorerHTTPStatusPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("api key must be sent as a bearer token, got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"quota"}}`))
	}))
	defer srv.Close()

	s, err := NewChatScorer(ChatConfig{BaseURL: srv.URL, APIKey: "secret", Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Score(context.Background(), "p", []string{"A"}, []string{"a"})
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("HTTP status must surface, got %v", err)
	}
	if errors.Is(err, ErrLabelsUnobserved) {
		t.Fatal("a 429 must not be misread as a truncated distribution")
	}
}

func TestChatScorerValidatesConfig(t *testing.T) {
	if _, err := NewChatScorer(ChatConfig{Model: "m"}); err == nil {
		t.Error("BaseURL required")
	}
	if _, err := NewChatScorer(ChatConfig{BaseURL: "http://x"}); err == nil {
		t.Error("Model required")
	}
	if _, err := NewChatScorer(ChatConfig{BaseURL: "http://x", Model: "m", TopLogProbs: 1}); err == nil {
		t.Error("TopLogProbs=1 cannot rank options")
	}
	s, err := NewChatScorer(ChatConfig{BaseURL: "http://x/", Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if s.TopLogProbs != 20 || s.BaseURL != "http://x" {
		t.Errorf("defaults/trailing-slash wrong: %+v", s)
	}
}
