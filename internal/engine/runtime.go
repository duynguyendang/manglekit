package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"codeberg.org/TauCeti/mangle-go/analysis"
	"codeberg.org/TauCeti/mangle-go/ast"
	"codeberg.org/TauCeti/mangle-go/engine"
	"codeberg.org/TauCeti/mangle-go/factstore"
	"codeberg.org/TauCeti/mangle-go/parse"
)

// ExternalPredicate is a simple function type for external predicates.
// It takes input values and returns output values or an error.
type ExternalPredicate func(ctx context.Context, inputs []any) ([][]any, error)

// ExternalPredicateRegistry holds external predicates that can be called from Datalog rules.
type ExternalPredicateRegistry struct {
	mu         sync.RWMutex
	predicates map[ast.PredicateSym]ExternalPredicate
}

// NewExternalPredicateRegistry creates a new registry for external predicates.
func NewExternalPredicateRegistry() *ExternalPredicateRegistry {
	return &ExternalPredicateRegistry{
		predicates: make(map[ast.PredicateSym]ExternalPredicate),
	}
}

// Register adds an external predicate to the registry.
func (r *ExternalPredicateRegistry) Register(name string, fn ExternalPredicate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if fn == nil {
		return fmt.Errorf("external predicate %s cannot be nil", name)
	}
	r.predicates[ast.PredicateSym{Symbol: name}] = fn
	return nil
}

// Get retrieves an external predicate by name.
func (r *ExternalPredicateRegistry) Get(name string) (ExternalPredicate, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, ok := r.predicates[ast.PredicateSym{Symbol: name}]
	return fn, ok
}

// List returns all registered external predicates.
func (r *ExternalPredicateRegistry) List() map[ast.PredicateSym]ExternalPredicate {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[ast.PredicateSym]ExternalPredicate)
	for k, v := range r.predicates {
		result[k] = v
	}
	return result
}

// Count returns the number of registered external predicates.
func (r *ExternalPredicateRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.predicates)
}

// MangleRuntime encapsulates the Google Mangle Datalog engine.
type MangleRuntime struct {
	mu sync.RWMutex

	// State fields (protected by mu)
	programInfo   *analysis.ProgramInfo
	strata        []analysis.Nodeset
	predToStratum map[ast.PredicateSym]int
	baseFactStore factstore.SimpleInMemoryStore
	ruleUnits     []parse.SourceUnit
	ready         bool // Flag to indicate if the runtime is initialized

	// Temporal store for time-based facts (optional)
	temporalStore   *factstore.TemporalStore
	temporalEnabled bool
	evaluationTime  time.Time

	// External predicates for calling Go functions from Datalog rules
	externalPreds *ExternalPredicateRegistry
}

// NewMangleRuntime initializes a new, empty MangleRuntime.
func NewMangleRuntime() *MangleRuntime {
	return &MangleRuntime{
		predToStratum: make(map[ast.PredicateSym]int),
		baseFactStore: factstore.NewSimpleInMemoryStore(),
		ready:         false,
		externalPreds: NewExternalPredicateRegistry(),
	}
}

// RegisterExternalPredicate adds an external predicate that can be called from Datalog rules.
// The predicate can then be used in policies like:
//
//	http_get(URL, Status) :- ...
//
// When the predicate is called, the external function receives the input arguments
// and returns output values that are added as facts to the engine.
//
// Parameters:
//   - name: The predicate name (e.g., "http_get", "time_now")
//   - fn: The function to call
//
// Returns an error if the predicate name is empty or the function is nil.
func (r *MangleRuntime) RegisterExternalPredicate(name string, fn ExternalPredicate) error {
	if name == "" {
		return fmt.Errorf("external predicate name cannot be empty")
	}
	return r.externalPreds.Register(name, fn)
}

// ExternalPredicates returns the external predicate registry for inspection.
func (r *MangleRuntime) ExternalPredicates() *ExternalPredicateRegistry {
	return r.externalPreds
}

// EnableTemporal enables temporal reasoning support.
// This must be called before loading any policies.
func (r *MangleRuntime) EnableTemporal() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.temporalStore = factstore.NewTemporalStore()
	r.temporalEnabled = true
}

// IsTemporalEnabled returns whether temporal reasoning is enabled.
func (r *MangleRuntime) IsTemporalEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.temporalEnabled
}

// AddTemporalFact adds a fact with a time interval.
// The fact is valid during the specified time range.
func (r *MangleRuntime) AddTemporalFact(factString string, start, end time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.temporalEnabled {
		return fmt.Errorf("temporal reasoning not enabled. Call EnableTemporal() first")
	}

	atom, err := parse.Atom(factString)
	if err != nil {
		return fmt.Errorf("failed to parse fact '%s': %w", factString, err)
	}

	interval := ast.TimeInterval(start, end)
	_, err = r.temporalStore.Add(atom, interval)
	return err
}

// AddTemporalFactAt adds a fact valid at a specific point in time.
func (r *MangleRuntime) AddTemporalFactAt(factString string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.temporalEnabled {
		return fmt.Errorf("temporal reasoning not enabled. Call EnableTemporal() first")
	}

	atom, err := parse.Atom(factString)
	if err != nil {
		return fmt.Errorf("failed to parse fact '%s': %w", factString, err)
	}

	interval := ast.NewPointInterval(at)
	_, err = r.temporalStore.Add(atom, interval)
	return err
}

// SetEvaluationTime sets the evaluation time for temporal queries.
// This is the "now" time used when evaluating temporal predicates like <-[0d, 30d].
func (r *MangleRuntime) SetEvaluationTime(t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.evaluationTime = t
}

// GetEvaluationTime returns the current evaluation time.
func (r *MangleRuntime) GetEvaluationTime() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.evaluationTime
}

// AddTemporalFactInPast adds a fact that was true for a duration in the past.
// The fact ends at the evaluation time if set, otherwise ends at now.
func (r *MangleRuntime) AddTemporalFactInPast(factString string, duration time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.temporalEnabled {
		return fmt.Errorf("temporal reasoning not enabled. Call EnableTemporal() first")
	}

	atom, err := parse.Atom(factString)
	if err != nil {
		return fmt.Errorf("failed to parse fact '%s': %w", factString, err)
	}

	endTime := r.evaluationTime
	if endTime.IsZero() {
		endTime = time.Now()
	}
	startTime := endTime.Add(-duration)

	interval := ast.TimeInterval(startTime, endTime)
	_, err = r.temporalStore.Add(atom, interval)
	return err
}

// AddTemporalFactInFuture adds a fact that will be true for a duration in the future.
// The fact starts at the evaluation time if set, otherwise starts at now.
func (r *MangleRuntime) AddTemporalFactInFuture(factString string, duration time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.temporalEnabled {
		return fmt.Errorf("temporal reasoning not enabled. Call EnableTemporal() first")
	}

	atom, err := parse.Atom(factString)
	if err != nil {
		return fmt.Errorf("failed to parse fact '%s': %w", factString, err)
	}

	startTime := r.evaluationTime
	if startTime.IsZero() {
		startTime = time.Now()
	}
	endTime := startTime.Add(duration)

	interval := ast.TimeInterval(startTime, endTime)
	_, err = r.temporalStore.Add(atom, interval)
	return err
}

// QueryTemporalFactsAt queries facts that are valid at a specific time.
func (r *MangleRuntime) QueryTemporalFactsAt(factPattern string, at time.Time) ([]string, error) {
	store := r.GetTemporalStore()
	if store == nil {
		return nil, fmt.Errorf("temporal store not available")
	}

	queryAtom, err := parse.Atom(factPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query '%s': %w", factPattern, err)
	}

	var results []string
	err = store.GetFactsAt(queryAtom, at, func(tf factstore.TemporalFact) error {
		results = append(results, tf.Atom.String())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return results, nil
}

// QueryTemporalFactsDuring queries facts that are valid during a time interval.
func (r *MangleRuntime) QueryTemporalFactsDuring(factPattern string, start, end time.Time) ([]string, error) {
	store := r.GetTemporalStore()
	if store == nil {
		return nil, fmt.Errorf("temporal store not available")
	}

	queryAtom, err := parse.Atom(factPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query '%s': %w", factPattern, err)
	}

	interval := ast.TimeInterval(start, end)
	var results []string
	err = store.GetFactsDuring(queryAtom, interval, func(tf factstore.TemporalFact) error {
		results = append(results, tf.Atom.String())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return results, nil
}

// ContainsTemporalFact checks if a fact is valid at the evaluation time.
func (r *MangleRuntime) ContainsTemporalFact(factString string) (bool, error) {
	store := r.GetTemporalStore()
	if store == nil {
		return false, fmt.Errorf("temporal store not available")
	}

	atom, err := parse.Atom(factString)
	if err != nil {
		return false, fmt.Errorf("failed to parse fact '%s': %w", factString, err)
	}

	evalTime := r.GetEvaluationTime()
	if evalTime.IsZero() {
		evalTime = time.Now()
	}

	return store.ContainsAt(atom, evalTime), nil
}

// AddEternalFact adds a fact that's always true (no temporal bounds).
func (r *MangleRuntime) AddEternalFact(factString string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	atom, err := parse.Atom(factString)
	if err != nil {
		return fmt.Errorf("failed to parse fact '%s': %w", factString, err)
	}

	_, err = r.temporalStore.AddEternal(atom)
	return err
}

// GetTemporalStore returns the temporal store for advanced operations.
func (r *MangleRuntime) GetTemporalStore() *factstore.TemporalStore {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.temporalStore
}

// Load loads Datalog rules and facts from the specified path.
// CRITICAL CHANGE: This REPLACES the current program state.
func (r *MangleRuntime) Load(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// 1. Resolve files (I/O) - No lock needed yet
	files, err := resolveFiles(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

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
		return fmt.Errorf("no .dlog or fact files found in %s", path)
	}

	// 2. Parse and Build State (Local Variables)
	// We build everything locally to ensure atomicity. If parsing fails,
	// the existing runtime state remains untouched.
	var newRuleUnits []parse.SourceUnit
	edbDeclarations := make(map[ast.PredicateSym]ast.Decl)

	// Parse Rules
	for _, ruleFile := range ruleFiles {
		unit, err := parseRuleFile(ruleFile)
		if err != nil {
			return fmt.Errorf("failed to parse rule file %s: %w", ruleFile, err)
		}
		newRuleUnits = append(newRuleUnits, unit)
	}

	// Parse Facts and build Base Store
	newBaseStore := factstore.NewSimpleInMemoryStore()
	for _, factFile := range factFiles {
		unit, err := parseRuleFile(factFile)
		if err != nil {
			return fmt.Errorf("failed to parse fact file %s: %w", factFile, err)
		}
		for _, clause := range unit.Clauses {
			if len(clause.Premises) == 0 {
				newBaseStore.Add(clause.Head)
			}
		}
	}

	// 3. Analyze and Stratify (CPU Intensive)
	programInfo, err := analysis.Analyze(newRuleUnits, edbDeclarations)
	if err != nil {
		return fmt.Errorf("failed to analyze program: %w", err)
	}

	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return fmt.Errorf("failed to stratify program: %w", err)
	}

	// 4. Initial Evaluation (Validation)
	// We run this on the local store to ensure the program doesn't crash on init.
	if _, err := engine.EvalStratifiedProgramWithStats(programInfo, strata, predToStratum, newBaseStore); err != nil {
		return fmt.Errorf("failed to evaluate base program: %w", err)
	}

	// 5. Atomic Swap (Critical Section)
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ruleUnits = newRuleUnits
	r.baseFactStore = newBaseStore
	r.programInfo = programInfo
	r.strata = strata
	r.predToStratum = predToStratum
	r.ready = true

	return nil
}

// LoadFromSource parses and loads a full Datalog program from a string.
// REPLACES current state.
func (r *MangleRuntime) LoadFromSource(source string) error {
	if source == "" {
		return fmt.Errorf("source cannot be empty")
	}

	cleaned := cleanSource(source)
	unit, err := parse.Unit(strings.NewReader(cleaned))
	if err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}

	// Local state build
	newRuleUnits := []parse.SourceUnit{unit}

	// Add external predicates as extra declarations BEFORE analysis
	// This allows predicates to be used in rules without explicit Decl in policy
	edbDeclarations := make(map[ast.PredicateSym]ast.Decl)
	extPreds := r.externalPreds.List()
	for predSym := range extPreds {
		// Only add if not already declared in the policy (checked after parsing)
		// For now, add all - analysis will use policy Decl if present
		if _, exists := edbDeclarations[predSym]; !exists {
			edbDeclarations[predSym] = ast.NewSyntheticDeclFromSym(predSym)
		}
	}

	programInfo, err := analysis.Analyze(newRuleUnits, edbDeclarations)
	if err != nil {
		return fmt.Errorf("failed to analyze program: %w", err)
	}

	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})

	// Create new store (resetting old facts if this is a full reload)
	newBaseStore := factstore.NewSimpleInMemoryStore()

	// Atomic Swap
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ruleUnits = newRuleUnits
	r.baseFactStore = newBaseStore
	r.programInfo = programInfo
	r.strata = strata
	r.predToStratum = predToStratum
	r.ready = true

	// Evaluate with empty base store
	if err := r.evaluate(r.baseFactStore); err != nil {
		return fmt.Errorf("failed to evaluate program: %w", err)
	}

	return nil
}

// LoadFacts injects a list of raw Datalog fact strings into the runtime's base knowledge.
func (r *MangleRuntime) LoadFacts(facts []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, factStr := range facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			return fmt.Errorf("failed to parse fact '%s': %w", factStr, err)
		}
		r.baseFactStore.Add(atom)
	}

	if r.ready {
		if err := r.evaluate(r.baseFactStore); err != nil {
			return fmt.Errorf("failed to evaluate program with new facts: %w", err)
		}
	}
	return nil
}

// LoadFromString parses and loads a full Datalog program provided as a string.
// IMPORTANT: This REPLACES the current program state.
func (r *MangleRuntime) LoadFromString(rule string) error {
	return r.LoadFromSource(rule)
}

// AddPolicy adds new rules to the existing program state (Incremental Loading).
func (r *MangleRuntime) AddPolicy(source string) error {
	if source == "" {
		return nil
	}

	cleaned := cleanSource(source)
	unit, err := parse.Unit(strings.NewReader(cleaned))
	if err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Append to existing rules
	newRuleUnits := make([]parse.SourceUnit, len(r.ruleUnits)+1)
	copy(newRuleUnits, r.ruleUnits)
	newRuleUnits[len(r.ruleUnits)] = unit

	// Re-Analyze
	edbDeclarations := make(map[ast.PredicateSym]ast.Decl)
	programInfo, err := analysis.Analyze(newRuleUnits, edbDeclarations)
	if err != nil {
		return fmt.Errorf("failed to analyze combined program: %w", err)
	}

	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return fmt.Errorf("failed to stratify combined program: %w", err)
	}

	// Update State
	r.ruleUnits = newRuleUnits
	r.programInfo = programInfo
	r.strata = strata
	r.predToStratum = predToStratum
	r.ready = true

	// Re-evaluate base facts with new rules
	if err := r.evaluate(r.baseFactStore); err != nil {
		return fmt.Errorf("failed to evaluate combined program: %w", err)
	}

	return nil
}

// ExecuteQuery runs a boolean Datalog query.
func (r *MangleRuntime) ExecuteQuery(facts []ast.Atom, queryStr string) (bool, error) {
	r.mu.RLock()
	// Check readiness under lock
	if !r.ready {
		r.mu.RUnlock()
		return false, fmt.Errorf("runtime not initialized")
	}

	// 1. Snapshot the state needed for evaluation
	// We copy the base store to avoid contaminating the global state with request-scoped facts.
	// Note: This is O(N) where N is base facts.
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)

	// Capture pointers to analysis structures (they are read-only during eval)
	pInfo := r.programInfo
	strata := r.strata
	pStratum := r.predToStratum

	r.mu.RUnlock() // Release lock early to allow concurrent evaluations

	// 2. Add temporary facts
	for _, fact := range facts {
		workingStore.Add(fact)
	}

	// 3. Evaluate (Expensive part, runs without blocking main lock)
	if _, err := engine.EvalStratifiedProgramWithStats(pInfo, strata, pStratum, workingStore); err != nil {
		return false, fmt.Errorf("evaluation failed: %w", err)
	}

	// 4. Check result
	queryAtom, err := parse.Atom(queryStr)
	if err != nil {
		return false, fmt.Errorf("failed to parse query '%s': %w", queryStr, err)
	}

	return workingStore.Contains(queryAtom), nil
}

// QueryWithSolutions executes a query and invokes callback for solutions.
func (r *MangleRuntime) QueryWithSolutions(facts []ast.Atom, queryStr string, onSolution func(map[string]any) error) error {
	r.mu.RLock()
	if !r.ready {
		r.mu.RUnlock()
		return fmt.Errorf("runtime not initialized")
	}

	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)
	pInfo := r.programInfo
	strata := r.strata
	pStratum := r.predToStratum
	r.mu.RUnlock()

	for _, fact := range facts {
		workingStore.Add(fact)
	}

	if _, err := engine.EvalStratifiedProgramWithStats(pInfo, strata, pStratum, workingStore); err != nil {
		return fmt.Errorf("evaluation failed: %w", err)
	}

	queryAtom, err := parse.Atom(queryStr)
	if err != nil {
		return fmt.Errorf("failed to parse query '%s': %w", queryStr, err)
	}

	q := ast.NewQuery(queryAtom.Predicate)

	return workingStore.GetFacts(q, func(factAtom ast.Atom) error {
		// Manual Unification
		if len(factAtom.Args) != len(queryAtom.Args) {
			return nil
		}

		solution := make(map[string]any)
		match := true

		for i, queryArg := range queryAtom.Args {
			if v, isVar := queryArg.(ast.Variable); isVar {
				// It's a variable, capture the value
				valStr, err := constantToString(factAtom.Args[i])
				if err != nil {
					return fmt.Errorf("error converting term: %w", err)
				}
				solution[v.Symbol] = valStr
			} else {
				// It's a constant, check equality
				if !queryArg.Equals(factAtom.Args[i]) {
					match = false
					break
				}
			}
		}

		if match {
			return onSolution(solution)
		}
		return nil
	})
}

// evaluate helper (internal use only, assumes lock is held or local store)
func (r *MangleRuntime) evaluate(store factstore.FactStore) error {
	opts := r.buildEvalOptions()
	_, err := engine.EvalStratifiedProgramWithStats(r.programInfo, r.strata, r.predToStratum, store, opts...)
	return err
}

// buildEvalOptions builds evaluation options including external predicates and temporal store.
func (r *MangleRuntime) buildEvalOptions() []engine.EvalOption {
	opts := []engine.EvalOption{}

	// Add temporal store if enabled
	if r.temporalEnabled && r.temporalStore != nil {
		opts = append(opts, engine.WithTemporalStore(r.temporalStore))
	}

	// Add evaluation time if set (required for temporal queries)
	if !r.evaluationTime.IsZero() {
		opts = append(opts, engine.WithEvaluationTime(r.evaluationTime))
	}

	// Add external predicates
	extPreds := r.externalPreds.List()
	if len(extPreds) > 0 {
		callbacks := make(map[ast.PredicateSym]engine.ExternalPredicateCallback, len(extPreds))
		for predSym, fn := range extPreds {
			callbacks[predSym] = &externalPredicateAdapter{fn: fn}
		}
		opts = append(opts, engine.WithExternalPredicates(callbacks))
	}

	return opts
}

// externalPredicateAdapter wraps a simple ExternalPredicate function
// to implement the Mangle engine's ExternalPredicateCallback interface.
type externalPredicateAdapter struct {
	fn ExternalPredicate
}

// ShouldPushdown returns false - we don't support pushdown optimization.
func (a *externalPredicateAdapter) ShouldPushdown() bool {
	return false
}

// ShouldQuery returns true to indicate we want to query for results.
func (a *externalPredicateAdapter) ShouldQuery(inputs []ast.Constant, filters []ast.BaseTerm, pushdown []ast.Term) bool {
	return true
}

// ExecuteQuery runs the external predicate and adds results to the store.
func (a *externalPredicateAdapter) ExecuteQuery(inputs []ast.Constant, filters []ast.BaseTerm, pushdown []ast.Term, cb func([]ast.BaseTerm)) error {
	ctx := context.Background()

	// Convert AST inputs to Go values
	goInputs := make([]any, len(inputs))
	for i, inp := range inputs {
		val, err := astConstantToGoValue(inp)
		if err != nil {
			return fmt.Errorf("failed to convert input %d: %w", i, err)
		}
		goInputs[i] = val
	}

	// Call the external function
	results, err := a.fn(ctx, goInputs)
	if err != nil {
		return fmt.Errorf("external predicate failed: %w", err)
	}

	// Convert results back to AST terms
	for _, result := range results {
		if len(result) == 0 {
			continue
		}
		terms := make([]ast.BaseTerm, len(result))
		for i, val := range result {
			terms[i] = goValueToAstConstant(val)
		}
		cb(terms)
	}

	return nil
}

// astConstantToGoValue converts an Mangle AST constant to a Go value.
func astConstantToGoValue(c ast.Constant) (any, error) {
	if v, err := c.NameValue(); err == nil {
		return v, nil
	}
	if v, err := c.StringValue(); err == nil {
		return v, nil
	}
	// Handle NumberType - store as int64
	if c.Type == ast.NumberType {
		return c.NumValue, nil
	}
	return c.Symbol, nil
}

// goValueToAstConstant converts a Go value to an Mangle AST constant.
func goValueToAstConstant(v any) ast.BaseTerm {
	switch val := v.(type) {
	case string:
		return ast.String(val)
	case int:
		return ast.Number(int64(val))
	case int64:
		return ast.Number(val)
	case float64:
		return ast.Float64(val)
	case bool:
		if val {
			return ast.TrueConstant
		}
		return ast.FalseConstant
	default:
		return ast.String(fmt.Sprintf("%v", val))
	}
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

func cleanSource(raw string) string {
	// Strip UTF-8 BOM
	if strings.HasPrefix(raw, "\xef\xbb\xbf") {
		raw = raw[3:]
	}

	// Normalize line endings
	s := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	kept := lines[:0]

	for _, ln := range lines {
		trimLn := strings.TrimSpace(ln)

		// 1. Skip empty lines
		if trimLn == "" {
			continue
		}

		// 2. Skip full line comments
		if strings.HasPrefix(trimLn, "%") || strings.HasPrefix(trimLn, "//") {
			continue
		}

		// 3. Handle inline comments while respecting quotes
		// We iterate through the string to find the start of a comment that is NOT inside a string.
		commentIdx := -1
		inQuote := false
		for i := 0; i < len(ln); i++ {
			char := ln[i]
			if char == '"' {
				// Handle escaped quotes if necessary, though Datalog usually implies simple escaping?
				// For now, toggle state. strictly speaking, we should check for backslash.
				escaped := false
				if i > 0 && ln[i-1] == '\\' {
					escaped = true
				}
				if !escaped {
					inQuote = !inQuote
				}
			}

			if !inQuote {
				// Check for %
				if char == '%' {
					commentIdx = i
					break
				}
				// Check for //
				if char == '/' && i+1 < len(ln) && ln[i+1] == '/' {
					commentIdx = i
					break
				}
			}
		}

		if commentIdx >= 0 {
			ln = ln[:commentIdx]
		}

		if strings.TrimSpace(ln) == "" {
			continue
		}

		kept = append(kept, ln)
	}
	return strings.Join(kept, "\n")
}

func parseRuleFile(file string) (parse.SourceUnit, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return parse.SourceUnit{}, fmt.Errorf("could not open rule file %s: %w", file, err)
	}
	cleaned := cleanSource(string(b))
	unit, err := parse.Unit(strings.NewReader(cleaned))
	if err != nil {
		return parse.SourceUnit{}, fmt.Errorf("could not parse rule file %s: %w", file, err)
	}
	return unit, nil
}

func constantToString(term ast.BaseTerm) (string, error) {
	if c, ok := term.(ast.Constant); ok {
		if v, err := c.StringValue(); err == nil {
			return v, nil
		}
		if v, err := c.NameValue(); err == nil {
			return v, nil
		}
		if v, err := c.NumberValue(); err == nil {
			return fmt.Sprintf("%d", v), nil
		}
		if v, err := c.Float64Value(); err == nil {
			return fmt.Sprintf("%g", v), nil
		}
		return "", fmt.Errorf("unsupported constant type: %v", c.Type)
	}
	return fmt.Sprintf("%v", term), nil
}

// resolveFiles remains the same as your original implementation...
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

func hasMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}
