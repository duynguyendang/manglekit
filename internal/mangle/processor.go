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

// PostProcess currently returns the chunks unchanged. It could be extended to
// leverage Mangle rules for policy enforcement or redaction.
func (p *processor) PostProcess(chunks []*types.Chunk, ctx *types.Context) ([]*types.Chunk, *[]types.Explanation) {
	return chunks, nil
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
