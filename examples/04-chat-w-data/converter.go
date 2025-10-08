package main

import (
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/parse"
)

// CustomFactConverter translates query-time information into a 'request' fact.
type CustomFactConverter struct{}

// Predicates declares that this converter only generates the 'request/2' predicate.
func (c *CustomFactConverter) Predicates() []ast.PredicateSym {
	return []ast.PredicateSym{
		{Symbol: "request", Arity: 2},
	}
}

// ToFacts generates just one fact: request(User, DocID).
func (c *CustomFactConverter) ToFacts(input any) ([]ast.Atom, error) {
	q, ok := input.(core.Query)
	if !ok {
		return nil, fmt.Errorf("expected core.Query, got %T", input)
	}
	if q.Meta == nil {
		return nil, nil
	}

	requestAtom, err := parse.Atom(`request(_,_)`)
	if err != nil {
		return nil, fmt.Errorf("internal error: could not parse dummy request atom: %w", err)
	}
	requestPredicate := requestAtom.Predicate

	reqData, ok := q.Meta["request"].(map[string]string)
	if !ok {
		return nil, fmt.Errorf("query metadata missing 'request' object")
	}
	user, userOk := reqData["user"]
	docID, docIDOk := reqData["doc_id"]
	if !userOk || !docIDOk {
		return nil, fmt.Errorf("request object in metadata is missing 'user' or 'doc_id'")
	}

	atoms := []ast.Atom{
		{
			Predicate: requestPredicate,
			Args:      []ast.BaseTerm{ast.String(user), ast.String(docID)},
		},
	}
	return atoms, nil
}