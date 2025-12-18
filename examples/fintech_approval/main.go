package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/sdk"
)

type CheckEligibility struct {
	User string `mangle:"user"`
}

type CheckLoan struct {
	ID string `mangle:"loan_id"`
}

func main() {
	ctx := context.Background()
	wd, _ := os.Getwd()

	// Wrapper blueprint to enable "Governance Check" via Actions
	wrapperPolicy := `
Decl user(Req, User).
Decl eligible(User).
Decl loan_id(Req, ID).
Decl deny(Req).

// Allow only if eligible. Implicitly deny otherwise (in Closed mode).
allow(Req) :- user(Req, User), eligible(User).

// Bridge for Loan Checks: Deny Req if the Loan ID is denied in knowledge base
deny(Req) :- loan_id(Req, ID), deny(ID).
`
	if err := os.WriteFile("governance.dl", []byte(wrapperPolicy), 0644); err != nil {
		panic(err)
	}
	defer os.Remove("governance.dl")

	// Combine policies
	creditPath := filepath.Join(wd, "examples/fintech_approval/credit.dl")
	creditContent, _ := os.ReadFile(creditPath)
	combined := string(creditContent) + "\n" + wrapperPolicy
	if err := os.WriteFile("combined.dl", []byte(combined), 0644); err != nil {
		panic(err)
	}
	defer os.Remove("combined.dl")

	cfg := &config.Config{
		FailureMode: config.FailureModeClosed, // Ensure Fail-Closed to block ineligible
		Policy: config.PolicyConfig{
			Path: "combined.dl",
		},
		Knowledge: config.KnowledgeConfig{
			Path: filepath.Join(wd, "examples/fintech_approval/data.ttl"),
		},
	}

	client, err := sdk.NewClientFromConfig(ctx, cfg)
	if err != nil {
		fmt.Printf("Failed to init client: %v\n", err)
		os.Exit(1)
	}
	defer client.Shutdown(ctx)

	// 5. Query for Eligibility via Action
	fmt.Println("\n--- Checking Eligibility (Recursive Logic) ---")

	checkAction := manglekit.Define(client, "check_eligibility", func(ctx context.Context, req CheckEligibility) (string, error) {
		return "eligible", nil
	})

	users := []string{"Alice", "Bob", "Charlie", "David", "Eve"}
	for _, user := range users {
		// Run Action. If denied, err will be AlignmentError.
		_, err := checkAction.Run(ctx, CheckEligibility{User: user})
		if err == nil {
			fmt.Printf("✅ %s is eligible.\n", user)
		} else {
			// In Fail-Closed, any failure (including alignment error) is a denial.
			fmt.Printf("❌ %s is NOT eligible (Blocked).\n", user)
		}
	}

	// 6. Query for Loan Denials via Action
	fmt.Println("\n--- Checking Loan Denials ---")
	checkLoanAction := manglekit.Define(client, "check_loan", func(ctx context.Context, req CheckLoan) (string, error) {
		return "approved", nil
	})

	reqs := []string{"Req1", "Req2", "Req3"}
	for _, reqID := range reqs {
		_, err := checkLoanAction.Run(ctx, CheckLoan{ID: reqID})
		if err != nil {
			fmt.Printf("❌ Loan %s is DENIED.\n", reqID)
		} else {
			fmt.Printf("✅ Loan %s is APPROVED (not denied).\n", reqID)
		}
	}
}
