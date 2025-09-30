package mangle

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

	programInfo, strata, predToStratum, err := loadProgram(cfg.RulesFile)
	if err != nil {
		return nil, fmt.Errorf("load mangle program: %w", err)
	}

	baseStore := factstore.NewSimpleInMemoryStore()

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

	explanation := "mangle expansions applied"
	if len(expansions) == 0 {
		explanation = "no mangle expansions found"
	}

	return &types.ExpandedQuery{
		NormalizedQuery: normalized,
		ExpansionTerms:  expansions,
		Filters:         filters,
		Explanation:     explanation,
	}, nil
}

// PostProcess filters chunks based on Mangle rules for policy enforcement.
func (p *processor) PostProcess(chunks []*types.Chunk, ctx *types.Context) ([]*types.Chunk, *[]types.Explanation) {
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(p.baseFactStore)

	// Add facts about the user context.
	for key, value := range ctx.UserContext {
		workingStore.Add(ast.NewAtom("user_attribute", ast.String(key), ast.String(fmt.Sprintf("%v", value))))
	}

	// Add facts about the chunks.
	for _, chunk := range chunks {
		workingStore.Add(ast.NewAtom("doc_id", ast.String(chunk.DocID)))
		workingStore.Add(ast.NewAtom("doc_content", ast.String(chunk.DocID), ast.String(chunk.Text)))
		for key, value := range chunk.Metadata {
			if valueStr, ok := value.(string); ok {
				workingStore.Add(ast.NewAtom("doc_metadata", ast.String(chunk.DocID), ast.String(key), ast.String(valueStr)))
			}
		}
	}

	if err := evaluate(p.programInfo, p.strata, p.predToStratum, workingStore); err != nil {
		// In case of an error, we should probably deny all chunks.
		return nil, &[]types.Explanation{{
			Type:   "mangle-post",
			Action: "deny",
			Reason: fmt.Sprintf("Mangle evaluation error: %v", err),
		}}
	}

	denied, err := collectKeyValue(workingStore, "deny")
	if err != nil {
		// Log the error but don't fail the request.
		fmt.Fprintf(os.Stderr, "could not collect 'deny' facts: %v", err)
	}

	var allowedChunks []*types.Chunk
	var explanations []types.Explanation

	for _, chunk := range chunks {
		if reason, ok := denied[chunk.DocID]; ok {
			explanations = append(explanations, types.Explanation{
				Type:   "mangle-post",
				Rule:   "deny",
				Action: "deny",
				Reason: reason,
				DocID:  chunk.DocID,
			})
		} else {
			allowedChunks = append(allowedChunks, chunk)
		}
	}

	return allowedChunks, &explanations
}

func loadProgram(path string) (*analysis.ProgramInfo, []analysis.Nodeset, map[ast.PredicateSym]int, error) {
	files, err := resolveDlogFiles(path)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(files) == 0 {
		return nil, nil, nil, fmt.Errorf("no .dlog files found in %q", path)
	}

	var units []parse.SourceUnit
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not open rule file %s: %w", file, err)
		}
		defer f.Close()

		unit, err := parse.Unit(f)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not parse rule file %s: %w", file, err)
		}
		units = append(units, unit)
	}

	programInfo, err := analysis.Analyze(units, nil)
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

func resolveDlogFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.IsDir() {
			return collectDlogFiles(path)
		}
		if !strings.HasSuffix(info.Name(), ".dlog") {
			return nil, fmt.Errorf("rule file %q must have .dlog extension", path)
		}
		return []string{path}, nil
	case errors.Is(err, fs.ErrNotExist):
		if hasMeta(path) {
			matches, globErr := filepath.Glob(path)
			if globErr != nil {
				return nil, globErr
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no rule files matched %q", path)
			}
			var files []string
			for _, match := range matches {
				resolved, err := resolveDlogFiles(match)
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

func collectDlogFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".dlog") {
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