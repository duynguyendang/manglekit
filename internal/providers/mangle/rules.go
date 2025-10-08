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

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/mangle/converters"
	"github.com/google/mangle/analysis"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/engine"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/parse"
)

func init() {
	// Register the new, type-safe constructor.
	manglekit.RegisterRules("mangle", New)
}

// ruleSet implements the core.RuleSet interface using the Mangle Datalog engine.
// It loads rule files, parses schema definitions, and uses fact converters to
// bridge the gap between Go objects and Datalog facts for evaluation.
type ruleSet struct {
	programInfo           *analysis.ProgramInfo
	strata                []analysis.Nodeset
	predToStratum         map[ast.PredicateSym]int
	baseFactStore         factstore.SimpleInMemoryStore
	preProcessConverters  []core.FactConverter
	postProcessConverters []core.FactConverter
}

// New creates a new Mangle-based core.FlowController from the supplied options.
// This constructor is responsible for the entire setup of the Mangle engine, including:
//   - Parsing schema definitions to create a base set of facts.
//   - Loading and analyzing all Datalog rule files (.dlog).
//   - Setting up the required fact converters for pre- and post-processing stages.
//   - Performing an initial evaluation of the base program.
//
// ctx is the context for initialization.
// opts provides the configuration, including paths to rule files and schemas.
// It returns a fully initialized core.FlowController ready for evaluation, or an error
// if any part of the setup fails.
func New(ctx context.Context, opts core.MangleOptions) (core.RuleSet, error) {
	// This function actually returns a *ruleSet, which implements core.FlowController.
	// The signature remains core.RuleSet to match the registry. The builder will
	// perform a type assertion.
	if len(opts.Path) == 0 {
		return nil, fmt.Errorf("mangle: at least one path in 'path' must be provided")
	}

	initialFacts, schemaDecls, err := parseSchemas(opts.SchemaSources)
	if err != nil {
		return nil, fmt.Errorf("mangle: failed to parse schemas: %w", err)
	}

	var preConverters, postConverters []core.FactConverter
	// Load default converters if explicitly requested, OR if no custom converters are specified.
	// This provides sane defaults for new users, while allowing advanced users to override.
	if opts.DefaultConverters || (len(opts.PreProcess) == 0 && len(opts.PostProcess) == 0) {
		qc, err := converters.NewQueryConverter()
		if err != nil {
			return nil, fmt.Errorf("mangle: failed to create queryConverter: %w", err)
		}
		ucc, err := converters.NewUserContextConverter()
		if err != nil {
			return nil, fmt.Errorf("mangle: failed to create userContextConverter: %w", err)
		}
		dc, err := converters.NewDocumentConverter()
		if err != nil {
			return nil, fmt.Errorf("mangle: failed to create documentConverter: %w", err)
		}
		preConverters = append(preConverters, qc, ucc)
		postConverters = append(postConverters, dc, ucc)
	}

	// Load custom converters
	for _, name := range opts.PreProcess {
		conv, err := loadConverter(name)
		if err != nil {
			return nil, fmt.Errorf("mangle: failed to load pre-process converter '%s': %w", name, err)
		}
		preConverters = append(preConverters, conv)
	}
	for _, name := range opts.PostProcess {
		conv, err := loadConverter(name)
		if err != nil {
			return nil, fmt.Errorf("mangle: failed to load post-process converter '%s': %w", name, err)
		}
		postConverters = append(postConverters, conv)
	}

	allConverters := append(preConverters, postConverters...)
	edbDecls := make(map[ast.PredicateSym]ast.Decl)
	for _, conv := range allConverters {
		for _, pred := range conv.Predicates() {
			edbDecls[pred] = ast.Decl{}
		}
	}
	// Add declarations from the newly parsed schemas.
	for _, decl := range schemaDecls {
		edbDecls[decl] = ast.Decl{}
	}

	programInfo, strata, predToStratum, err := loadProgram(opts.Path, edbDecls)
	if err != nil {
		return nil, fmt.Errorf("mangle: could not load program: %w", err)
	}

	baseStore := factstore.NewSimpleInMemoryStore()
	for _, fact := range initialFacts {
		baseStore.Add(fact)
	}

	if err := evaluate(programInfo, strata, predToStratum, baseStore); err != nil {
		return nil, fmt.Errorf("mangle: could not evaluate base program: %w", err)
	}

	return &ruleSet{
		programInfo:           programInfo,
		strata:                strata,
		predToStratum:         predToStratum,
		baseFactStore:         baseStore,
		preProcessConverters:  preConverters,
		postProcessConverters: postConverters,
	}, nil
}

// loadConverter looks up a converter by name from the registry, instantiates it,
// and returns it as a core.FactConverter.
func loadConverter(name string) (core.FactConverter, error) {
	constructor, err := manglekit.Get(manglekit.Registry.Component, name)
	if err != nil {
		return nil, err
	}
	constructorFn, ok := constructor.(func(map[string]any) (any, error))
	if !ok {
		return nil, fmt.Errorf("invalid constructor type for converter '%s'", name)
	}
	instance, err := constructorFn(nil) // Assuming no params for now
	if err != nil {
		return nil, fmt.Errorf("failed to construct converter '%s': %w", name, err)
	}
	converter, ok := instance.(core.FactConverter)
	if !ok {
		return nil, fmt.Errorf("constructed component '%s' is not a core.FactConverter", name)
	}
	return converter, nil
}

// parseSchemas handles the logic of reading and parsing schema definition files.
// This logic was moved from the builder to here.
func parseSchemas(sources []core.SchemaSource) ([]ast.Atom, []ast.PredicateSym, error) {
	if len(sources) == 0 {
		return nil, nil, nil
	}

	var allFacts []ast.Atom
	var allDecls []ast.PredicateSym

	for _, source := range sources {
		// 1. Look up the parser constructor from the registry.
		parserConstructor, err := manglekit.Get(manglekit.Registry.SchemaParser, source.Type)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get schema parser constructor '%s': %w", source.Type, err)
		}

		// 2. Assert the constructor type and create a parser instance.
		constructorFn, ok := parserConstructor.(func(map[string]any) (any, error))
		if !ok {
			return nil, nil, fmt.Errorf("invalid constructor type for schema parser '%s'", source.Type)
		}
		parserAny, err := constructorFn(nil) // Assume no params for now.
		if err != nil {
			return nil, nil, fmt.Errorf("failed to construct schema parser '%s': %w", source.Type, err)
		}
		parser, ok := parserAny.(core.SchemaParser)
		if !ok {
			return nil, nil, fmt.Errorf("constructed parser for '%s' is not a core.SchemaParser", source.Type)
		}

		// 3. Collect predicate declarations from the parser.
		allDecls = append(allDecls, parser.Predicates()...)

		// 4. Read the schema file.
		f, err := os.Open(source.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open schema file '%s': %w", source.Path, err)
		}
		defer f.Close()

		// 5. Parse and collect facts.
		facts, err := parser.Parse(f)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse schema '%s' with parser '%s': %w", source.Path, source.Type, err)
		}
		allFacts = append(allFacts, facts...)
	}
	return allFacts, allDecls, nil
}

// Evaluate runs the Mangle evaluation for a specific pipeline stage (Pre or Post).
// It dispatches to the appropriate internal processing function based on the stage.
// This method satisfies the core.RuleSet interface.
//
// stage determines whether to run the pre-processing or post-processing logic.
// q is the user query, available for inspection and mutation.
// a is the answer object, which is primarily used in the Post-processing stage.
// It returns a RuleResult indicating success and any mutations, or an error.
func (r *ruleSet) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	switch stage {
	case core.Pre:
		return r.preProcess(q)
	case core.Post:
		return r.postProcess(q, a)
	}
	return core.RuleResult{}, fmt.Errorf("unknown stage: %v", stage)
}

// preProcess normalizes the user query and enriches it with expansions.
func (r *ruleSet) preProcess(query core.Query) (core.RuleResult, error) {
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)

	for _, converter := range r.preProcessConverters {
		facts, err := converter.ToFacts(query)
		if err != nil {
			return core.RuleResult{}, fmt.Errorf("mangle: pre-process converter failed: %w", err)
		}
		for _, fact := range facts {
			workingStore.Add(fact)
		}
	}

	if err := evaluate(r.programInfo, r.strata, r.predToStratum, workingStore); err != nil {
		return core.RuleResult{Allowed: false, Reason: "Mangle pre-process failed"}, fmt.Errorf("mangle: pre-process evaluation failed: %w", err)
	}

	// Check for skipped stages first.
	skipped, err := collectStrings(workingStore, "skip_stage", 1)
	if err != nil {
		return core.RuleResult{Allowed: false, Reason: "could not collect 'skip_stage' facts"}, err
	}
	skippedMap := make(map[string]bool)
	for _, s := range skipped {
		skippedMap[s] = true
	}

	// Check for deny facts. If any exist, the request is not allowed.
	denied, err := collectKeyValue(workingStore, "deny")
	if err != nil {
		return core.RuleResult{Allowed: false, Reason: "could not collect 'deny' facts"}, err
	}
	if len(denied) > 0 {
		// Collect the first reason for the error message.
		var reason string
		for r := range denied {
			reason = r
			break
		}

		// Create a mutation function to attach the full denial reasons to the answer.
		mutateFn := func(q *core.Query, a *core.Answer) {
			if a.Meta == nil {
				a.Meta = make(map[string]any)
			}
			a.Meta["mangle_denied_reasons"] = denied
		}

		// Return a result that denies access, provides a reason, and includes the
		// mutation function to record the details. The orchestrator will convert this
		// into a core.ErrDenied.
		return core.RuleResult{Allowed: false, Reason: reason, Mutate: mutateFn, SkippedStages: skippedMap}, nil
	}

	expansions, err := collectStrings(workingStore, "expanded_query", 2)
	if err != nil {
		return core.RuleResult{Allowed: false, Reason: "could not collect expansions"}, err
	}
	filters, err := collectKeyValue(workingStore, "query_filter")
	if err != nil {
		return core.RuleResult{Allowed: false, Reason: "could not collect filters"}, err
	}

	mutateFn := func(q *core.Query, a *core.Answer) {
		if q.Meta == nil {
			q.Meta = make(map[string]any)
		}
		q.Meta["filters"] = filters
		q.Meta["expansion_terms"] = expansions
	}

	return core.RuleResult{Allowed: true, Mutate: mutateFn, SkippedStages: skippedMap}, nil
}

// Query executes a read-only Datalog query against the base, pre-evaluated facts
// of the program. This is used to retrieve static configuration and definitions,
// such as the stages of a declarative flow. It implements the core.Querier interface.
func (r *ruleSet) Query(ctx context.Context, query string, onSolution func(map[string]any) error) error {
	queryAtom, err := parse.Atom(query)
	if err != nil {
		return fmt.Errorf("mangle: could not parse query atom '%s': %w", query, err)
	}
	q := ast.NewQuery(queryAtom.Predicate)

	return r.baseFactStore.GetFacts(q, func(factAtom ast.Atom) error {
		// Manual unification of the query atom against the fact atom.
		match := true
		if len(factAtom.Args) != len(queryAtom.Args) {
			return nil // Should not happen if predicate is the same.
		}
		for i, queryArg := range queryAtom.Args {
			if _, isVar := queryArg.(ast.Variable); isVar {
				continue // Variable matches anything.
			}
			// If queryArg is a constant, it must match the corresponding factArg.
			if !queryArg.Equals(factAtom.Args[i]) {
				match = false
				break
			}
		}
		if !match {
			return nil // Does not match, skip to the next fact.
		}

		// It's a match, now extract variables to build the solution map.
		solution := make(map[string]any)
		for i, arg := range queryAtom.Args {
			if v, ok := arg.(ast.Variable); ok {
				term := factAtom.Args[i]
				var val string
				var err error
				if c, ok := term.(ast.Constant); ok {
					val, err = constantToString(c)
					if err != nil {
						return fmt.Errorf("could not convert solution term '%v' to string: %w", term, err)
					}
				} else {
					// This should not happen if facts are well-formed.
					return fmt.Errorf("expected a constant in query solution, but got %T", term)
				}
				solution[v.Symbol] = val
			}
		}
		return onSolution(solution)
	})
}

// postProcess filters an answer based on Mangle rules.
func (r *ruleSet) postProcess(query core.Query, answer *core.Answer) (core.RuleResult, error) {
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)

	originalDocs := make(map[string]core.Doc)
	if docs, ok := answer.Meta["original_docs"].([]core.Doc); ok {
		for _, doc := range docs {
			originalDocs[doc.ID] = doc
		}
	}

	docsToProcess := make(map[string]core.Doc)
	for _, citation := range answer.Citations {
		if doc, ok := originalDocs[citation.ID]; ok {
			docsToProcess[doc.ID] = doc
		}
	}

	for _, converter := range r.postProcessConverters {
		// Skip the document converter in this loop; it's handled separately.
		if _, ok := converter.(*converters.DocumentConverter); ok {
			continue
		}
		facts, err := converter.ToFacts(query)
		if err != nil {
			return core.RuleResult{}, fmt.Errorf("mangle: post-process converter failed: %w", err)
		}
		for _, fact := range facts {
			workingStore.Add(fact)
		}
	}

	docConverter, err := converters.NewDocumentConverter()
	if err != nil {
		return core.RuleResult{}, fmt.Errorf("mangle: could not create document converter: %w", err)
	}
	for _, doc := range docsToProcess {
		facts, err := docConverter.ToFacts(doc)
		if err != nil {
			return core.RuleResult{}, fmt.Errorf("mangle: post-process document converter failed for doc %s: %w", doc.ID, err)
		}
		for _, fact := range facts {
			workingStore.Add(fact)
		}
	}

	if err := evaluate(r.programInfo, r.strata, r.predToStratum, workingStore); err != nil {
		return core.RuleResult{Allowed: false, Reason: "Mangle post-process failed"}, fmt.Errorf("mangle: post-process evaluation failed: %w", err)
	}

	denied, err := collectKeyValue(workingStore, "deny")
	if err != nil {
		return core.RuleResult{Allowed: false, Reason: "could not collect 'deny' facts"}, err
	}

	var allowedCitations []core.Citation
	for _, citation := range answer.Citations {
		if _, isDenied := denied[citation.ID]; !isDenied {
			allowedCitations = append(allowedCitations, citation)
		}
	}

	answer.Citations = allowedCitations
	if len(denied) > 0 {
		if answer.Meta == nil {
			answer.Meta = make(map[string]any)
		}
		answer.Meta["mangle_denied_reasons"] = denied
	}

	return core.RuleResult{Allowed: true}, nil
}

// --- Mangle Helper Functions ---
func loadProgram(paths []string, edbDeclarations map[ast.PredicateSym]ast.Decl) (*analysis.ProgramInfo, []analysis.Nodeset, map[ast.PredicateSym]int, error) {
	var files []string
	for _, path := range paths {
		resolved, err := resolveDlogFiles(path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not resolve rule path %q: %w", path, err)
		}
		files = append(files, resolved...)
	}
	if len(files) == 0 {
		return nil, nil, nil, fmt.Errorf("no .dlog files found in any of the paths")
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

	programInfo, err := analysis.Analyze(units, edbDeclarations)
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
		value, err := constantToString(constant)
		if err != nil {
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
