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

// TestChatScorerTruncatedIsNotSilentlyRenormalized: missing label = unknowable.
func TestChatScorerTruncatedIsNotSilentlyRenormalized(t *testing.T) {
	content := logProbContentForTest{Token: "A", LogProb: -0.2, TopLogProbs: []struct {
		Token   string  `json:"token"`
		LogProb float64 `json:"logprob"`
	}{top("A", -0.2), top("B", -1.8), top("maybe", -2.5)}}
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
	if !strings.Contains(err.Error(), "C") {
		t.Errorf("error must name the missing label, got %v", err)
	}

	// And the governed path: the flag set the policy sees.
	th := Thresholds{Auto: 0.8, Margin: 0.2, Review: 0.7}
	env := Request("x", nil, th)
	if env.Metadata["unobserved_distribution"] != "true" {
		t.Fatalf("escalation flag missing: %v", env.Metadata)
	}
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
