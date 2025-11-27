// Package mangle provides the primary implementation of the `core.RuleSet` and
// `core.FlowController` interfaces, using the Google Mangle Datalog engine.
package mangle

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/v1/internal/providers/rules/mangle/converters"
	"github.com/google/mangle/analysis"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/engine"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/parse"
)

func Register(r *manglekit.Registry) {
	manglekit.Register(r, &core.MangleOptions{},
		func(ctx context.Context, deps diapi.RuleSetDeps, cfg *core.MangleOptions) (core.RuleSet, error) {
			return New(ctx, *cfg, deps.Registry.(*manglekit.Registry))
		},
	)
}

var builtinRedactions = map[string]*regexp.Regexp{
	"phone": regexp.MustCompile(`[0-9]{3}-[0-9]{3}-[0-9]{4}`),
	"email": regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`),
}

// RuleSet implements the `core.RuleSet` and `core.FlowController` interfaces
// using the Mangle Datalog engine. It manages the lifecycle of a Datalog
// program, including loading rule files, parsing external schema definitions,
// and using `FactConverter` components to bridge the gap between Go objects
// (like the query and answer) and the Datalog facts needed for evaluation.
type RuleSet struct {
	programInfo           *analysis.ProgramInfo
	strata                []analysis.Nodeset
	predToStratum         map[ast.PredicateSym]int
	baseFactStore         factstore.SimpleInMemoryStore
	preProcessConverters  []core.FactConverter
	postProcessConverters []core.FactConverter
}

// New is the constructor for the Mangle `RuleSet`. It is registered with the
// MangleKit registry and is responsible for the complex process of initializing
// the Mangle engine.
//
// The initialization process involves:
//  1. Parsing external schema definitions (e.g., JSON schemas) into Datalog facts.
//  2. Loading and constructing all configured `FactConverter` components.
//  3. Determining the set of all possible facts (the EDB) based on the configuration mode.
//  4. Loading and parsing all Datalog rule files (`.dlog`).
//  5. Performing static analysis and stratification of the Datalog program.
//  6. Seeding a base fact store with facts from files and schemas.
//  7. Performing an initial evaluation of the base program.
//
// ctx is the context for initialization.
// opts provides the configuration, including paths to rule files, schemas, and converters.
// It returns a fully initialized `core.RuleSet` (which also satisfies `core.FlowController`)
// or an error if any part of the initialization fails.
func New(ctx context.Context, opts core.MangleOptions, r *manglekit.Registry) (*RuleSet, error) {
	if len(opts.Path) == 0 {
		return nil, fmt.Errorf("mangle: at least one path in 'path' must be provided")
	}
	log := opts.Logger
	if log == nil {
		log = obslogger.NewStdLogger().With("component", "rules", "provider", "mangle")
	}

	// 1) Parse schemas (if any). We only take schema facts here; EDB declarations
	// come from either code (code-first) or .dlog Decl (file-first).
	schemaFacts, schemaDecls, err := parseSchemas(opts.SchemaSources, r)
	if err != nil {
		return nil, fmt.Errorf("mangle: failed to parse schemas: %w", err)
	}

	// 2) Load converters (for ToFacts). We DO NOT rely on them for EDB if FileFirst=true.
	var preConverters, postConverters []core.FactConverter
	forceDefaults := opts.DefaultConverters || (len(opts.PreProcess) == 0 && len(opts.PostProcess) == 0)
	if forceDefaults {
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

	// Custom converters
	for _, name := range opts.PreProcess {
		conv, err := loadConverter(name, r)
		if err != nil {
			return nil, fmt.Errorf("mangle: failed to load pre-process converter '%s': %w", name, err)
		}
		preConverters = append(preConverters, conv)
	}
	for _, name := range opts.PostProcess {
		conv, err := loadConverter(name, r)
		if err != nil {
			return nil, fmt.Errorf("mangle: failed to load post-process converter '%s': %w", name, err)
		}
		postConverters = append(postConverters, conv)
	}

	// 3) Build EDB declarations depending on mode.
	// In file-first mode, let .dlog Decl define EDB/IDB; do not pre-register here.
	edbDecls := make(map[ast.PredicateSym]ast.Decl)
	if !opts.FileFirst {
		// Code-first: collect from converters + schemaDecls
		allConverters := append(preConverters, postConverters...)
		for _, conv := range allConverters {
			for _, pred := range conv.Predicates() {
				edbDecls[pred] = ast.Decl{}
			}
		}
		for _, decl := range schemaDecls {
			edbDecls[decl] = ast.Decl{}
		}
		log.Infof("mangle rules boot", "mode", "code-first")
		// Sort predicates for deterministic output
		predicates := make([]ast.PredicateSym, 0, len(edbDecls))
		for p := range edbDecls {
			predicates = append(predicates, p)
		}
		sort.Slice(predicates, func(i, j int) bool {
			if predicates[i].Symbol != predicates[j].Symbol {
				return predicates[i].Symbol < predicates[j].Symbol
			}
			return predicates[i].Arity < predicates[j].Arity
		})
		for _, p := range predicates {
			log.Debugf("mangle predicate registered", "predicate", p.Symbol, "arity", p.Arity)
		}
	} else {
		log.Infof("mangle rules boot", "mode", "file-first")
	}

	// 4) Load program (rules + facts)
	programInfo, strata, predToStratum, fileFacts, err := loadProgram(opts.Path, edbDecls, log)
	if err != nil {
		return nil, fmt.Errorf("mangle: could not load program: %w", err)
	}

	// 5) Seed base store with schema facts + file facts
	baseStore := factstore.NewSimpleInMemoryStore()
	for _, f := range append(schemaFacts, fileFacts...) {
		baseStore.Add(f)
	}

	// 6) Evaluate base program
	if err := evaluate(programInfo, strata, predToStratum, baseStore); err != nil {
		return nil, fmt.Errorf("mangle: could not evaluate base program: %w", err)
	}

	return &RuleSet{
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
func loadConverter(name string, r *manglekit.Registry) (core.FactConverter, error) {
	if r == nil {
		return nil, fmt.Errorf("cannot load converter '%s': registry is nil", name)
	}
	factory, err := r.Get(core.Kind("fact_converter"), name)
	if err != nil {
		return nil, fmt.Errorf("failed to get factory for fact_converter '%s': %w", name, err)
	}
	instance, err := factory.Build(context.Background(), diapi.NoopDeps{}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct converter '%s': %w", name, err)
	}
	return instance.(core.FactConverter), nil
}

// parseSchemas handles reading and parsing schema definition files.
func parseSchemas(sources []core.SchemaSource, r *manglekit.Registry) ([]ast.Atom, []ast.PredicateSym, error) {
	if len(sources) == 0 {
		return nil, nil, nil
	}

	var allFacts []ast.Atom
	var allDecls []ast.PredicateSym

	for _, source := range sources {
		if r == nil {
			return nil, nil, fmt.Errorf("cannot parse schema '%s': registry is nil", source.Path)
		}
		// 1. Lookup parser factory
		factory, err := r.Get(core.KindSchemaParser, source.Type)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get schema parser factory '%s': %w", source.Type, err)
		}

		// 2. Construct parser
		instance, err := factory.Build(context.Background(), diapi.NoopDeps{}, nil) // Assume no params for now.
		if err != nil {
			return nil, nil, fmt.Errorf("failed to construct schema parser '%s': %w", source.Type, err)
		}
		parser := instance.(core.SchemaParser)

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
// It creates a temporary fact store, adds the base facts, converts the current
// query and answer state into new facts using the appropriate converters, runs
// the Datalog engine, and then collects the results (like denials or mutations)
// to return to the orchestrator.
// This method satisfies the `core.RuleSet` interface.
func (r *RuleSet) Evaluate(ctx context.Context, stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	switch stage {
	case core.Pre:
		return r.preProcess(ctx, q)
	case core.Post:
		return r.postProcess(ctx, q, a)
	}
	return core.RuleResult{}, fmt.Errorf("unknown stage: %v", stage)
}

// preProcess normalizes the user query and enriches it with expansions.
func (r *RuleSet) preProcess(ctx context.Context, query core.Query) (core.RuleResult, error) {
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

	return r.collectPreResults(workingStore)
}

// EvaluateFacts satisfies the core.RuleSet interface by evaluating rules against explicit facts.
func (r *RuleSet) EvaluateFacts(ctx context.Context, stage core.Stage, facts []ast.Atom, a *core.Answer) (core.RuleResult, error) {
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)
	for _, f := range facts {
		workingStore.Add(f)
	}

	if err := evaluate(r.programInfo, r.strata, r.predToStratum, workingStore); err != nil {
		return core.RuleResult{Allowed: false, Reason: "Mangle evaluation failed"}, fmt.Errorf("mangle: evaluation failed: %w", err)
	}

	if stage == core.Pre {
		return r.collectPreResults(workingStore)
	}
	return core.RuleResult{Allowed: true}, nil
}

func (r *RuleSet) collectPreResults(workingStore factstore.ReadOnlyFactStore) (core.RuleResult, error) {
	// Skipped stages
	skipped, err := collectStrings(workingStore, "skip_stage", 1)
	if err != nil {
		return core.RuleResult{}, fmt.Errorf("failed to collect 'skip_stage' facts: %w", err)
	}
	skippedMap := make(map[string]bool)
	for _, s := range skipped {
		skippedMap[s] = true
	}

	// Deny?
	denied, err := collectKeyValue(workingStore, "deny")
	if err != nil {
		return core.RuleResult{}, fmt.Errorf("failed to collect 'deny' facts: %w", err)
	}
	if len(denied) > 0 {
		// Sort denied keys for deterministic reason selection
		deniedKeys := make([]string, 0, len(denied))
		for r := range denied {
			deniedKeys = append(deniedKeys, r)
		}
		sort.Strings(deniedKeys)
		reason := deniedKeys[0]
		mutateFn := func(q *core.Query, a *core.Answer) {
			if a.Meta == nil {
				a.Meta = make(map[string]any)
			}
			a.Meta["mangle_denied_reasons"] = denied
		}
		return core.RuleResult{Allowed: false, Reason: reason, Mutate: mutateFn, SkippedStages: skippedMap}, nil
	}

	expansions, err := collectStrings(workingStore, "expanded_query", 2)
	if err != nil {
		return core.RuleResult{}, fmt.Errorf("failed to collect 'expanded_query' facts: %w", err)
	}
	filters, err := collectKeyValue(workingStore, "query_filter")
	if err != nil {
		return core.RuleResult{}, fmt.Errorf("failed to collect 'query_filter' facts: %w", err)
	}

	meta, err := collectKeyValue(workingStore, "add_meta")
	if err != nil {
		return core.RuleResult{}, fmt.Errorf("failed to collect 'add_meta' facts: %w", err)
	}

	mutateFn := func(q *core.Query, a *core.Answer) {
		if q.Meta == nil {
			q.Meta = make(map[string]any)
		}
		q.Meta["filters"] = filters
		q.Meta["expansion_terms"] = expansions

		if a.Meta == nil {
			a.Meta = make(map[string]any)
		}
		for k, v := range meta {
			a.Meta[k] = v
		}
	}

	return core.RuleResult{Allowed: true, Mutate: mutateFn, SkippedStages: skippedMap}, nil
}

// Query executes a read-only Datalog query against the base, pre-evaluated facts.
// It is used by the declarative orchestrator to fetch its execution plan by
// querying for `flow_stage` and `stage_tool` facts. The query is a simple atom
// string (e.g., `foo(X, "bar")`), and the results are streamed to the `onSolution`
// callback.
// This method satisfies the `core.Querier` interface, making `RuleSet` a `core.FlowController`.
func (r *RuleSet) Query(ctx context.Context, query string, onSolution func(map[string]any) error) error {
	queryAtom, err := parse.Atom(query)
	if err != nil {
		return fmt.Errorf("mangle: could not parse query atom '%s': %w", query, err)
	}
	q := ast.NewQuery(queryAtom.Predicate)

	return r.baseFactStore.GetFacts(q, func(factAtom ast.Atom) error {
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

// Post evaluates post-retrieval rules before the LLM stage. It converts the current
// query, user context, evidence, and execution metadata into Mangle facts and
// returns a structured result describing any mutations requested by the rules.
func (r *RuleSet) Post(ctx context.Context, q core.Query, evidence []core.Doc, meta map[string]any) (core.PostRuleResult, error) {
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)

	for _, converter := range r.postProcessConverters {
		if _, ok := converter.(*converters.DocumentConverter); ok {
			continue
		}
		facts, err := converter.ToFacts(q)
		if err != nil {
			return core.PostRuleResult{}, fmt.Errorf("mangle: post-rules converter failed: %w", err)
		}
		for _, fact := range facts {
			workingStore.Add(fact)
		}
	}

	docConverter, err := converters.NewDocumentConverter()
	if err != nil {
		return core.PostRuleResult{}, fmt.Errorf("mangle: could not create document converter: %w", err)
	}

	for _, doc := range evidence {
		facts, err := docConverter.ToFacts(doc)
		if err != nil {
			return core.PostRuleResult{}, fmt.Errorf("mangle: post-rules document converter failed for doc %s: %w", doc.ID, err)
		}
		for _, fact := range facts {
			workingStore.Add(fact)
		}
		workingStore.Add(ast.NewAtom("retrieved_doc", ast.String(doc.ID)))
	}

	if err := addPostMetaFacts(workingStore, evidence, meta); err != nil {
		return core.PostRuleResult{}, fmt.Errorf("failed to add post-meta facts: %w", err)
	}

	if err := evaluate(r.programInfo, r.strata, r.predToStratum, workingStore); err != nil {
		return core.PostRuleResult{}, fmt.Errorf("mangle: post-rules evaluation failed: %w", err)
	}

	denyReasons, err := collectStrings(workingStore, "deny", 1)
	if err != nil {
		return core.PostRuleResult{}, fmt.Errorf("mangle: could not collect 'deny' facts: %w", err)
	}
	denyMap, err := collectKeyValue(workingStore, "deny")
	if err != nil {
		return core.PostRuleResult{}, fmt.Errorf("mangle: could not collect 'deny' keyed facts: %w", err)
	}

	dropIDs, err := collectStrings(workingStore, "drop_doc", 1)
	if err != nil {
		return core.PostRuleResult{}, fmt.Errorf("mangle: could not collect 'drop_doc' facts: %w", err)
	}
	dropReasons, err := collectKeyValue(workingStore, "drop_doc")
	if err != nil {
		return core.PostRuleResult{}, fmt.Errorf("mangle: could not collect 'drop_doc' keyed facts: %w", err)
	}

	dropSet := make(map[string]struct{}, len(dropIDs)+len(dropReasons))
	for _, id := range dropIDs {
		dropSet[id] = struct{}{}
		if _, ok := dropReasons[id]; !ok {
			dropReasons[id] = ""
		}
	}
	// Sort dropReasons keys for deterministic iteration
	reasonKeys := make([]string, 0, len(dropReasons))
	for id := range dropReasons {
		reasonKeys = append(reasonKeys, id)
	}
	sort.Strings(reasonKeys)
	for _, id := range reasonKeys {
		dropSet[id] = struct{}{}
	}

	globalRedacts, err := collectStrings(workingStore, "redact", 1)
	if err != nil {
		return core.PostRuleResult{}, fmt.Errorf("mangle: could not collect 'redact' facts: %w", err)
	}
	docRedacts, err := collectKeyValue(workingStore, "redact")
	if err != nil {
		return core.PostRuleResult{}, fmt.Errorf("mangle: could not collect 'redact' keyed facts: %w", err)
	}

	var redactionSpecs []redactionSpec
	for _, label := range globalRedacts {
		redactionSpecs = append(redactionSpecs, redactionSpec{Label: label})
	}
	for docID, label := range docRedacts {
		redactionSpecs = append(redactionSpecs, redactionSpec{DocID: docID, Label: label})
	}

	filtered := make([]core.Doc, 0, len(evidence))
	var appliedRedactions []map[string]any
	for _, doc := range evidence {
		if _, dropped := dropSet[doc.ID]; dropped {
			continue
		}
		updatedDoc, applied := applyRedactionsToDoc(doc, redactionSpecs)
		filtered = append(filtered, updatedDoc)
		appliedRedactions = append(appliedRedactions, applied...)
	}

	ruleResults := make([]map[string]any, 0)
	if len(denyReasons) == 0 {
		for _, reason := range denyMap {
			denyReasons = append(denyReasons, reason)
		}
	}
	seenReasons := make(map[string]struct{})
	for _, reason := range denyReasons {
		if reason == "" {
			continue
		}
		if _, ok := seenReasons[reason]; ok {
			continue
		}
		seenReasons[reason] = struct{}{}
		ruleResults = append(ruleResults, map[string]any{
			"rule_id": fmt.Sprintf("deny_%s", reason),
			"effect":  "deny",
			"reason":  reason,
		})
	}

	for docID, reason := range dropReasons {
		entry := map[string]any{
			"rule_id": fmt.Sprintf("drop_doc_%s", docID),
			"effect":  "drop_doc",
			"doc_id":  docID,
		}
		if reason != "" {
			entry["reason"] = reason
		}
		ruleResults = append(ruleResults, entry)
	}

	for _, spec := range redactionSpecs {
		entry := map[string]any{
			"rule_id": fmt.Sprintf("redact_%s", spec.Label),
			"effect":  "redact",
			"pattern": spec.Label,
		}
		if spec.DocID != "" {
			entry["doc_id"] = spec.DocID
		}
		ruleResults = append(ruleResults, entry)
	}

	resultMeta := map[string]any{
		"fired_rules": len(ruleResults),
	}
	if len(dropReasons) > 0 {
		resultMeta["dropped_docs"] = dropReasons
	}
	if len(appliedRedactions) > 0 {
		resultMeta["redactions"] = appliedRedactions
	}
	if len(ruleResults) > 0 {
		resultMeta["rule_results"] = ruleResults
	}

	var denyReason string
	if len(denyReasons) > 0 {
		denyReason = denyReasons[0]
	}
	if denyReason != "" {
		resultMeta["denied_reason"] = denyReason
	}

	return core.PostRuleResult{
		Filtered: filtered,
		Denied:   denyReason != "",
		Reason:   denyReason,
		Meta:     resultMeta,
	}, nil
}

// postProcess filters an answer based on Mangle rules.
func (r *RuleSet) postProcess(ctx context.Context, query core.Query, answer *core.Answer) (core.RuleResult, error) {
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

type redactionSpec struct {
	DocID string
	Label string
}

func applyRedactionsToDoc(doc core.Doc, specs []redactionSpec) (core.Doc, []map[string]any) {
	updated := doc
	var applied []map[string]any
	for _, spec := range specs {
		if spec.DocID != "" && spec.DocID != doc.ID {
			continue
		}
		newText, matched := applySingleRedaction(updated.Text, spec.Label)
		if matched {
			applied = append(applied, map[string]any{
				"doc_id":  doc.ID,
				"pattern": spec.Label,
			})
			updated.Text = newText
		}
	}
	return updated, applied
}

func applySingleRedaction(text, label string) (string, bool) {
	if label == "" {
		return text, false
	}
	if strings.HasPrefix(label, "regex:") {
		pattern := strings.TrimPrefix(label, "regex:")
		re, err := regexp.Compile(pattern)
		if err != nil {
			return text, false
		}
		replaced := re.ReplaceAllString(text, "[REDACTED]")
		return replaced, replaced != text
	}
	if re, ok := builtinRedactions[label]; ok {
		replaced := re.ReplaceAllString(text, "[REDACTED]")
		return replaced, replaced != text
	}
	if strings.Contains(text, label) {
		return strings.ReplaceAll(text, label, "[REDACTED]"), true
	}
	return text, false
}

func addPostMetaFacts(store factstore.FactStore, evidence []core.Doc, meta map[string]any) error {
	store.Add(ast.NewAtom("retrieved_count", ast.Number(int64(len(evidence)))))

	if meta == nil {
		meta = map[string]any{}
	}
	if _, ok := meta["retrieved_count"]; !ok {
		meta["retrieved_count"] = len(evidence)
	}

	if best, ok := meta["best_score"]; ok {
		if term := numericTerm(best); term != nil {
			store.Add(ast.NewAtom("best_score", term))
		}
	}
	if countTerm := numericTerm(meta["retrieved_count"]); countTerm != nil {
		store.Add(ast.NewAtom("retrieved_count_value", countTerm))
	}

	for key, value := range meta {
		if key == "best_score" || key == "retrieved_count" {
			continue
		}
		store.Add(ast.NewAtom("pipeline_meta", ast.String(key), ast.String(fmt.Sprintf("%v", value))))
	}
	return nil
}

func numericTerm(value any) ast.BaseTerm {
	switch v := value.(type) {
	case int:
		return ast.Number(int64(v))
	case int32:
		return ast.Number(int64(v))
	case int64:
		return ast.Number(v)
	case float32:
		return ast.Float64(float64(v))
	case float64:
		return ast.Float64(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return ast.Number(int64(i))
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return ast.Float64(f)
		}
	}
	return nil
}

// --- Mangle Helper Functions ---

func isRuleFile(p string) bool { return strings.HasSuffix(p, ".dlog") }
func isFactFile(p string) bool {
	return strings.HasSuffix(p, ".facts") ||
		strings.HasSuffix(p, ".fact") ||
		strings.HasSuffix(p, ".edb") ||
		strings.HasSuffix(p, ".data")
}

func loadProgram(paths []string, edbDeclarations map[ast.PredicateSym]ast.Decl, logger core.Logger) (*analysis.ProgramInfo, []analysis.Nodeset, map[ast.PredicateSym]int, []ast.Atom, error) {
	var ruleFiles, factFiles []string
	for _, path := range paths {
		resolved, err := resolveFiles(path)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("could not resolve rule path %q: %w", path, err)
		}
		for _, file := range resolved {
			switch {
			case isRuleFile(file):
				ruleFiles = append(ruleFiles, file)
			case isFactFile(file):
				factFiles = append(factFiles, file)
			}
		}
	}
	if len(ruleFiles) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("no .dlog files found in any of the paths")
	}

	var units []parse.SourceUnit
	for _, file := range ruleFiles {
		unit, err := parseFile(file)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		units = append(units, unit)
	}

	var initialFacts []ast.Atom
	for _, file := range factFiles {
		unit, err := parseFile(file)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		// A fact is a clause with an empty body (no premises).
		for _, clause := range unit.Clauses {
			if len(clause.Premises) == 0 {
				initialFacts = append(initialFacts, clause.Head)
			}
		}
	}

	logger.Debugf("mangle rule inputs", "rule_files", ruleFiles, "fact_files", factFiles)

	programInfo, err := analysis.Analyze(units, edbDeclarations)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("mangle: static analysis failed: %w", err)
	}

	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("mangle: stratification failed: %w", err)
	}

	return programInfo, strata, predToStratum, initialFacts, nil
}

func parseFile(file string) (parse.SourceUnit, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return parse.SourceUnit{}, fmt.Errorf("could not open rule file %s: %w", file, err)
	}
	// strip UTF-8 BOM if any
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		b = b[3:]
	}
	// normalize newlines and drop lines that are just "."
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	lines := strings.Split(s, "\n")
	kept := lines[:0]
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "." {
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
				return nil, fmt.Errorf("no rule files matched %q", path)
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

func evaluate(programInfo *analysis.ProgramInfo, strata []analysis.Nodeset, predToStratum map[ast.PredicateSym]int, store factstore.FactStore) error {
	_, err := engine.EvalStratifiedProgramWithStats(programInfo, strata, predToStratum, store)
	if err != nil {
		return fmt.Errorf("mangle engine evaluation failed: %w", err)
	}
	return nil
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
		return nil, fmt.Errorf("failed to get facts for predicate '%s': %w", predicate, err)
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
		return nil, fmt.Errorf("failed to get facts for predicate '%s': %w", predicate, err)
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

// Reason takes a set of input facts, runs the Datalog engine, and returns the
// resulting facts. This method is the core of the Reasoner implementation.
func (r *RuleSet) Reason(ctx context.Context, inputFacts []ast.Atom) ([]ast.Atom, error) {
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)

	for _, fact := range inputFacts {
		workingStore.Add(fact)
	}

	if err := evaluate(r.programInfo, r.strata, r.predToStratum, workingStore); err != nil {
		return nil, fmt.Errorf("mangle: reasoner evaluation failed: %w", err)
	}

	var collectedFacts []ast.Atom
	err := workingStore.GetFacts(ast.NewQuery(ast.PredicateSym{Symbol: "solution", Arity: 2}), func(fact ast.Atom) error {
		collectedFacts = append(collectedFacts, fact)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("mangle: failed to collect reasoner results: %w", err)
	}

	return collectedFacts, nil
}
