package engine

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	mangleanalysis "codeberg.org/TauCeti/mangle-go/analysis"
	mangleast "codeberg.org/TauCeti/mangle-go/ast"
	mangleengine "codeberg.org/TauCeti/mangle-go/engine"
	manglefactstore "codeberg.org/TauCeti/mangle-go/factstore"
	mangleparse "codeberg.org/TauCeti/mangle-go/parse"
)

// Note: SimpleInMemoryStore is a value type, not a pointer.
// Pooling would require significant refactoring or wrapper types.
// For high-throughput scenarios, prefer PolicyEngine which supports
// shared fact stores and incremental evaluation.

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

	// Create fact store for this evaluation
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
//   - Supports nested structs (recursively flattened with prefix).
//   - Supports maps (treated as key-value pairs).
//   - Supports slices (each element flattened with index suffix).
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

	atoms, err := structToFactsRecursive(entityID, entity, "")
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return nil, fmt.Errorf("no valid facts could be extracted from entity")
	}
	return atoms, nil
}

func structToFactsRecursive(entityID string, entity any, prefix string) ([]mangleast.Atom, error) {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, fmt.Errorf("entity cannot be nil")
		}
		val = val.Elem()
	}

	var atoms []mangleast.Atom
	var errs []error

	// collectErr accumulates field-level errors so we don't silently drop them.
	collectErr := func(label string, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("field %s: %w", label, err))
		}
	}

	switch val.Kind() {
	case reflect.Struct:
		typ := val.Type()
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			fieldVal := val.Field(i)

			if !field.IsExported() {
				continue
			}

			tag := field.Tag.Get("mangle")
			if tag == "-" {
				continue
			}
			if tag == "" {
				tag = strings.ToLower(field.Name)
			}
			if prefix != "" {
				tag = prefix + "_" + tag
			}

			fieldAtoms, err := structToFactsRecursive(entityID, fieldVal.Interface(), tag)
			collectErr(tag, err)
			atoms = append(atoms, fieldAtoms...)
		}

	case reflect.Map:
		iter := val.MapRange()
		for iter.Next() {
			key := fmt.Sprintf("%v", iter.Key().Interface())
			value := iter.Value().Interface()
			predicate := prefix + "_" + key
			valueAtoms, err := structToFactsRecursive(entityID, value, predicate)
			collectErr(predicate, err)
			atoms = append(atoms, valueAtoms...)
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i).Interface()
			elemPrefix := fmt.Sprintf("%s_%d", prefix, i)
			elemAtoms, err := structToFactsRecursive(entityID, elem, elemPrefix)
			collectErr(elemPrefix, err)
			atoms = append(atoms, elemAtoms...)
		}

	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		atoms = append(atoms, mangleast.NewAtom(prefix, mangleast.String(entityID), mangleast.Number(val.Int())))

	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		atoms = append(atoms, mangleast.NewAtom(prefix, mangleast.String(entityID), mangleast.Number(int64(val.Uint()))))

	case reflect.Float32, reflect.Float64:
		atoms = append(atoms, mangleast.NewAtom(prefix, mangleast.String(entityID), mangleast.String(fmt.Sprintf("%f", val.Float()))))

	case reflect.String:
		atoms = append(atoms, mangleast.NewAtom(prefix, mangleast.String(entityID), mangleast.String(val.String())))

	case reflect.Bool:
		boolStr := "false"
		if val.Bool() {
			boolStr = "true"
		}
		atoms = append(atoms, mangleast.NewAtom(prefix, mangleast.String(entityID), mangleast.String(boolStr)))
	}

	if len(atoms) == 0 && val.Kind() == reflect.Struct {
		return nil, fmt.Errorf("no valid facts could be extracted from entity")
	}

	// Return any accumulated field-level errors. The atoms collected so far
	// are still returned — callers that only check atoms will get partial
	// results, while callers that check the error see what was skipped.
	if len(errs) > 0 {
		return atoms, errors.Join(errs...)
	}
	return atoms, nil
}
