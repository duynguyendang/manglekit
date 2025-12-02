package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/duynguyendang/manglekit/engine"
	"github.com/duynguyendang/manglekit/engine/resources"
)

func main() {
	// 1. Initialize Engine
	// We use NewMangleRuntime directly to access low-level query capabilities
	eng := engine.NewMangleRuntime()

	// 2. Load Policy
	wd, _ := os.Getwd()
	policyPath := filepath.Join(wd, "examples/fintech_approval/credit.dl")
	fmt.Printf("Loading policy from: %s\n", policyPath)
	if err := eng.Load(policyPath); err != nil {
		fmt.Printf("Failed to load policy: %v\n", err)
		os.Exit(1)
	}

	// 3. Load Data from RDF
	dataPath := filepath.Join(wd, "examples/fintech_approval/data.ttl")
	fmt.Printf("Loading RDF data from: %s\n", dataPath)
	facts, err := resources.LoadFromPath(dataPath)
	if err != nil {
		fmt.Printf("Failed to load RDF data: %v\n", err)
		os.Exit(1)
	}

	// 4. Inject Facts into Engine
	if err := eng.LoadFacts(facts); err != nil {
		fmt.Printf("Failed to load facts into engine: %v\n", err)
		os.Exit(1)
	}

	// 5. Query for Eligibility
	fmt.Println("\n--- Checking Eligibility (Recursive Logic) ---")
	users := []string{"Alice", "Bob", "Charlie", "David", "Eve"}
	for _, user := range users {
		query := fmt.Sprintf("eligible(\"%s\")", user)
		eligible, err := eng.ExecuteQuery(nil, query)
		if err != nil {
			fmt.Printf("Error querying eligibility for %s: %v\n", user, err)
			continue
		}
		if eligible {
			fmt.Printf("✅ %s is eligible.\n", user)
		} else {
			fmt.Printf("❌ %s is NOT eligible.\n", user)
		}
	}

	// 6. Query for Loan Denials
	fmt.Println("\n--- Checking Loan Denials ---")
	reqs := []string{"Req1", "Req2", "Req3"}
	for _, req := range reqs {
		query := fmt.Sprintf("deny(\"%s\")", req)
		denied, err := eng.ExecuteQuery(nil, query)
		if err != nil {
			fmt.Printf("Error querying denial for %s: %v\n", req, err)
			continue
		}
		if denied {
			fmt.Printf("❌ Loan %s is DENIED.\n", req)
		} else {
			fmt.Printf("✅ Loan %s is APPROVED (not denied).\n", req)
		}
	}
}
