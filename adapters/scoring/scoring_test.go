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

	"github.com/duynguyendang/manglekit/core"
)

func TestBuildNormalizesOverOptionsOnly(t *testing.T) {
	labels := []string{"A", "B", "C"}
	options := []string{"billing", "tech", "account"}
	// Unnormalized raw weights: total 2.0, so the winner is 0.6 after renorm.
	d, err := Build(labels, options, []float64{1.2, 0.6, 0.2})
	if err != nil {
		t.Fatal(err)
	}
	sum := 0.0
	for _, v := range d.Distribution {
		sum += v
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("distribution must total 100%%, got %.6f", sum)
	}
	if d.Choice != "billing" || math.Abs(d.Prob("billing")-0.6) > 1e-9 {
		t.Fatalf("wrong winner: %+v", d.Distribution)
	}
	if math.Abs(d.Margin-0.3) > 1e-9 { // .6 - .3
		t.Errorf("margin = %.4f, want 0.3", d.Margin)
	}
	if d.Entropy <= 0 || d.Entropy > math.Log(3) {
		t.Errorf("entropy out of range for 3 options: %.4f", d.Entropy)
	}
}

func TestBuildRejectsDegenerateInput(t *testing.T) {
	if _, err := Build([]string{"A"}, []string{"a", "b"}, []float64{1}); err == nil {
		t.Error("length mismatch must error")
	}
	if _, err := Build(nil, nil, nil); err == nil {
		t.Error("no options must error")
	}
	if _, err := Build([]string{"A"}, []string{"a"}, []float64{0}); err == nil {
		t.Error("all-zero probabilities must error rather than divide by zero")
	}
}

// TestBuildRejectsDuplicates: a repeated option would collapse the
// distribution (fewer keys than options) and could report a Choice carrying
// another index's probability. Refuse instead of guessing.
func TestBuildRejectsDuplicates(t *testing.T) {
	_, err := Build([]string{"A", "B"}, []string{"billing", "billing"}, []float64{0.6, 0.4})
	if err == nil || !strings.Contains(err.Error(), `duplicate option "billing"`) {
		t.Fatalf("duplicate options must error naming them, got %v", err)
	}
	_, err = Build([]string{"A", "A"}, []string{"billing", "tech"}, []float64{0.6, 0.4})
	if err == nil || !strings.Contains(err.Error(), `duplicate label "A"`) {
		t.Fatalf("duplicate labels must error naming them, got %v", err)
	}
}

// TestBuildTieBreaksToFirstOption pins the convention: a Decision must not
// depend on which duplicate arrived last, and reporting must be reproducible.
func TestBuildTieBreaksToFirstOption(t *testing.T) {
	d, err := Build([]string{"A", "B", "C"}, []string{"billing", "tech", "account"},
		[]float64{0.5, 0.5, 0.0})
	if err != nil {
		t.Fatal(err)
	}
	if d.Choice != "billing" {
		t.Fatalf("ties must resolve to the first option, got %q", d.Choice)
	}
	if d.Margin != 0 {
		t.Fatalf("a tie has zero margin, got %.4f", d.Margin)
	}
}

// TestBuildRejectsLogProbabilities documents the loud-failure property: every
// logprob is <= 0, so clamping yields a zero total instead of a confident
// wrong answer.
func TestBuildRejectsLogProbabilities(t *testing.T) {
	_, err := Build([]string{"A", "B"}, []string{"a", "b"}, []float64{-0.3, -1.5})
	if err == nil || !strings.Contains(err.Error(), "sum to zero") {
		t.Fatalf("log-probabilities passed by mistake must error clearly, got %v", err)
	}
}

func TestSortedOptionsIsDeterministic(t *testing.T) {
	d, err := Build([]string{"A", "B", "C"}, []string{"zeta", "alpha", "mid"}, []float64{0.2, 0.2, 0.6})
	if err != nil {
		t.Fatal(err)
	}
	got := d.sortedOptions()
	want := []string{"mid", "alpha", "zeta"} // ties broken alphabetically
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("sorted order = %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// Thresholds: read from the policy, and distrust anything ambiguous
// ---------------------------------------------------------------------------

type fakeQuery struct {
	rows map[string][]map[string]string
	err  error
}

func (f fakeQuery) Query(_ context.Context, _ []string, q string) ([]map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows[q], nil
}

func thresholdRows(name string, vals ...string) map[string][]map[string]string {
	rows := map[string][]map[string]string{}
	var out []map[string]string
	for _, v := range vals {
		out = append(out, map[string]string{"V": v})
	}
	rows[`threshold("`+name+`", V)`] = out
	return rows
}

func mergeRows(sets ...map[string][]map[string]string) map[string][]map[string]string {
	out := map[string][]map[string]string{}
	for _, s := range sets {
		for k, v := range s {
			out[k] = append(out[k], v...)
		}
	}
	return out
}

func TestThresholdsFromPolicy(t *testing.T) {
	q := fakeQuery{rows: mergeRows(
		thresholdRows("auto", "0.80"),
		thresholdRows("margin", "0.20"),
		thresholdRows("review", "0.70"),
	)}
	th, err := ThresholdsFromPolicy(context.Background(), q, "threshold")
	if err != nil {
		t.Fatal(err)
	}
	if th.Auto != 0.80 || th.Margin != 0.20 || th.Review != 0.70 {
		t.Fatalf("bad parse: %+v", th)
	}
}

func TestThresholdsFromPolicyFailModes(t *testing.T) {
	ctx := context.Background()

	if _, err := ThresholdsFromPolicy(ctx, nil, "threshold"); err == nil {
		t.Error("nil query must error")
	}

	// Missing bar: a silently-absent threshold would become a permissive 0.0.
	_, err := ThresholdsFromPolicy(ctx, fakeQuery{rows: mergeRows(
		thresholdRows("margin", "0.2"), thresholdRows("review", "0.7"))}, "threshold")
	if err == nil || !strings.Contains(err.Error(), `threshold("auto")`) {
		t.Errorf("missing auto must error naming it, got %v", err)
	}

	// Ambiguous: this is exactly what a bare fact + ReloadPolicy produces.
	_, err = ThresholdsFromPolicy(ctx, fakeQuery{rows: mergeRows(
		thresholdRows("auto", "0.80", "0.70"),
		thresholdRows("margin", "0.2"), thresholdRows("review", "0.7"))}, "threshold")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("two values for one threshold must error, got %v", err)
	}

	// Inverted bars: every case would escalate.
	_, err = ThresholdsFromPolicy(ctx, fakeQuery{rows: mergeRows(
		thresholdRows("auto", "0.60"), thresholdRows("margin", "0.2"), thresholdRows("review", "0.90"))}, "threshold")
	if err == nil || !strings.Contains(err.Error(), "exceeds auto bar") {
		t.Errorf("review > auto must error, got %v", err)
	}

	// Non-numeric.
	_, err = ThresholdsFromPolicy(ctx, fakeQuery{rows: mergeRows(
		thresholdRows("auto", "high"), thresholdRows("margin", "0.2"), thresholdRows("review", "0.1"))}, "threshold")
	if err == nil || !strings.Contains(err.Error(), "not a number") {
		t.Errorf("non-numeric threshold must error, got %v", err)
	}

	// Engine error propagates.
	if _, err := ThresholdsFromPolicy(ctx, fakeQuery{err: errors.New("engine down")}, "threshold"); err == nil {
		t.Error("query error must propagate")
	}
}

// ---------------------------------------------------------------------------
// Flags: the only place a number becomes a decision input
// ---------------------------------------------------------------------------

func decisionFor(t *testing.T, top, second float64) *Decision {
	t.Helper()
	d, err := Build([]string{"A", "B", "C"}, []string{"a", "b", "c"},
		[]float64{top, second, 1 - top - second})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestThresholdsFlags(t *testing.T) {
	cases := []struct {
		name       string
		th         Thresholds
		top, sec   float64
		wantFlags  []string
		wantAbsent []string
	}{
		{
			name: "clears the bar", th: Thresholds{Auto: 0.80, Margin: 0.20, Review: 0.70},
			top: 0.90, sec: 0.07, wantAbsent: []string{"below_auto", "below_review", "thin_margin"},
		},
		{
			name: "review band", th: Thresholds{Auto: 0.80, Margin: 0.20, Review: 0.70},
			top: 0.74, sec: 0.18, wantFlags: []string{"below_auto"}, wantAbsent: []string{"below_review", "thin_margin"},
		},
		{
			name: "below floor", th: Thresholds{Auto: 0.80, Margin: 0.20, Review: 0.70},
			top: 0.55, sec: 0.37, wantFlags: []string{"below_auto", "below_review"}, wantAbsent: []string{"thin_margin"},
		},
		{
			// Reaching the margin rule needs a LOW auto bar: with three labels
			// summing to 1, top >= 0.80 forces the runner-up <= 0.20, so the
			// margin is >= 0.60 and can never be "thin". Pinned below.
			name: "thin margin (low bar)", th: Thresholds{Auto: 0.50, Margin: 0.25, Review: 0.40},
			top: 0.50, sec: 0.45, wantFlags: []string{"thin_margin"}, wantAbsent: []string{"below_auto", "below_review"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := decisionFor(t, tc.top, tc.sec)
			flags := tc.th.Flags(d)
			for _, f := range tc.wantFlags {
				if flags[f] != "true" {
					t.Errorf("expected flag %q, got %v", f, flags)
				}
			}
			for _, f := range tc.wantAbsent {
				if _, ok := flags[f]; ok {
					t.Errorf("unexpected flag %q in %v", f, flags)
				}
			}
		})
	}
}

// TestMarginRuleIsUnreachableForThreeLabelsAtHighBars documents the geometry:
// a margin gate only earns its keep on denser label sets or after calibration
// moves mass. Silence here would let someone "fix" a non-bug.
func TestMarginRuleIsUnreachableForThreeLabelsAtHighBars(t *testing.T) {
	th := Thresholds{Auto: 0.80, Margin: 0.20, Review: 0.70}
	for top := 0.80; top <= 0.999; top += 0.02 {
		for second := 0.001; second <= 1-top; second += 0.02 {
			d := decisionFor(t, top, second)
			if d.Top() >= th.Auto && d.Margin < th.Margin {
				t.Fatalf("unexpected thin-margin case at top=%.2f second=%.2f margin=%.3f", top, second, d.Margin)
			}
		}
	}
}

// TestFlagsForUnobservedDistribution: no probability does NOT mean no risk.
// The flag set is the policy's input for refusing to act.
func TestFlagsForUnobservedDistribution(t *testing.T) {
	th := Thresholds{Auto: 0.80, Margin: 0.2, Review: 0.7}
	flags := th.Flags(nil)
	if flags["unobserved_distribution"] != "true" {
		t.Fatalf("nil decision must publish unobserved_distribution, got %v", flags)
	}
	if len(flags) != 1 {
		t.Fatalf("no other flag may be asserted without a distribution, got %v", flags)
	}
}

// ---------------------------------------------------------------------------
// Request: the envelope the supervisor pre-check actually evaluates
// ---------------------------------------------------------------------------

func TestRequestCarriesFactsAndFlags(t *testing.T) {
	th := Thresholds{Auto: 0.80, Margin: 0.2, Review: 0.7}
	d := decisionFor(t, 0.74, 0.18)

	env := Request(map[string]string{"ticket": "T-1"}, d, th)
	if env.Metadata["below_auto"] != "true" {
		t.Fatalf("flags must reach metadata: %v", env.Metadata)
	}
	if env.Metadata["choice"] != "a" {
		t.Errorf("choice metadata missing: %v", env.Metadata)
	}
	joined := strings.Join(env.Facts, " ")
	for _, want := range []string{`confidence("0.740")`, `margin_value("0.560")`, `entropy_value(`} {
		if !strings.Contains(joined, want) {
			t.Errorf("facts must carry %s, got %q", want, joined)
		}
	}
	// Every fact must be a closed Datalog sentence.
	for _, f := range env.Facts {
		if !strings.HasSuffix(f, ").") {
			t.Errorf("fact not terminated: %q", f)
		}
	}
}

func TestRequestForUnobserved(t *testing.T) {
	th := Thresholds{Auto: 0.8, Margin: 0.2, Review: 0.7}
	env := Request("prompt", nil, th)
	if env.Metadata["unobserved_distribution"] != "true" {
		t.Fatalf("missing flag: %v", env.Metadata)
	}
	if len(env.Facts) != 0 {
		t.Errorf("no numbers may be published without a distribution, got %v", env.Facts)
	}
}

// TestRequestMergesIntoAnExistingEnvelope: ExecuteByName accepts a
// core.Envelope as input and unwraps it, so wrapping one inside another would
// silently break the action's payload type assertion. Caller facts, labels and
// metadata must survive the merge.
func TestRequestMergesIntoAnExistingEnvelope(t *testing.T) {
	th := Thresholds{Auto: 0.80, Margin: 0.20, Review: 0.70}
	d := decisionFor(t, 0.74, 0.18)

	base := core.NewEnvelope(routePayload{Ticket: "T-9"})
	base.SecurityLabels = append(base.SecurityLabels, "tainted")
	base.Facts = append(base.Facts, `tenant("Req", "acme").`)
	base.Metadata["tenant"] = "acme"

	env := Request(base, d, th)

	if _, nested := env.Payload.(core.Envelope); nested {
		t.Fatal("payload must not be an envelope wrapped in an envelope")
	}
	if _, ok := env.Payload.(routePayload); !ok {
		t.Fatalf("inner payload must be preserved, got %T", env.Payload)
	}
	if env.Metadata["tenant"] != "acme" || env.Metadata["below_auto"] != "true" {
		t.Fatalf("caller metadata and scoring flags must coexist: %v", env.Metadata)
	}
	if len(env.SecurityLabels) != 1 || env.SecurityLabels[0] != "tainted" {
		t.Fatalf("security labels must survive: %v", env.SecurityLabels)
	}
	joined := strings.Join(env.Facts, " ")
	if !strings.Contains(joined, `tenant("Req", "acme")`) || !strings.Contains(joined, `confidence("0.740")`) {
		t.Fatalf("both caller facts and scoring facts must be present: %q", joined)
	}
}

type routePayload struct{ Ticket string }

// TestScoreServerSpacePrefixIsPartOfTheCacheKey: "A" and " A" are different
// token IDs, so flipping SpacePrefix after a cached lookup must not reuse ids.
func TestScoreServerSpacePrefixIsPartOfTheCacheKey(t *testing.T) {
	ids := map[string][]int{"A": {11}, "B": {12}, " A": {21}, " B": {22}}
	var got [][]int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tokenize":
			var req tokenizeRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(map[string]any{"tokens": ids[req.Text]})
		case "/v1/score":
			var req scoreRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			got = append(got, req.LabelTokenIDs)
			_ = json.NewEncoder(w).Encode(map[string]any{"scores": [][]float64{{0.8, 0.2}}})
		}
	}))
	defer srv.Close()

	s, err := NewScoreServer(ScoreConfig{BaseURL: srv.URL, Model: "m",
		Tokenizer: NewHTTPTokenizer(srv.URL, nil)})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	labels, options := []string{"A", "B"}, []string{"a", "b"}
	if _, err := s.Score(ctx, "p", labels, options); err != nil {
		t.Fatal(err)
	}
	s.SpacePrefix = true // same labels, different token surface
	if _, err := s.Score(ctx, "p", labels, options); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 score calls, got %d", len(got))
	}
	if got[0][0] != 11 || got[1][0] != 21 {
		t.Fatalf("cache must key on the space prefix: %v", got)
	}
}
