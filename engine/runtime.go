package engine

import (
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
)

// MangleRuntime encapsulates the Google Mangle Datalog engine and provides
// high-level operations for loading rules and executing queries.
type MangleRuntime struct {
	// programInfo contains the analyzed and stratified Datalog program
	programInfo *analysis.ProgramInfo
	// strata contains the stratification layers of the program for safe evaluation
	strata []analysis.Nodeset
	// predToStratum maps predicates to their strata for evaluation ordering
	predToStratum map[ast.PredicateSym]int
	// baseFactStore holds the base facts loaded from files
	baseFactStore factstore.SimpleInMemoryStore
}

// NewMangleRuntime creates a new MangleRuntime with an empty program and fact store.
func NewMangleRuntime() *MangleRuntime {
	return &MangleRuntime{
		predToStratum: make(map[ast.PredicateSym]int),
		baseFactStore: factstore.NewSimpleInMemoryStore(),
	}
}

// Load loads Datalog rules from the given file path or directory.
// It supports:
// - Single .dlog rule files
// - Single .facts/.edb/.data fact files
// - Directories (recursively)
// - Glob patterns (e.g., "rules/*.dlog")
//
// After loading, the program is analyzed and stratified for evaluation.
func (r *MangleRuntime) Load(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Resolve the path into a list of actual files
	files, err := resolveFiles(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Separate into rule files and fact files
	var ruleFiles, factFiles []string
	for _, file := range files {
		switch {
		case isRuleFile(file):
			ruleFiles = append(ruleFiles, file)
		case isFactFile(file):
			factFiles = append(factFiles, file)
		}
	}

	if len(ruleFiles) == 0 && len(factFiles) == 0 {
		return fmt.Errorf("no .dlog or fact files found")
	}

	// Parse rule files to build the program
	var units []parse.SourceUnit
	edbDeclarations := make(map[ast.PredicateSym]ast.Decl)

	for _, ruleFile := range ruleFiles {
		unit, err := parseRuleFile(ruleFile)
		if err != nil {
			return fmt.Errorf("failed to parse rule file %s: %w", ruleFile, err)
		}
		units = append(units, unit)
	}

	// Parse fact files and collect facts
	var initialFacts []ast.Atom
	for _, factFile := range factFiles {
		unit, err := parseRuleFile(factFile)
		if err != nil {
			return fmt.Errorf("failed to parse fact file %s: %w", factFile, err)
		}
		// A fact is a clause with an empty body (no premises).
		for _, clause := range unit.Clauses {
			if len(clause.Premises) == 0 {
				initialFacts = append(initialFacts, clause.Head)
			}
		}
	}

	// Add initial facts to base store
	for _, fact := range initialFacts {
		r.baseFactStore.Add(fact)
	}

	// Analyze the program
	programInfo, err := analysis.Analyze(units, edbDeclarations)
	if err != nil {
		return fmt.Errorf("failed to analyze program: %w", err)
	}

	// Stratify the program
	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return fmt.Errorf("failed to stratify program: %w", err)
	}

	r.programInfo = programInfo
	r.strata = strata
	r.predToStratum = predToStratum

	// Perform initial evaluation with base facts
	if err := r.evaluate(r.baseFactStore); err != nil {
		return fmt.Errorf("failed to evaluate base program: %w", err)
	}

	return nil
}

// LoadFacts adds a list of Datalog fact strings to the base fact store.
// These facts become part of the static knowledge base.
func (r *MangleRuntime) LoadFacts(facts []string) error {
	for _, factStr := range facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			return fmt.Errorf("failed to parse fact '%s': %w", factStr, err)
		}
		r.baseFactStore.Add(atom)
	}

	// If a program is already loaded, re-evaluate to update derived facts
	if r.programInfo != nil {
		if err := r.evaluate(r.baseFactStore); err != nil {
			return fmt.Errorf("failed to evaluate program with new facts: %w", err)
		}
	}
	return nil
}

// LoadFromString loads a Datalog rule from a string.
// The rule should be a complete Datalog clause (e.g., "deny(Req) :- amount(Req, X), fn:gt(X, 100).").
func (r *MangleRuntime) LoadFromString(rule string) error {
	if rule == "" {
		return fmt.Errorf("rule cannot be empty")
	}

	// Parse the rule
	clause, err := parse.Clause(rule)
	if err != nil {
		return fmt.Errorf("failed to parse rule: %w", err)
	}

	// Create a source unit with the single clause
	sourceUnit := parse.SourceUnit{Clauses: []ast.Clause{clause}}
	edbDeclarations := make(map[ast.PredicateSym]ast.Decl)

	// Analyze the program
	programInfo, err := analysis.Analyze([]parse.SourceUnit{sourceUnit}, edbDeclarations)
	if err != nil {
		return fmt.Errorf("failed to analyze rule: %w", err)
	}

	// Stratify the program
	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return fmt.Errorf("failed to stratify rule: %w", err)
	}

	r.programInfo = programInfo
	r.strata = strata
	r.predToStratum = predToStratum

	// Perform initial evaluation with base facts (may be empty)
	if err := r.evaluate(r.baseFactStore); err != nil {
		return fmt.Errorf("failed to evaluate program with rule: %w", err)
	}

	return nil
}

// ExecuteQuery executes a Datalog query against the given facts.
// It returns true if the query is satisfied, false otherwise.
//
// The query is expected to be an atom (e.g., "deny(Req)").
// For more complex queries, use QueryWithSolutions.
func (r *MangleRuntime) ExecuteQuery(facts []ast.Atom, queryStr string) (bool, error) {
	if r.programInfo == nil {
		return false, fmt.Errorf("runtime not initialized; call Load or LoadFromString first")
	}

	// Create a working fact store and merge base facts
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)

	// Add the provided facts
	for _, fact := range facts {
		workingStore.Add(fact)
	}

	// Evaluate the program with the combined facts
	if err := r.evaluate(workingStore); err != nil {
		return false, fmt.Errorf("failed to evaluate query: %w", err)
	}

	// Parse the query atom
	queryAtom, err := parse.Atom(queryStr)
	if err != nil {
		return false, fmt.Errorf("failed to parse query '%s': %w", queryStr, err)
	}

	// Check if the query atom exists in the fact store
	found := workingStore.Contains(queryAtom)
	return found, nil
}

// QueryWithSolutions executes a query and returns all matching solutions.
// The callback is invoked for each solution found.
// Solutions are represented as maps of variable names to values.
func (r *MangleRuntime) QueryWithSolutions(facts []ast.Atom, queryStr string, onSolution func(map[string]any) error) error {
	if r.programInfo == nil {
		return fmt.Errorf("runtime not initialized; call Load or LoadFromString first")
	}

	// Create a working fact store and merge base facts
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)

	// Add the provided facts
	for _, fact := range facts {
		workingStore.Add(fact)
	}

	// Evaluate the program with the combined facts
	if err := r.evaluate(workingStore); err != nil {
		return fmt.Errorf("failed to evaluate query: %w", err)
	}

	// Parse the query atom
	queryAtom, err := parse.Atom(queryStr)
	if err != nil {
		return fmt.Errorf("failed to parse query '%s': %w", queryStr, err)
	}

	q := ast.NewQuery(queryAtom.Predicate)

	return workingStore.GetFacts(q, func(factAtom ast.Atom) error {
		// Manual unification of the query atom against the fact atom.
		match := true
		if len(factAtom.Args) != len(queryAtom.Args) {
			return nil
		}
		for i, queryArg := range queryAtom.Args {
			if _, isVar := queryArg.(ast.Variable); isVar {
				continue
			}
			if !queryArg.Equals(factAtom.Args[i]) {
				match = false
				break
			}
		}
		if !match {
			return nil
		}

		// Build solution map
		solution := make(map[string]any)
		for i, arg := range queryAtom.Args {
			if v, ok := arg.(ast.Variable); ok {
				term := factAtom.Args[i]
				var val string
				if c, ok := term.(ast.Constant); ok {
					var err error
					val, err = constantToString(c)
					if err != nil {
						return fmt.Errorf("could not convert solution term '%v' to string: %w", term, err)
					}
				} else {
					return fmt.Errorf("expected a constant in query solution, but got %T", term)
				}
				solution[v.Symbol] = val
			}
		}
		return onSolution(solution)
	})
}

// evaluate performs stratified evaluation of the program with the given fact store.
func (r *MangleRuntime) evaluate(store factstore.FactStore) error {
	if r.programInfo == nil || r.strata == nil || r.predToStratum == nil {
		return fmt.Errorf("runtime not initialized")
	}

	_, err := engine.EvalStratifiedProgramWithStats(r.programInfo, r.strata, r.predToStratum, store)
	if err != nil {
		return fmt.Errorf("mangle engine evaluation failed: %w", err)
	}
	return nil
}

// --- Helper Functions ---

func isRuleFile(p string) bool {
	return strings.HasSuffix(p, ".dlog") || strings.HasSuffix(p, ".dl")
}

func isFactFile(p string) bool {
	return strings.HasSuffix(p, ".facts") ||
		strings.HasSuffix(p, ".fact") ||
		strings.HasSuffix(p, ".edb") ||
		strings.HasSuffix(p, ".data")
}

// parseRuleFile parses a single .dlog rule file into a source unit.
func parseRuleFile(file string) (parse.SourceUnit, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return parse.SourceUnit{}, fmt.Errorf("could not open rule file %s: %w", file, err)
	}

	// Strip UTF-8 BOM if present
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		b = b[3:]
	}

	// Normalize newlines and drop lines that are just "."
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	lines := strings.Split(s, "\n")
	kept := lines[:0]
	for _, ln := range lines {
		trimLn := strings.TrimSpace(ln)
		// Skip empty lines, lines that are just ".", and comments
		if trimLn == "." || strings.HasPrefix(trimLn, "%") || strings.HasPrefix(trimLn, "//") {
			continue
		}
		kept = append(kept, ln)
	}
	cleaned := strings.Join(kept, "\n")

	unit, err := parse.Unit(strings.NewReader(cleaned))
	if err != nil {
		return parse.SourceUnit{}, fmt.Errorf("could not parse rule file %s: %w", file, err)
	}
	return unit, nil
}

// resolveFiles resolves a path (file, directory, or glob) into a list of file paths.
func resolveFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.IsDir() {
			return collectFiles(path)
		}
		return []string{path}, nil
	case errors.Is(err, fs.ErrNotExist):
		if hasMeta(path) {
			matches, globErr := filepath.Glob(path)
			if globErr != nil {
				return nil, fmt.Errorf("path globbing failed for %q: %w", path, globErr)
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no files matched %q", path)
			}
			var files []string
			for _, match := range matches {
				resolved, err := resolveFiles(match)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve glob match %q: %w", match, err)
				}
				files = append(files, resolved...)
			}
			sort.Strings(files)
			return files, nil
		}
		return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
	default:
		return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
	}
}

// collectFiles recursively collects all files from a directory.
func collectFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %q: %w", root, err)
	}
	sort.Strings(files)
	return files, nil
}

// hasMeta checks if a path contains glob metacharacters.
func hasMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

// constantToString converts a Mangle constant to a string.
func constantToString(c ast.Constant) (string, error) {
	if v, err := c.StringValue(); err == nil {
		return v, nil
	}
	if v, err := c.NameValue(); err == nil {
		return v, nil
	}
	return "", fmt.Errorf("unsupported constant type: %v", c.Type)
}
