package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// scoreStub serves both /v1/score and /tokenize the way SGLang does, and
// records how many times each was hit so caching behaviour is observable.
func scoreStub(t *testing.T, probs []float64, tokenIDs map[string][]int) (url string, scoreCalls, tokCalls *atomic.Int64) {
	t.Helper()
	var sc, tc atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/score":
			sc.Add(1)
			var req scoreRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !req.ApplySoftmax {
				t.Errorf("must ask the server for a restricted softmax")
			}
			if len(req.LabelTokenIDs) == 0 {
				t.Errorf("label_token_ids must be populated")
			}
			if len(req.Items) != 1 {
				t.Errorf("expected a single item (the prompt), got %d", len(req.Items))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"scores": [][]float64{probs}})
		case "/tokenize":
			tc.Add(1)
			var req tokenizeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			ids, ok := tokenIDs[req.Text]
			if !ok {
				ids = []int{} // unknown surface -> zero tokens -> validation fails
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"tokens": ids, "count": len(ids)})
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &sc, &tc
}

var defaultIDs = map[string][]int{"A": {32}, "B": {33}, "C": {34}, " A": {32}, " B": {33}, " C": {34}}

func TestScoreServerSingleCallRestrictedSoftmax(t *testing.T) {
	url, scoreCalls, tokCalls := scoreStub(t, []float64{0.05, 0.90, 0.05}, defaultIDs)
	s, err := NewScoreServer(ScoreConfig{
		BaseURL: url, Model: "m",
		Tokenizer: NewHTTPTokenizer(url, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	d, err := s.Score(ctx, "Route this", []string{"A", "B", "C"}, []string{"billing", "tech", "account"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Choice != "tech" {
		t.Fatalf("winner = %q", d.Choice)
	}
	if scoreCalls.Load() != 1 {
		t.Errorf("one prompt must be one /v1/score call, got %d", scoreCalls.Load())
	}
	firstTok := tokCalls.Load()
	if firstTok == 0 {
		t.Error("labels must be validated through /tokenize")
	}

	// Second prompt: token ids are cached, so no new tokenizer traffic.
	if _, err := s.Score(ctx, "Route that", []string{"A", "B", "C"}, []string{"billing", "tech", "account"}); err != nil {
		t.Fatal(err)
	}
	if tokCalls.Load() != firstTok {
		t.Errorf("token ids must be cached per label set: %d -> %d", firstTok, tokCalls.Load())
	}
	if scoreCalls.Load() != 2 {
		t.Errorf("each prompt still needs its own score call, got %d", scoreCalls.Load())
	}
}

// TestScoreServerRejectsMultiTokenLabel is the " A" vs "A" trap: a label that
// is not exactly one token cannot be scored by reading one logit.
func TestScoreServerRejectsMultiTokenLabel(t *testing.T) {
	url, _, _ := scoreStub(t, []float64{1}, map[string][]int{"maybe": {32, 33}})
	s, err := NewScoreServer(ScoreConfig{BaseURL: url, Model: "m", Tokenizer: NewHTTPTokenizer(url, nil)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Score(context.Background(), "p", []string{"maybe"}, []string{"x"})
	if err == nil || !strings.Contains(err.Error(), "requires exactly 1") {
		t.Fatalf("multi-token label must be rejected, got %v", err)
	}
}

func TestScoreServerSpacePrefixUsesSpacedSurface(t *testing.T) {
	// Only " A" is a known single token; bare "A" tokenizes to two, so a
	// scorer that forgot the space prefix must fail.
	ids := map[string][]int{" A": {32}, " B": {33}, " C": {34}, "A": {1, 2}, "B": {1, 2}, "C": {1, 2}}
	url, _, _ := scoreStub(t, []float64{0.9, 0.05, 0.05}, ids)
	s, err := NewScoreServer(ScoreConfig{
		BaseURL: url, Model: "m", Tokenizer: NewHTTPTokenizer(url, nil), SpacePrefix: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Score(context.Background(), "p", []string{"A", "B", "C"}, []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("space-prefixed scoring should succeed: %v", err)
	}
	if d.Choice != "a" {
		t.Fatalf("winner = %q", d.Choice)
	}
}

func TestScoreServerLabelTokenIDsBypassesTokenizer(t *testing.T) {
	// Server with /v1/score only: any /tokenize hit fails the test.
	var tokHits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokenize" {
			tokHits.Add(1)
			http.NotFound(w, r)
			return
		}
		var req scoreRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.LabelTokenIDs) != 3 || req.LabelTokenIDs[2] != 999 {
			t.Errorf("expected supplied ids, got %v", req.LabelTokenIDs)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"scores": [][]float64{{0.1, 0.2, 0.7}}})
	}))
	defer srv.Close()

	s, err := NewScoreServer(ScoreConfig{
		BaseURL: srv.URL, Model: "m",
		LabelTokenIDs: map[string]int{"A": 1, "B": 2, "C": 999},
	})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Score(context.Background(), "p", []string{"A", "B", "C"}, []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Choice != "c" {
		t.Fatalf("winner = %q", d.Choice)
	}
	if tokHits.Load() != 0 {
		t.Errorf("LabelTokenIDs must skip /tokenize entirely, %d hits", tokHits.Load())
	}
}

func TestScoreServerMissingLabelID(t *testing.T) {
	s, err := NewScoreServer(ScoreConfig{BaseURL: "http://unused", Model: "m", LabelTokenIDs: map[string]int{"A": 1}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Score(context.Background(), "p", []string{"A", "B"}, []string{"a", "b"})
	if err == nil || !strings.Contains(err.Error(), `label "B"`) {
		t.Fatalf("must name the unresolvable label, got %v", err)
	}
}

func TestScoreServerValidatesConfig(t *testing.T) {
	if _, err := NewScoreServer(ScoreConfig{Model: "m"}); err == nil {
		t.Error("BaseURL required")
	}
	if _, err := NewScoreServer(ScoreConfig{BaseURL: "http://x"}); err == nil {
		t.Error("Model required")
	}
	if _, err := NewScoreServer(ScoreConfig{BaseURL: "http://x", Model: "m"}); err == nil {
		t.Error("needs a Tokenizer or LabelTokenIDs")
	}
}

func TestScoreServerTruncatedResponseIsNotSilent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokenize" {
			_ = json.NewEncoder(w).Encode(map[string]any{"tokens": []int{32}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"scores": [][]float64{{0.5}}}) // one of two labels
	}))
	defer srv.Close()

	s, err := NewScoreServer(ScoreConfig{BaseURL: srv.URL, Model: "m", Tokenizer: NewHTTPTokenizer(srv.URL, nil)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Score(context.Background(), "p", []string{"A", "B"}, []string{"a", "b"})
	if err == nil || strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("a short score vector must be reported cleanly, got %v", err)
	}
	if !strings.Contains(err.Error(), "probabilities") {
		t.Fatalf("error must explain the mismatch, got %v", err)
	}
	var target error = ErrLabelsUnobserved
	if errors.Is(err, target) {
		t.Log("note: short response is also a labels-unobserved condition")
	}
}
