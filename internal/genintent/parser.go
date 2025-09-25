package genintent

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/firebase/genkit/go/genkit"
	"ndduy.dev/manglekit/internal/types"
)

// Config configures the Genkit-backed intent parser stage.
//
// The FlowName can be used to override the Genkit flow identifier when wiring
// the parser into a larger orchestration graph.
type Config struct {
	FlowName string `yaml:"flowName"`
}

// parser implements types.IntentParser using a Genkit flow.
type flowRunner interface {
	Run(ctx context.Context, input *types.QueryInput) (*types.IntentResult, error)
}

type parser struct {
	flow flowRunner
}

// New constructs a new parser backed by a Genkit flow.
func New(g *genkit.Genkit, cfg Config) (types.IntentParser, error) {
	if g == nil {
		return nil, errors.New("genkit instance is required")
	}
	flowName := cfg.FlowName
	if flowName == "" {
		flowName = "parse_intent_ner"
	}

	flow := genkit.DefineFlow(g, flowName, func(ctx context.Context, input *types.QueryInput) (*types.IntentResult, error) {
		if input == nil {
			return nil, errors.New("query input is required")
		}
		if strings.TrimSpace(input.Query) == "" {
			return &types.IntentResult{Intent: "unknown", Explanation: "empty query"}, nil
		}
		intent := detectIntent(input.Query)
		entities := extractEntities(input.Query)

		explanation := fmt.Sprintf("intent=%s, entities=%d", intent.Name, len(flattenEntityValues(entities)))
		return &types.IntentResult{
			Intent:      intent.Name,
			Confidence:  intent.Score,
			Entities:    entities,
			Explanation: explanation,
		}, nil
	})

	return &parser{flow: flow}, nil
}

// Parse executes the Genkit flow to obtain the intent and extracted entities.
func (p *parser) Parse(ctx context.Context, input *types.QueryInput) (*types.IntentResult, error) {
	if p == nil || p.flow == nil {
		return nil, errors.New("intent parser is not initialised")
	}
	return p.flow.Run(ctx, input)
}

type intentSignal struct {
	Name  string
	Score float64
}

func detectIntent(query string) intentSignal {
	lowered := strings.ToLower(query)
	if strings.Contains(lowered, "error") || strings.Contains(lowered, "fail") || strings.Contains(lowered, "crash") {
		return intentSignal{Name: "troubleshoot", Score: 0.8}
	}
	for _, prefix := range []string{"how", "what", "why", "when", "where"} {
		if strings.HasPrefix(lowered, prefix+" ") || strings.HasPrefix(lowered, prefix+"?") {
			return intentSignal{Name: "question", Score: 0.75}
		}
	}
	if strings.HasPrefix(lowered, "compare") || strings.Contains(lowered, "versus") {
		return intentSignal{Name: "comparison", Score: 0.7}
	}
	if strings.HasPrefix(lowered, "summarise") || strings.HasPrefix(lowered, "summarize") {
		return intentSignal{Name: "summarisation", Score: 0.7}
	}
	return intentSignal{Name: "informational", Score: 0.6}
}

var (
	versionRegexp  = regexp.MustCompile(`\b(v?\d+(?:\.\d+)+)\b`)
	ticketRegexp   = regexp.MustCompile(`\b[A-Z]{2,}-\d+\b`)
	osTokens       = []string{"windows", "macos", "linux", "ubuntu", "ios", "android"}
	fileExtensions = []string{"pdf", "csv", "docx", "pptx"}
)

func extractEntities(query string) map[string][]string {
	entities := make(map[string][]string)

	versions := versionRegexp.FindAllString(query, -1)
	if len(versions) > 0 {
		entities["version"] = dedupeStrings(versions)
	}

	tickets := ticketRegexp.FindAllString(query, -1)
	if len(tickets) > 0 {
		entities["ticket"] = dedupeStrings(tickets)
	}

	lowered := strings.ToLower(query)
	var products []string
	for _, token := range strings.Fields(query) {
		if isAllCaps(token) && len(token) > 2 && len(token) < 10 {
			products = append(products, strings.Trim(token, ",.;:"))
		}
	}
	if len(products) > 0 {
		entities["product"] = dedupeStrings(products)
	}

	for _, osToken := range osTokens {
		if strings.Contains(lowered, osToken) {
			entities["platform"] = append(entities["platform"], osToken)
		}
	}

	for _, ext := range fileExtensions {
		if strings.Contains(lowered, ext) {
			entities["artifact"] = append(entities["artifact"], ext)
		}
	}

	for key, vals := range entities {
		entities[key] = dedupeStrings(vals)
	}

	return entities
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		lower := strings.ToLower(v)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func flattenEntityValues(entities map[string][]string) []string {
	var flat []string
	for _, vals := range entities {
		flat = append(flat, vals...)
	}
	return flat
}

func isAllCaps(token string) bool {
	hasLetter := false
	for _, r := range token {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}
