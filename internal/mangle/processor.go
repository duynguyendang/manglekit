package mangle

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/mangle/analysis"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/engine"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/parse"
	"ndduy.dev/manglekit/internal/types"
)

// Config configures the Mangle processor.
type Config struct {
	RulesFile string `yaml:"rulesFile"`
	FactsFile string `yaml:"factsFile"`
}

type processor struct {
	cfg           Config
	programInfo   *analysis.ProgramInfo
	strata        []analysis.Nodeset
	predToStratum map[ast.PredicateSym]int
	baseFactStore factstore.SimpleInMemoryStore
}

// New creates a new Mangle processor from the supplied configuration.
func New(ctx context.Context, cfg Config) (types.Processor, error) {
	_ = ctx
	if cfg.RulesFile == "" {
		return nil, fmt.Errorf("mangle rules file must be provided")
	}
	if cfg.FactsFile == "" {
		return nil, fmt.Errorf("mangle facts file must be provided")
	}

	programInfo, strata, predToStratum, err := loadProgram(cfg.RulesFile)
	if err != nil {
		return nil, fmt.Errorf("load mangle program: %w", err)
	}

	baseStore, err := loadFacts(cfg.FactsFile)
	if err != nil {
		return nil, fmt.Errorf("load mangle facts: %w", err)
	}

	// Evaluate the program once to derive any static facts.
	if err := evaluate(programInfo, strata, predToStratum, baseStore); err != nil {
		return nil, fmt.Errorf("evaluate base mangle program: %w", err)
	}

	return &processor{
		cfg:           cfg,
		programInfo:   programInfo,
		strata:        strata,
		predToStratum: predToStratum,
		baseFactStore: baseStore,
	}, nil
}

// PreProcess normalizes the user query, enriches it with expansions using Mangle
// and returns the expanded query representation.
func (p *processor) PreProcess(input *types.QueryInput) (*types.ExpandedQuery, error) {
	normalized := strings.ToLower(strings.TrimSpace(input.Query))

	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(p.baseFactStore)

	tokens := tokenize(normalized)
	for _, token := range tokens {
		workingStore.Add(ast.NewAtom("query_token", ast.String(token)))
	}
	workingStore.Add(ast.NewAtom("raw_query", ast.String(input.Query)))
	workingStore.Add(ast.NewAtom("normalized_query", ast.String(normalized)))

	if input != nil && input.Intent != nil {
		for key, values := range input.Intent.Entities {
			for _, value := range values {
				workingStore.Add(ast.NewAtom("input_entity", ast.String(key), ast.String(strings.ToLower(value))))
			}
		}
	}

	if err := evaluate(p.programInfo, p.strata, p.predToStratum, workingStore); err != nil {
		return nil, fmt.Errorf("evaluate query program: %w", err)
	}

	expansions, err := collectStrings(workingStore, "expanded_query", 2)
	if err != nil {
		return nil, fmt.Errorf("collect expansions: %w", err)
	}
	filters, err := collectKeyValue(workingStore, "query_filter")
	if err != nil {
		return nil, fmt.Errorf("collect filters: %w", err)
	}

	normalizedTerms, err := collectStrings(workingStore, "normalized_term", 1)
	if err != nil {
		return nil, fmt.Errorf("collect normalized terms: %w", err)
	}

	stopwords, err := collectStopwords(workingStore, "stopword")
	if err != nil {
		return nil, fmt.Errorf("collect stopwords: %w", err)
	}

	normalizedTerms = filterStopwords(normalizedTerms, stopwords)

	mustTerms, shouldTerms, err := collectTermBuckets(workingStore, "term_constraint")
	if err != nil {
		return nil, fmt.Errorf("collect term constraints: %w", err)
	}

	mustTerms = dedupeStrings(filterStopwords(mustTerms, stopwords))
	shouldTerms = dedupeStrings(filterStopwords(shouldTerms, stopwords))
	shouldTerms = differenceStrings(shouldTerms, mustTerms)

	entities, err := collectEntities(workingStore, "query_entity")
	if err != nil {
		return nil, fmt.Errorf("collect entities: %w", err)
	}

	metadataConstraints, err := collectConstraints(workingStore, "query_constraint")
	if err != nil {
		return nil, fmt.Errorf("collect metadata constraints: %w", err)
	}

	explanation := "mangle expansions applied"
	if len(expansions) == 0 {
		explanation = "no mangle expansions found"
	}

	constraints := types.ConstraintSet{
		Terms: types.TermConstraints{
			Must:   mustTerms,
			Should: shouldTerms,
		},
		Metadata: metadataConstraints,
	}
	if visibility, ok := filters["visibility"]; ok {
		constraints.Visibility = visibility
	}

	return &types.ExpandedQuery{
		NormalizedQuery: normalized,
		NormalizedTerms: normalizedTerms,
		ExpansionTerms:  expansions,
		Entities:        entities,
		Filters:         filters,
		Constraints:     constraints,
		Explanation:     explanation,
	}, nil
}

// PostProcess enforces policy constraints on the retrieved chunks.
func (p *processor) PostProcess(chunks []*types.Chunk, ctx *types.Context) ([]*types.Chunk, *[]types.Explanation) {
	if len(chunks) == 0 {
		return nil, nil
	}

	var (
		filtered     []*types.Chunk
		explanations []types.Explanation
	)

	constraints := types.ConstraintSet{}
	if ctx != nil {
		constraints = ctx.Constraints
	}

	now := time.Now()
	userTenant := ""
	if ctx != nil && ctx.UserContext != nil {
		if tenant, ok := ctx.UserContext["tenant"].(string); ok {
			userTenant = strings.ToLower(strings.TrimSpace(tenant))
		}
	}

	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		if chunk.Metadata == nil {
			chunk.Metadata = map[string]any{}
		}

		visibility := strings.ToLower(asString(chunk.Metadata["visibility"]))
		if visibility == "" {
			visibility = "public"
		}

		if constraints.Visibility != "" && constraints.Visibility != "*" && visibility != constraints.Visibility {
			explanations = append(explanations, types.Explanation{
				Type:      "mangle-post",
				Rule:      "visibility_filter",
				Action:    "dropped",
				Reason:    fmt.Sprintf("requires visibility=%s but chunk visibility=%s", constraints.Visibility, visibility),
				Timestamp: now,
			})
			continue
		}

		chunkTenant := strings.ToLower(asString(chunk.Metadata["tenant"]))
		if chunkTenant == "" {
			chunkTenant = "*"
		}

		if userTenant != "" && chunkTenant != "*" && chunkTenant != userTenant {
			explanations = append(explanations, types.Explanation{
				Type:      "mangle-post",
				Rule:      "tenant_filter",
				Action:    "dropped",
				Reason:    fmt.Sprintf("chunk tenant=%s does not match user tenant=%s", chunkTenant, userTenant),
				Timestamp: now,
			})
			continue
		}

		metadataRejected := false
		for _, c := range constraints.Metadata {
			if c.Field == "" {
				continue
			}
			chunkVal := strings.ToLower(asString(chunk.Metadata[c.Field]))
			if chunkVal == "" {
				metadataRejected = true
				explanations = append(explanations, types.Explanation{
					Type:      "mangle-post",
					Rule:      "metadata_filter",
					Action:    "dropped",
					Reason:    fmt.Sprintf("missing required metadata field %s", c.Field),
					Timestamp: now,
				})
				break
			}
			match := false
			for _, v := range c.Values {
				if strings.EqualFold(chunkVal, v) || v == "*" {
					match = true
					break
				}
			}
			if !match {
				metadataRejected = true
				explanations = append(explanations, types.Explanation{
					Type:      "mangle-post",
					Rule:      "metadata_filter",
					Action:    "dropped",
					Reason:    fmt.Sprintf("metadata %s=%s rejected", c.Field, chunkVal),
					Timestamp: now,
				})
				break
			}
		}
		if metadataRejected {
			continue
		}

		if redact, ok := chunk.Metadata["redact"].(bool); ok && redact {
			redacted := redactChunk(chunk.Text)
			if redacted != chunk.Text {
				explanations = append(explanations, types.Explanation{
					Type:      "mangle-post",
					Rule:      "redaction",
					Action:    "modified",
					Reason:    fmt.Sprintf("sensitive content redacted from chunk %s", chunk.ID),
					Timestamp: now,
				})
				chunk.Text = redacted
			}
		}

		filtered = append(filtered, chunk)
	}

	if len(explanations) == 0 {
		return filtered, nil
	}
	return filtered, &explanations
}

func loadProgram(path string) (*analysis.ProgramInfo, []analysis.Nodeset, map[ast.PredicateSym]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	defer file.Close()

	unit, err := parse.Unit(file)
	if err != nil {
		return nil, nil, nil, err
	}

	programInfo, err := analysis.Analyze([]parse.SourceUnit{unit}, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return programInfo, strata, predToStratum, nil
}

func loadFacts(path string) (factstore.SimpleInMemoryStore, error) {
	store := factstore.NewSimpleInMemoryStore()

	files, err := resolveFactFiles(path)
	if err != nil {
		return store, err
	}
	if len(files) == 0 {
		return store, fmt.Errorf("no fact files found in %q", path)
	}

	for _, file := range files {
		if err := loadFactsFromFile(store, file); err != nil {
			return store, err
		}
	}

	return store, nil
}

func resolveFactFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.IsDir() {
			return collectFactFiles(path)
		}
		return []string{path}, nil
	case errors.Is(err, fs.ErrNotExist):
		if hasMeta(path) {
			matches, globErr := filepath.Glob(path)
			if globErr != nil {
				return nil, globErr
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no fact files matched %q", path)
			}
			var files []string
			for _, match := range matches {
				resolved, err := resolveFactFiles(match)
				if err != nil {
					return nil, err
				}
				files = append(files, resolved...)
			}
			sort.Strings(files)
			return files, nil
		}
		return nil, err
	default:
		return nil, err
	}
}

func collectFactFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isFactFile(d.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func hasMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

var factFileExtensions = map[string]struct{}{
	".db":    {},
	".data":  {},
	".dlog":  {},
	".dl":    {},
	".edb":   {},
	".fact":  {},
	".facts": {},
	".txt":   {},
}

func isFactFile(name string) bool {
	if strings.HasPrefix(name, ".") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(name))
	if _, ok := factFileExtensions[ext]; ok {
		return true
	}
	if ext == "" && strings.Contains(strings.ToLower(name), "fact") {
		return true
	}
	return false
}

func loadFactsFromFile(store factstore.SimpleInMemoryStore, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		atom, err := parse.Atom(line)
		if err != nil {
			return fmt.Errorf("parse fact %q (line %d in %s): %w", line, lineNum, path, err)
		}
		store.Add(atom)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read facts file %s: %w", path, err)
	}
	return nil
}

func evaluate(programInfo *analysis.ProgramInfo, strata []analysis.Nodeset, predToStratum map[ast.PredicateSym]int, store factstore.FactStore) error {
	_, err := engine.EvalStratifiedProgramWithStats(programInfo, strata, predToStratum, store)
	return err
}

func collectStrings(store factstore.ReadOnlyFactStore, predicate string, arity int) ([]string, error) {
	pred := ast.PredicateSym{Symbol: predicate, Arity: arity}
	results := make(map[string]struct{})

	err := store.GetFacts(ast.NewQuery(pred), func(atom ast.Atom) error {
		if len(atom.Args) < arity {
			return nil
		}
		arg := atom.Args[arity-1]
		constant, ok := arg.(ast.Constant)
		if !ok {
			return nil
		}
		value, err := constant.StringValue()
		if err != nil {
			// fallback for names
			if name, nerr := constant.NameValue(); nerr == nil {
				results[name] = struct{}{}
				return nil
			}
			return nil
		}
		results[value] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(results))
	for v := range results {
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

func collectKeyValue(store factstore.ReadOnlyFactStore, predicate string) (map[string]string, error) {
	pred := ast.PredicateSym{Symbol: predicate, Arity: 2}
	filters := make(map[string]string)
	err := store.GetFacts(ast.NewQuery(pred), func(atom ast.Atom) error {
		if len(atom.Args) != 2 {
			return nil
		}
		keyConst, ok := atom.Args[0].(ast.Constant)
		if !ok {
			return nil
		}
		valConst, ok := atom.Args[1].(ast.Constant)
		if !ok {
			return nil
		}
		key, err := constantToString(keyConst)
		if err != nil {
			return nil
		}
		val, err := constantToString(valConst)
		if err != nil {
			return nil
		}
		filters[key] = val
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filters, nil
}

func collectTermBuckets(store factstore.ReadOnlyFactStore, predicate string) (must []string, should []string, err error) {
	pred := ast.PredicateSym{Symbol: predicate, Arity: 2}
	err = store.GetFacts(ast.NewQuery(pred), func(atom ast.Atom) error {
		if len(atom.Args) != 2 {
			return nil
		}
		bucketConst, ok := atom.Args[0].(ast.Constant)
		if !ok {
			return nil
		}
		termConst, ok := atom.Args[1].(ast.Constant)
		if !ok {
			return nil
		}
		bucket, err := constantToString(bucketConst)
		if err != nil {
			return nil
		}
		term, err := constantToString(termConst)
		if err != nil {
			return nil
		}
		switch strings.ToLower(bucket) {
		case "must":
			must = append(must, term)
		case "should":
			should = append(should, term)
		}
		return nil
	})
	return must, should, err
}

func collectEntities(store factstore.ReadOnlyFactStore, predicate string) (map[string][]string, error) {
	pred := ast.PredicateSym{Symbol: predicate, Arity: 2}
	entities := make(map[string]map[string]struct{})
	err := store.GetFacts(ast.NewQuery(pred), func(atom ast.Atom) error {
		if len(atom.Args) != 2 {
			return nil
		}
		keyConst, ok := atom.Args[0].(ast.Constant)
		if !ok {
			return nil
		}
		valConst, ok := atom.Args[1].(ast.Constant)
		if !ok {
			return nil
		}
		key, err := constantToString(keyConst)
		if err != nil {
			return nil
		}
		value, err := constantToString(valConst)
		if err != nil {
			return nil
		}
		bucket, ok := entities[key]
		if !ok {
			bucket = make(map[string]struct{})
			entities[key] = bucket
		}
		bucket[value] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string, len(entities))
	for key, vals := range entities {
		var list []string
		for v := range vals {
			list = append(list, v)
		}
		sort.Strings(list)
		result[key] = list
	}
	return result, nil
}

func collectConstraints(store factstore.ReadOnlyFactStore, predicate string) ([]types.MetadataConstraint, error) {
	pred := ast.PredicateSym{Symbol: predicate, Arity: 3}
	constraints := make(map[string]*types.MetadataConstraint)
	err := store.GetFacts(ast.NewQuery(pred), func(atom ast.Atom) error {
		if len(atom.Args) != 3 {
			return nil
		}
		typeConst, ok := atom.Args[0].(ast.Constant)
		if !ok {
			return nil
		}
		fieldConst, ok := atom.Args[1].(ast.Constant)
		if !ok {
			return nil
		}
		valueConst, ok := atom.Args[2].(ast.Constant)
		if !ok {
			return nil
		}
		constraintType, err := constantToString(typeConst)
		if err != nil {
			return nil
		}
		if strings.ToLower(constraintType) != "metadata" {
			return nil
		}
		field, err := constantToString(fieldConst)
		if err != nil {
			return nil
		}
		value, err := constantToString(valueConst)
		if err != nil {
			return nil
		}
		key := strings.ToLower(field)
		constraint, ok := constraints[key]
		if !ok {
			constraint = &types.MetadataConstraint{
				Field:    key,
				Operator: "eq",
				Source:   "mangle",
			}
			constraints[key] = constraint
		}
		constraint.Values = append(constraint.Values, strings.ToLower(value))
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]types.MetadataConstraint, 0, len(constraints))
	for _, c := range constraints {
		c.Values = dedupeStrings(c.Values)
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Field < out[j].Field
	})
	return out, nil
}

func collectStopwords(store factstore.ReadOnlyFactStore, predicate string) (map[string]struct{}, error) {
	pred := ast.PredicateSym{Symbol: predicate, Arity: 1}
	stopwords := make(map[string]struct{})
	err := store.GetFacts(ast.NewQuery(pred), func(atom ast.Atom) error {
		if len(atom.Args) != 1 {
			return nil
		}
		constVal, ok := atom.Args[0].(ast.Constant)
		if !ok {
			return nil
		}
		value, err := constantToString(constVal)
		if err != nil {
			return nil
		}
		stopwords[strings.ToLower(value)] = struct{}{}
		return nil
	})
	return stopwords, err
}

func filterStopwords(values []string, stopwords map[string]struct{}) []string {
	if len(values) == 0 || len(stopwords) == 0 {
		return values
	}
	filtered := values[:0]
	for _, v := range values {
		key := strings.ToLower(strings.TrimSpace(v))
		if key == "" {
			continue
		}
		if _, ok := stopwords[key]; ok {
			continue
		}
		filtered = append(filtered, v)
	}
	return filtered
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	var result []string
	for _, v := range values {
		vv := strings.TrimSpace(v)
		if vv == "" {
			continue
		}
		key := strings.ToLower(vv)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, vv)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result
}

func differenceStrings(values []string, exclusions []string) []string {
	if len(values) == 0 || len(exclusions) == 0 {
		return values
	}
	exclude := make(map[string]struct{}, len(exclusions))
	for _, v := range exclusions {
		exclude[strings.ToLower(strings.TrimSpace(v))] = struct{}{}
	}
	out := values[:0]
	for _, v := range values {
		key := strings.ToLower(strings.TrimSpace(v))
		if key == "" {
			continue
		}
		if _, ok := exclude[key]; ok {
			continue
		}
		out = append(out, v)
	}
	return out
}

func asString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func redactChunk(text string) string {
	if text == "" {
		return text
	}
	var b strings.Builder
	for _, r := range text {
		if unicode.IsDigit(r) {
			b.WriteRune('#')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func constantToString(c ast.Constant) (string, error) {
	if v, err := c.StringValue(); err == nil {
		return v, nil
	}
	if v, err := c.NameValue(); err == nil {
		return v, nil
	}
	return "", fmt.Errorf("unsupported constant type: %v", c.Type)
}

func tokenize(query string) []string {
	if query == "" {
		return nil
	}
	fields := strings.FieldsFunc(query, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})

	seen := make(map[string]struct{})
	result := make([]string, 0, len(fields))
	for _, token := range fields {
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		result = append(result, token)
	}
	return result
}
