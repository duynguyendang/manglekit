package engine

import (
	"fmt"
	"reflect"
	"strings"

	mangleanalysis "github.com/google/mangle/analysis"
	mangleast "github.com/google/mangle/ast"
	mangleengine "github.com/google/mangle/engine"
	manglefactstore "github.com/google/mangle/factstore"
	mangleparse "github.com/google/mangle/parse"
)

// Evaluator provides capabilities for evaluating a single Datalog rule against a Go struct.
// It is primarily used for ad-hoc rule checking or dynamic policy evaluation where a full
// rule set management overhead is not required.
type Evaluator struct {
	rule     string
	clause   mangleast.Clause
	ruleHead string // e.g., "deny", "allow", "route"
}

// NewEvaluator creates a new Evaluator instance from a Datalog rule string.
// The rule must be a valid Datalog clause (e.g., "deny(Req) :- ...").
//
// Parameters:
//   - rule: The Datalog rule string to parse and prepare.
//
// Returns:
//   - A pointer to a configured Evaluator, or an error if the rule is invalid.
func NewEvaluator(rule string) (*Evaluator, error) {
	if rule == "" {
		return nil, fmt.Errorf("rule cannot be empty")
	}

	clause, err := mangleparse.Clause(rule)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rule: %w", err)
	}

	// Extract the rule head predicate name (e.g., "deny" from "deny(Req)")
	ruleHead := ""
	if clause.Head.Predicate.Symbol != "" {
		ruleHead = clause.Head.Predicate.Symbol
	}

	return &Evaluator{
		rule:     rule,
		clause:   clause,
		ruleHead: ruleHead,
	}, nil
}

// EvaluateResult encapsulates the outcome of a rule evaluation.
type EvaluateResult struct {
	// Matched indicates whether the rule's head predicate was derived (i.e., the rule triggered).
	Matched bool
	// EntityID is the unique identifier of the entity that was evaluated.
	EntityID string
	// RuleHead is the name of the predicate that was checked (e.g., "deny").
	RuleHead string
}

// Evaluate executes the configured Datalog rule against a provided Go struct.
// It automatically converts the struct fields into Datalog facts using reflection.
//
// Parameters:
//   - entityID: A unique string identifier for the entity (e.g., request ID).
//   - entity: The Go struct to evaluate. Fields can be customized with `mangle` struct tags.
//
// Returns:
//   - An EvaluateResult indicating whether the rule matched, or an error if evaluation failed.
func (e *Evaluator) Evaluate(entityID string, entity any) (EvaluateResult, error) {
	result := EvaluateResult{
		EntityID: entityID,
		RuleHead: e.ruleHead,
	}

	// Convert entity to Mangle facts
	facts, err := structToFacts(entityID, entity)
	if err != nil {
		return result, fmt.Errorf("failed to convert entity to facts: %w", err)
	}

	// Set up the fact store and add initial facts
	store := manglefactstore.NewSimpleInMemoryStore()
	knownPredicates := make(map[mangleast.PredicateSym]mangleast.Decl)

	for _, atom := range facts {
		store.Add(atom)
		if _, ok := knownPredicates[atom.Predicate]; !ok {
			knownPredicates[atom.Predicate] = mangleast.NewSyntheticDeclFromSym(atom.Predicate)
		}
	}

	// Analyze the program
	program := []mangleast.Clause{e.clause}
	programInfo, err := mangleanalysis.AnalyzeOneUnit(mangleparse.SourceUnit{Clauses: program}, knownPredicates)
	if err != nil {
		return result, fmt.Errorf("failed to analyze program: %w", err)
	}

	// Evaluate - this materializes all consequences into the store
	if err := mangleengine.EvalProgram(programInfo, store); err != nil {
		return result, fmt.Errorf("failed to evaluate program: %w", err)
	}

	// Check if the rule head was derived for this entity
	queryStr := fmt.Sprintf(`%s("%s")`, e.ruleHead, entityID)
	queryAtom, err := mangleparse.Atom(queryStr)
	if err != nil {
		return result, fmt.Errorf("failed to parse query: %w", err)
	}

	result.Matched = store.Contains(queryAtom)
	return result, nil
}

// structToFacts converts a Go struct into Mangle Datalog atoms.
// It maps struct fields to predicates in the format: predicate("entityID", value).
//
// Rules:
//   - Exported fields are converted.
//   - `mangle` struct tag controls the predicate name.
//   - If no tag is present, the field name (lowercased) is used.
//   - Supports basic types: int, uint, float (as string), string, bool.
func structToFacts(entityID string, entity any) ([]mangleast.Atom, error) {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, fmt.Errorf("entity cannot be nil")
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("entity must be a struct, got %v", val.Kind())
	}

	typ := val.Type()
	var atoms []mangleast.Atom

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get predicate name from tag or field name
		tag := field.Tag.Get("mangle")
		if tag == "-" {
			continue // Skip fields marked with mangle:"-"
		}
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		// Create the atom based on field type
		var atom mangleast.Atom
		switch fieldVal.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.Number(fieldVal.Int()))
		case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.Number(int64(fieldVal.Uint())))
		case reflect.Float32, reflect.Float64:
			// Mangle doesn't have float, convert to string
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.String(fmt.Sprintf("%f", fieldVal.Float())))
		case reflect.String:
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.String(fieldVal.String()))
		case reflect.Bool:
			boolStr := "false"
			if fieldVal.Bool() {
				boolStr = "true"
			}
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.String(boolStr))
		default:
			// Skip unsupported types
			continue
		}
		atoms = append(atoms, atom)
	}

	if len(atoms) == 0 {
		return nil, fmt.Errorf("no valid facts could be extracted from entity")
	}

	return atoms, nil
}
