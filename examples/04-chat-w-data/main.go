package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"github.com/google/mangle/analysis"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/engine"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/parse"
	"github.com/joho/godotenv"
)

// document models the structured data we want to protect.
type document struct {
	Symbol          string
	ID              string
	Department      string
	Confidentiality string
	Columns         map[string]string
	Order           []string
	Sensitive       map[string]bool
}

func main() {
	_ = godotenv.Load()

	docs := []document{
		{
			Symbol:          "doc1",
			ID:              "A123",
			Department:      "sales",
			Confidentiality: "normal",
			Columns: map[string]string{
				"customer_name": "Acme Corp",
				"email":         "contact@acme.com",
				"revenue":       "100000",
				"notes":         "Initial deal",
			},
			Order: []string{"customer_name", "email", "revenue", "notes"},
			Sensitive: map[string]bool{
				"email": true,
				"notes": true,
			},
		},
		{
			Symbol:          "doc2",
			ID:              "B456",
			Department:      "marketing",
			Confidentiality: "high",
			Columns: map[string]string{
				"lead_name": "Globex Inc",
				"email":     "sales@globex.inc",
				"score":     "95",
			},
			Order: []string{"lead_name", "email", "score"},
			Sensitive: map[string]bool{
				"email": true,
			},
		},
		{
			Symbol:          "doc3",
			ID:              "S777",
			Department:      "sales",
			Confidentiality: "restricted",
			Columns: map[string]string{
				"account":   "Initech",
				"deal_size": "250000",
				"owner":     "bsmith",
				"notes":     "Q3 expansion plan",
			},
			Order: []string{"account", "deal_size", "owner", "notes"},
			Sensitive: map[string]bool{
				"notes": true,
			},
		},
	}

	query := core.Query{
		Text: "Summarize the documents about sales and marketing",
		Meta: map[string]any{
			"user_context": map[string]any{
				"user_id":    "alice",
				"role":       "analyst",
				"department": "sales",
				"doc_id":     "A123",
				"purpose":    "analytics",
			},
		},
	}

	policyStore, err := buildFactStore(docs, query)
	if err != nil {
		log.Fatalf("failed to build fact store: %v", err)
	}

	programInfo, strata, predToStratum, err := loadPolicy()
	if err != nil {
		log.Fatalf("failed to load policy: %v", err)
	}

	if _, err := engine.EvalStratifiedProgramWithStats(programInfo, strata, predToStratum, policyStore); err != nil {
		log.Fatalf("failed to evaluate policy: %v", err)
	}

	allowedDocs, err := collectSet(policyStore, "can_chat_with_data")
	if err != nil {
		log.Fatalf("failed to collect can_chat_with_data facts: %v", err)
	}
	deniedDocList, err := collectSet(policyStore, "deny_retrieve")
	if err != nil {
		log.Fatalf("failed to collect deny_retrieve facts: %v", err)
	}
	retrievableDocs, err := collectSet(policyStore, "can_retrieve")
	if err != nil {
		log.Fatalf("failed to collect can_retrieve facts: %v", err)
	}
	hasVisible, err := collectSet(policyStore, "has_visible_column")
	if err != nil {
		log.Fatalf("failed to collect has_visible_column facts: %v", err)
	}
	visibleColumns, err := collectMulti(policyStore, "visible_column")
	if err != nil {
		log.Fatalf("failed to collect visible columns: %v", err)
	}
	maskedColumns, err := collectMaskInstructions(policyStore)
	if err != nil {
		log.Fatalf("failed to collect mask instructions: %v", err)
	}

	allowedSet := make(map[string]struct{}, len(allowedDocs))
	for _, symbol := range allowedDocs {
		allowedSet[symbol] = struct{}{}
	}

	allowedIDs := make(map[string]struct{}, len(allowedDocs))
	for _, doc := range docs {
		if _, ok := allowedSet[doc.Symbol]; ok {
			allowedIDs[doc.ID] = struct{}{}
		}
	}

	deniedReasons := buildDeniedReasons(docs, allowedDocs, deniedDocList, retrievableDocs, hasVisible)

	var citations []core.Citation
	for _, doc := range docs {
		if _, ok := allowedSet[doc.Symbol]; !ok {
			continue
		}
		snippet := renderSnippet(doc, visibleColumns[doc.Symbol], maskedColumns[doc.Symbol])
		citations = append(citations, core.Citation{ID: doc.ID, Snippet: snippet})
	}

	answer := core.Answer{
		Text:      fmt.Sprintf("Found %d document(s) after enforcing Mangle policy.", len(citations)),
		Citations: citations,
	}
	if len(deniedReasons) > 0 {
		answer.Meta = map[string]any{"mangle_denied_reasons": deniedReasons}
	}

	fmt.Println(answer.Text)
	fmt.Println("\nCitations:")
	for _, citation := range answer.Citations {
		fmt.Printf("- ID: %s, Snippet: %s\n", citation.ID, citation.Snippet)
	}

	if len(deniedReasons) > 0 {
		fmt.Println("\nDenied documents:")
		ids := make([]string, 0, len(deniedReasons))
		for id := range deniedReasons {
			if _, ok := allowedIDs[id]; ok {
				continue
			}
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			fmt.Printf("- ID: %s, Reasons: %s\n", id, strings.Join(deniedReasons[id], "; "))
		}
	}
}

func buildFactStore(docs []document, query core.Query) (*factstore.SimpleInMemoryStore, error) {
	store := factstore.NewSimpleInMemoryStore()

	for _, doc := range docs {
		store.Add(ast.NewAtom("doc", ast.String(doc.Symbol)))
		store.Add(ast.NewAtom("doc_id", ast.String(doc.Symbol), ast.String(doc.ID)))
		store.Add(ast.NewAtom("doc_department", ast.String(doc.Symbol), ast.String(doc.Department)))
		store.Add(ast.NewAtom("doc_confidentiality", ast.String(doc.Symbol), ast.String(doc.Confidentiality)))
		for _, column := range doc.Order {
			store.Add(ast.NewAtom("column", ast.String(doc.Symbol), ast.String(column)))
			if doc.Sensitive[column] {
				store.Add(ast.NewAtom("sensitive_column", ast.String(doc.Symbol), ast.String(column)))
			}
		}
	}

	userCtx, _ := query.Meta["user_context"].(map[string]any)
	for key, value := range userCtx {
		store.Add(ast.NewAtom("user_attribute", ast.String(key), ast.String(fmt.Sprintf("%v", value))))
	}

	return &store, nil
}

func loadPolicy() (*analysis.ProgramInfo, []analysis.Nodeset, map[ast.PredicateSym]int, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, nil, nil, fmt.Errorf("failed to resolve current file path")
	}
	policyPath := filepath.Join(filepath.Dir(currentFile), "policy", "access_control.dlog")

	f, err := os.Open(policyPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open policy file: %w", err)
	}
	defer f.Close()

	unit, err := parse.Unit(f)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse policy file: %w", err)
	}

	programInfo, err := analysis.Analyze([]parse.SourceUnit{unit}, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to analyze policy: %w", err)
	}

	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to stratify policy: %w", err)
	}

	return programInfo, strata, predToStratum, nil
}

func collectSet(store factstore.ReadOnlyFactStore, predicate string) ([]string, error) {
	query := ast.NewQuery(ast.PredicateSym{Symbol: predicate, Arity: 1})
	values := make(map[string]struct{})
	if err := store.GetFacts(query, func(atom ast.Atom) error {
		if len(atom.Args) != 1 {
			return nil
		}
		val, err := termToString(atom.Args[0])
		if err != nil {
			return err
		}
		values[val] = struct{}{}
		return nil
	}); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(values))
	for v := range values {
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

func collectMulti(store factstore.ReadOnlyFactStore, predicate string) (map[string][]string, error) {
	query := ast.NewQuery(ast.PredicateSym{Symbol: predicate, Arity: 2})
	results := make(map[string]map[string]struct{})
	if err := store.GetFacts(query, func(atom ast.Atom) error {
		if len(atom.Args) != 2 {
			return nil
		}
		key, err := termToString(atom.Args[0])
		if err != nil {
			return err
		}
		val, err := termToString(atom.Args[1])
		if err != nil {
			return err
		}
		if _, ok := results[key]; !ok {
			results[key] = make(map[string]struct{})
		}
		results[key][val] = struct{}{}
		return nil
	}); err != nil {
		return nil, err
	}

	out := make(map[string][]string, len(results))
	for key, vals := range results {
		items := make([]string, 0, len(vals))
		for v := range vals {
			items = append(items, v)
		}
		sort.Strings(items)
		out[key] = items
	}
	return out, nil
}

func collectMaskInstructions(store factstore.ReadOnlyFactStore) (map[string]map[string]string, error) {
	query := ast.NewQuery(ast.PredicateSym{Symbol: "masked_column", Arity: 3})
	results := make(map[string]map[string]string)
	if err := store.GetFacts(query, func(atom ast.Atom) error {
		if len(atom.Args) != 3 {
			return nil
		}
		docID, err := termToString(atom.Args[0])
		if err != nil {
			return err
		}
		column, err := termToString(atom.Args[1])
		if err != nil {
			return err
		}
		mode, err := termToString(atom.Args[2])
		if err != nil {
			return err
		}
		if _, ok := results[docID]; !ok {
			results[docID] = make(map[string]string)
		}
		results[docID][column] = mode
		return nil
	}); err != nil {
		return nil, err
	}
	return results, nil
}

func buildDeniedReasons(docs []document, allowed, denied, retrievable, visible []string) map[string][]string {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, symbol := range allowed {
		allowedSet[symbol] = struct{}{}
	}

	deniedSet := make(map[string]struct{}, len(denied))
	for _, symbol := range denied {
		deniedSet[symbol] = struct{}{}
	}

	retrievableSet := make(map[string]struct{}, len(retrievable))
	for _, symbol := range retrievable {
		retrievableSet[symbol] = struct{}{}
	}

	visibleSet := make(map[string]struct{}, len(visible))
	for _, symbol := range visible {
		visibleSet[symbol] = struct{}{}
	}

	reasons := make(map[string][]string)
	for _, doc := range docs {
		if _, ok := allowedSet[doc.Symbol]; ok {
			continue
		}

		var docReasons []string
		if _, restricted := deniedSet[doc.Symbol]; restricted {
			docReasons = append(docReasons, "restricted document requires privileged access")
		}
		if _, canRetrieve := retrievableSet[doc.Symbol]; !canRetrieve {
			if _, restricted := deniedSet[doc.Symbol]; !restricted {
				docReasons = append(docReasons, "no retrieval entitlement matched user context")
			}
		} else if _, hasVisible := visibleSet[doc.Symbol]; !hasVisible {
			docReasons = append(docReasons, "retrievable but no visible columns after policy")
		}

		if len(docReasons) == 0 {
			continue
		}
		reasons[doc.ID] = docReasons
	}

	return reasons
}

func termToString(term ast.Term) (string, error) {
	constant, ok := term.(ast.Constant)
	if !ok {
		return "", fmt.Errorf("term is not a constant: %T", term)
	}
	if v, err := constant.StringValue(); err == nil {
		return v, nil
	}
	if v, err := constant.NameValue(); err == nil {
		return v, nil
	}
	return "", fmt.Errorf("unsupported constant type: %v", constant.Type)
}

func renderSnippet(doc document, visible []string, masks map[string]string) string {
	visibleSet := make(map[string]struct{}, len(visible))
	for _, column := range visible {
		visibleSet[column] = struct{}{}
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("department=%s", doc.Department))
	parts = append(parts, fmt.Sprintf("confidentiality=%s", doc.Confidentiality))

	for _, column := range doc.Order {
		if _, ok := visibleSet[column]; ok {
			value := doc.Columns[column]
			if mode, masked := masks[column]; masked {
				value = fmt.Sprintf("<%s>", mode)
			}
			parts = append(parts, fmt.Sprintf("%s=%s", column, value))
			continue
		}
		if mode, masked := masks[column]; masked {
			parts = append(parts, fmt.Sprintf("%s=<%s>", column, mode))
		}
	}

	return strings.Join(parts, ", ")
}
