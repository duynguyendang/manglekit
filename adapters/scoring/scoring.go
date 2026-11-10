// Package scoring brings the fixed-choice classification pattern — score a
// small set of single-token options, normalize over exactly those labels, and
// let a policy decide what may act — into the kernel.
//
// It is deliberately a PREPARER, not an Action. Enforcement happens in the
// supervisor's pre-check, which evaluates the REQUEST envelope; a scoring step
// that lived inside an action would run after the gate had already said yes.
// So this package turns a prompt into a Decision, and a Decision into the
// envelope you hand to ExecuteByName:
//
//	dec, err := scorer.Score(ctx, prompt, []string{"A", "B", "C"}, teams)
//	th, err := scoring.ThresholdsFromPolicyScaled(ctx, client.Engine(), "threshold_pm", scoring.PerMille)
//	env := scoring.Request(req, dec, th)
//	out, err := client.ExecuteByName(ctx, "route_ticket", env)
//
// Division of labour (ADR-005): **Go measures and formats; the policy decides.**
// Request publishes the observation as per-mille integers — choice/1,
// score_pm/2 per class, score_lcb_pm/2 when bounded, score_source/1 — and the
// policy compares them with :lt/:ge against its own bars:
//
//	halt("Req", "low_confidence_send_to_review", "T1") :-
//	    action_operation("Req", "route_ticket"),
//	    choice(C), score_pm(C, P), class_bar(C, B), :lt(P, B).
//
// Mangle's comparison builtins are integer-only (a float operand fails the
// evaluation with "value 0.9 is not a number"), which is why probabilities are
// scaled; cross-fact comparison itself works fine, so the arithmetic — and
// therefore the decision — genuinely lives in the policy, and the Explain tree
// shows it ("740 < 800" as a [builtin] step).
//
// Thresholds/Flags still exist for reporting and for policies that prefer
// Go-computed booleans, but they are no longer what makes the gate decide.
//
// A label whose probability cannot be observed is handled with bounds, not
// guesses: a truncated top-K view yields sound LOWER bounds (Truncated, Lower,
// Missing) because every unlisted token has p <= p_K <= p_top, so truncation
// cannot change the winner. Only when NO option is observable does it become
// ErrLabelsUnobserved, published as unobserved_distribution so the policy
// refuses. Nothing here ever renormalizes over survivors and calls it
// confidence.
package scoring

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/duynguyendang/manglekit/core"
)

// ErrLabelsUnobserved means one or more option labels had no observable
// probability (e.g. a hosted API returned a truncated top_logprobs list).
// The distribution over the options is unknowable, so no Decision is produced.
var ErrLabelsUnobserved = errors.New("scoring: option labels not all observable")

// Decision is one fixed-choice read: the winning business option plus the
// probability the model assigned to every option.
type Decision struct {
	// Choice is a business option (the label's meaning), never a raw token.
	Choice string
	// Distribution maps option -> probability, normalized to sum to 1 over
	// exactly the options given. All other vocabulary mass is discarded —
	// that is what "restricted" softmax means.
	//
	// When Truncated is set these are sound LOWER bounds (see Truncated), so a
	// consumer that ignores the flag under-estimates confidence instead of
	// over-estimating it.
	Distribution map[string]float64
	// Margin is top minus runner-up; Entropy is Shannon entropy in nats.
	Margin  float64
	Entropy float64

	// Source names where the numbers came from, e.g. "score_server",
	// "chat_logprobs", "chat_logprobs_truncated", "mock". It is published to
	// the policy as score_source/1 so a policy can refuse to auto-route on
	// evidence it considers too soft — the tier of the evidence is a policy
	// input, not a Go default.
	Source string

	// Truncated reports that some option's probability was NOT observable
	// (a hosted top_logprobs list is a truncated view of the vocabulary).
	// Missing names them, and Lower carries the sound lower bound per option.
	Truncated bool
	Missing   []string
	Lower     map[string]float64
	// Upper is the survivor-normalized view (what you would get pretending the
	// unobserved options do not exist). Reporting only — never gate on it when
	// Truncated is set; gate on Distribution/Lower, which are sound bounds.
	Upper map[string]float64
}

// PerMille is the integer scale used when publishing probabilities to a policy:
// Mangle's :lt/:le/:gt/:ge are integer-only and reject floats loudly, so a
// probability becomes per-mille (0.740 -> 740). Per-mille matches the
// granularity of the thresholds people actually write (0.80, 0.70) and reads
// naturally in a proof tree ("740 < 800").
const PerMille = 1000

// ToPerMille scales a probability to the integer form a policy compares.
// It TRUNCATES toward zero: a borderline value rounds DOWN, so it fails the
// bar and lands in review/escalate rather than auto-routing. Fail-closed by
// construction — rounding half-up would let 0.7996 clear a 0.80 bar.
func ToPerMille(p float64) int64 {
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		return PerMille
	}
	return int64(p * PerMille)
}

// FromPerMille is the inverse, for reporting and calibration.
func FromPerMille(pm int64) float64 { return float64(pm) / PerMille }

// Prob returns the probability of an option (0 when absent).
func (d *Decision) Prob(option string) float64 { return d.Distribution[option] }

// Top returns the winning option's probability.
func (d *Decision) Top() float64 { return d.Distribution[d.Choice] }

// Scorer produces a Decision for one prompt. Implementations must return a
// distribution normalized over exactly the supplied labels.
type Scorer interface {
	// Score labels the prompt with labels (the single-token surfaces, in
	// option order) and reports the winner by business option name.
	Score(ctx context.Context, prompt string, labels, options []string) (*Decision, error)
}

// ---------------------------------------------------------------------------
// Shared math (exported so any backend — including a caller's own — builds the
// Decision the same way)
// ---------------------------------------------------------------------------

// Build turns raw probabilities (any positive scale; they are renormalized)
// into a Decision over options. probs may be probabilities from a server-side
// restricted softmax, or exp(logprob) from a hosted API — either way the
// result sums to 1 over exactly the options given.
//
// Ties break toward the FIRST option, so a Decision never depends on map
// iteration or on which duplicate happened to arrive last.
//
// Passing log-probabilities by mistake fails loudly ("sum to zero") instead of
// producing confident nonsense: every logprob is <= 0 and clamps to 0.
func Build(labels, options []string, probs []float64) (*Decision, error) {
	if len(labels) != len(options) || len(options) != len(probs) {
		return nil, fmt.Errorf("scoring: labels=%d options=%d probs=%d must match", len(labels), len(options), len(probs))
	}
	if len(options) == 0 {
		return nil, errors.New("scoring: no options to choose among")
	}
	// A repeated option would collapse the distribution (fewer keys than
	// options) and could report a Choice carrying another index's
	// probability; a repeated label would mis-map token ids server-side.
	// Both are caller bugs, so refuse rather than guess.
	if dup := firstDuplicate(options); dup != "" {
		return nil, fmt.Errorf("scoring: duplicate option %q — options must be distinct", dup)
	}
	if dup := firstDuplicate(labels); dup != "" {
		return nil, fmt.Errorf("scoring: duplicate label %q — labels must be distinct", dup)
	}
	total := 0.0
	for _, p := range probs {
		if p < 0 {
			p = 0 // clamp numerical dust
		}
		total += p
	}
	if total <= 0 {
		return nil, errors.New("scoring: probabilities sum to zero — nothing was observed (log-probabilities passed where probabilities were expected?)")
	}

	dist := make(map[string]float64, len(options))
	best := 0
	top, second := 0.0, 0.0
	for i, o := range options {
		p := probs[i]
		if p < 0 {
			p = 0
		}
		p /= total
		dist[o] = p
		if p > top { // strict: first index wins ties
			second, top = top, p
			best = i
		} else if p > second {
			second = p
		}
	}

	entropy := 0.0
	for _, p := range dist {
		if p > 0 {
			entropy -= p * math.Log(p)
		}
	}
	return &Decision{Choice: options[best], Distribution: dist, Margin: top - second, Entropy: entropy}, nil
}

// firstDuplicate returns the first value seen twice, or "" when all are unique.
func firstDuplicate(xs []string) string {
	seen := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			return x
		}
		seen[x] = struct{}{}
	}
	return ""
}

// Softmax converts log-probabilities to probabilities (numerically stable).
// Provided so backends do not each re-derive exp(logp).
func Softmax(logProbs []float64) []float64 {
	if len(logProbs) == 0 {
		return nil
	}
	max := logProbs[0]
	for _, v := range logProbs {
		if v > max {
			max = v
		}
	}
	out := make([]float64, len(logProbs))
	sum := 0.0
	for i, v := range logProbs {
		out[i] = math.Exp(v - max)
		sum += out[i]
	}
	if sum == 0 {
		return out
	}
	for i := range out {
		out[i] /= sum
	}
	return out
}

// ValidateSingleTokens rejects any label that is not exactly one token — the
// trap where " A" and "A" are different token IDs, and a two-token label makes
// reading a single logit meaningless.
//
// Scope: this guards the label_token_ids path ([ScoreServer]), which reads one
// logit per label and therefore cannot tolerate a split label. The hosted
// logprob path ([ChatScorer]) matches labels against returned token surfaces
// instead, so a split label simply never matches and surfaces as
// [ErrLabelsUnobserved] — safe, but reported differently.
func ValidateSingleTokens(ctx context.Context, tk Tokenizer, labels []string) error {
	for _, lb := range labels {
		n, err := tk.TokenLen(ctx, lb)
		if err != nil {
			return fmt.Errorf("tokenize label %q: %w", lb, err)
		}
		if n != 1 {
			return fmt.Errorf("label %q occupies %d tokens; fixed-choice scoring requires exactly 1", lb, n)
		}
	}
	return nil
}

// Tokenizer resolves text to its token count (and, for one-token text, its
// id) on the serving backend.
type Tokenizer interface {
	TokenLen(ctx context.Context, text string) (int, error)
}

// TokenIDResolver extends Tokenizer with single-token id lookup, which the
// label_token_ids style of scoring needs.
type TokenIDResolver interface {
	Tokenizer
	SingleTokenID(ctx context.Context, text string) (int, error)
}

// ---------------------------------------------------------------------------
// Thresholds — read FROM the policy, never hardcoded here
// ---------------------------------------------------------------------------

// Thresholds are the confidence bars a routing policy applies to a Decision.
//
// NOTE: since the per-class/comparison-in-policy design, these values are for
// REPORTING (logs, calibration, the deprecated Go-side Flags). The gate itself
// compares integers inside the policy with :lt/:ge — see Request and ADR-005.
type Thresholds struct {
	// Auto: the top probability must clear this to be auto-routed.
	Auto float64
	// Margin: the minimum lead over the runner-up (advisory by convention).
	Margin float64
	// Review: below this floor, escalate to a human rather than review-queue.
	Review float64
}

// PolicyQuery is the slice of the engine surface needed to read policy facts
// (satisfied by core.Evaluator / client.Engine()).
type PolicyQuery interface {
	Query(ctx context.Context, facts []string, query string) ([]map[string]string, error)
}

// ThresholdsFromPolicy reads threshold("<name>", V) facts from the ACTIVE
// policy program. After a client.ReloadPolicy these are the NEW numbers, which
// is the whole point: changing a bar needs no rebuild.
//
// It expects facts named auto, margin and review, and errors if a name is
// missing, non-numeric, or ambiguous — a silently absent threshold would turn
// into a silently permissive 0.0. For integer (per-mille) policies use
// ThresholdsFromPolicyScaled.
func ThresholdsFromPolicy(ctx context.Context, q PolicyQuery, predicate string) (Thresholds, error) {
	if q == nil {
		return Thresholds{}, errors.New("scoring: nil policy query")
	}
	if predicate == "" {
		predicate = "threshold"
	}
	get := func(name string) (float64, error) {
		sols, err := q.Query(ctx, nil, fmt.Sprintf(`%s("%s", V)`, predicate, name))
		if err != nil {
			return 0, fmt.Errorf("query %s(%q): %w", predicate, name, err)
		}
		if len(sols) == 0 {
			return 0, fmt.Errorf("policy defines no %s(%q)", predicate, name)
		}
		if len(sols) > 1 {
			return 0, fmt.Errorf("policy %s(%q) is ambiguous (%d values) — a reloaded program may still carry the old definition", predicate, name, len(sols))
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(sols[0]["V"]), 64)
		if err != nil {
			return 0, fmt.Errorf("policy %s(%q)=%q is not a number: %w", predicate, name, sols[0]["V"], err)
		}
		return v, nil
	}

	var th Thresholds
	var err error
	if th.Auto, err = get("auto"); err != nil {
		return th, err
	}
	if th.Margin, err = get("margin"); err != nil {
		return th, err
	}
	if th.Review, err = get("review"); err != nil {
		return th, err
	}
	if th.Review > th.Auto {
		return th, fmt.Errorf("policy review floor %.2f exceeds auto bar %.2f — every case would escalate", th.Review, th.Auto)
	}
	return th, nil
}

// Flags compares a Decision against the thresholds and returns the boolean
// meta flags a policy pattern-matches on. Only triggered flags are present:
// policy rules key on fact EXISTENCE, which is how the engine expresses
// "the comparison was true".
//
// A nil decision (an unobservable distribution) yields just the
// unobserved_distribution flag — the policy, not Go, then refuses.
func (t Thresholds) Flags(d *Decision) map[string]string {
	if d == nil {
		return map[string]string{"unobserved_distribution": "true"}
	}
	top := d.Top()
	flags := map[string]string{}
	if top < t.Review {
		flags["below_review"] = "true"
	}
	if top < t.Auto {
		flags["below_auto"] = "true"
	}
	if top >= t.Auto && d.Margin < t.Margin {
		flags["thin_margin"] = "true"
	}
	return flags
}

// ---------------------------------------------------------------------------
// Envelope construction — the bridge into the supervised call
// ---------------------------------------------------------------------------

// Request builds the envelope to pass to client.ExecuteByName for a routing
// call: the meta flags derived from the policy thresholds, plus the model's
// own numbers as explicit facts so a proof tree can ground them
// (confidence / margin_value / entropy_value).
//
// payload may be the action's input value OR an already-built core.Envelope —
// ExecuteByName accepts both, so nesting an envelope inside an envelope would
// silently break the action's type assertion. When payload is an Envelope its
// facts, labels and metadata are preserved and the scoring facts/flags are
// merged in (scoring flags win on key collision).
//
// A nil decision marks the distribution unobservable and publishes no numbers:
// the policy then decides, via the unobserved_distribution flag, that the
// request must not be auto-routed.
func Request(payload any, d *Decision, t Thresholds) core.Envelope {
	var env core.Envelope
	if existing, ok := payload.(core.Envelope); ok {
		env = existing
		if env.Metadata == nil {
			env.Metadata = map[string]any{}
		}
	} else {
		env = core.NewEnvelope(payload)
	}

	for k, v := range t.Flags(d) {
		env.Metadata[k] = v
	}
	if d == nil {
		return env
	}
	env.Metadata["choice"] = d.Choice
	if d.Source != "" {
		env.Metadata["score_source"] = d.Source
	}
	env.Facts = append(env.Facts,
		fmt.Sprintf("confidence(%q).", formatProb(d.Top())),
		fmt.Sprintf("margin_value(%q).", formatProb(d.Margin)),
		fmt.Sprintf("entropy_value(%q).", formatProb(d.Entropy)),
		fmt.Sprintf("choice(%q).", d.Choice),
		fmt.Sprintf("margin_value_pm(%d).", ToPerMille(d.Margin)),
	)
	if d.Source != "" {
		env.Facts = append(env.Facts, fmt.Sprintf("score_source(%q).", d.Source))
	}
	if d.Truncated {
		env.Facts = append(env.Facts, `truncated_top_k("true").`)
		env.Metadata["truncated_top_k"] = "true"
	}

	// Per-class integer scores: this is what a policy compares with :lt/:ge,
	// so the DECISION arithmetic lives in the policy rather than in Go flags.
	// Sorted for deterministic fact order (map iteration is not), which keeps
	// proof trees and golden tests stable.
	for _, opt := range d.sortedOptions() {
		env.Facts = append(env.Facts, fmt.Sprintf("score_pm(%q, %d).", opt, ToPerMille(d.Distribution[opt])))
	}
	// Sound lower bounds when part of the vocabulary was unobservable. A
	// policy that gates on score_lcb_pm can never be over-confident about a
	// truncated read; one that gates on score_pm still gets the upper view.
	if d.Truncated {
		for _, opt := range d.sortedOptions() {
			env.Facts = append(env.Facts, fmt.Sprintf("score_lcb_pm(%q, %d).", opt, ToPerMille(d.Lower[opt])))
		}
	}
	return env
}

// RequestNoScore builds the envelope for a decision that has a CHOICE but no
// distribution — the generation lane: the model wrote a letter and that is all
// the evidence there is. It deliberately publishes no number, because a
// fabricated 1.0 would let any confidence bar pass and the proof tree would
// look authoritative about a value nobody measured. The policy gates on the
// ABSENCE of a score instead:
//
//	halt("Req", "no_score_present_review", "T1") :-
//	    action_operation("Req", "route_ticket"),
//	    meta("no_score_present", "true").
func RequestNoScore(payload any, choice string) core.Envelope {
	var env core.Envelope
	if existing, ok := payload.(core.Envelope); ok {
		env = existing
		if env.Metadata == nil {
			env.Metadata = map[string]any{}
		}
	} else {
		env = core.NewEnvelope(payload)
	}
	env.Metadata["no_score_present"] = "true"
	if choice != "" {
		env.Metadata["choice"] = choice
		env.Facts = append(env.Facts, fmt.Sprintf("choice(%q).", choice))
	}
	return env
}

// ThresholdsFromPolicyScaled reads integer threshold facts (e.g.
// threshold_pm("auto", 800)) and converts them to probabilities using scale.
// It is the integer counterpart of ThresholdsFromPolicy: Mangle's comparison
// builtins are integer-only, so a policy that does its own :lt comparisons
// holds integers, while Go still wants floats for reporting and calibration.
//
// The same distrust applies: missing, non-numeric, inverted or ambiguous bars
// are hard errors.
func ThresholdsFromPolicyScaled(ctx context.Context, q PolicyQuery, predicate string, scale int) (Thresholds, error) {
	if scale <= 0 {
		return Thresholds{}, fmt.Errorf("scoring: scale must be positive, got %d", scale)
	}
	raw, err := ThresholdsFromPolicyInts(ctx, q, predicate)
	if err != nil {
		return Thresholds{}, err
	}
	return Thresholds{
		Auto:   float64(raw.Auto) / float64(scale),
		Margin: float64(raw.Margin) / float64(scale),
		Review: float64(raw.Review) / float64(scale),
	}, nil
}

// IntThresholds are threshold bars in the policy's own integer scale.
type IntThresholds struct {
	Auto   int64
	Margin int64
	Review int64
}

// ThresholdsFromPolicyInts reads integer bars without rescaling — useful when
// the caller wants to compare in the policy's own units.
func ThresholdsFromPolicyInts(ctx context.Context, q PolicyQuery, predicate string) (IntThresholds, error) {
	if q == nil {
		return IntThresholds{}, errors.New("scoring: nil policy query")
	}
	if predicate == "" {
		predicate = "threshold_pm"
	}
	get := func(name string) (int64, error) {
		sols, err := q.Query(ctx, nil, fmt.Sprintf(`%s("%s", V)`, predicate, name))
		if err != nil {
			return 0, fmt.Errorf("query %s(%q): %w", predicate, name, err)
		}
		if len(sols) == 0 {
			return 0, fmt.Errorf("policy defines no %s(%q)", predicate, name)
		}
		if len(sols) > 1 {
			return 0, fmt.Errorf("policy %s(%q) is ambiguous (%d values) — a reloaded program may still carry the old definition", predicate, name, len(sols))
		}
		v, err := strconv.ParseInt(strings.TrimSpace(sols[0]["V"]), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("policy %s(%q)=%q is not an integer: %w", predicate, name, sols[0]["V"], err)
		}
		return v, nil
	}
	var th IntThresholds
	var err error
	if th.Auto, err = get("auto"); err != nil {
		return th, err
	}
	if th.Margin, err = get("margin"); err != nil {
		return th, err
	}
	if th.Review, err = get("review"); err != nil {
		return th, err
	}
	if th.Review > th.Auto {
		return th, fmt.Errorf("policy review floor %d exceeds auto bar %d — every case would escalate", th.Review, th.Auto)
	}
	return th, nil
}

func formatProb(p float64) string { return strconv.FormatFloat(p, 'f', 3, 64) }

// sortedOptions returns the distribution keys by descending probability with
// alphabetical tie-break — stable ordering for logs and tables (map iteration
// is not). Unexported on purpose: reporting is application-layer, and keeping
// it out of the public surface is the ADR-002 discipline.
func (d *Decision) sortedOptions() []string {
	keys := make([]string, 0, len(d.Distribution))
	for k := range d.Distribution {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if d.Distribution[keys[i]] == d.Distribution[keys[j]] {
			return keys[i] < keys[j]
		}
		return d.Distribution[keys[i]] > d.Distribution[keys[j]]
	})
	return keys
}
