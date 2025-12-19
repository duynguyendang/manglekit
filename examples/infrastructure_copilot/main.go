package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
)

// Metadata represents Kubernetes object metadata
type Metadata struct {
	Namespace string            `mangle:"namespace"`
	Labels    map[string]string `mangle:"labels"`
}

// KubernetesRequest represents the input for the guardrail
type KubernetesRequest struct {
	Operation  string   `mangle:"req_operation"`
	IsPeakHour string   `mangle:"is_peak_hour"`
	Metadata   Metadata `mangle:"metadata"`
}

func main() {
	// 1. Initialize Manglekit Client
	ctx := context.Background()

	// Use Facade. Load blueprint via option.
	client := manglekit.Must(manglekit.NewClient(
		ctx,
		manglekit.WithBlueprintPath("examples/infrastructure_copilot/safety.dl"),
	))

	// 2. Define the high-risk operation
	// This function simulates an action on a Kubernetes cluster.
	deletePod := func(ctx context.Context, req KubernetesRequest) (string, error) {
		return fmt.Sprintf("Executed %s on pod in %s", req.Operation, req.Metadata.Namespace), nil
	}

	// 3. Protect the operation
	// This wraps the function with the policy engine.
	action := manglekit.Define(client, "k8s_guardrail", deletePod)

	// Get the logger from the client
	logger := client.Logger()

	// 4. Test Cases

	// Case A: Allowed Operation (Read in Production)
	logger.Info("--- Case A: Read in Production (Allowed) ---")
	reqA := KubernetesRequest{
		Operation:  "READ",
		IsPeakHour: "true",
		Metadata: Metadata{
			Namespace: "production",
			Labels:    map[string]string{"app": "web"},
		},
	}
	if res, err := action.Run(ctx, reqA); err != nil {
		// Expect success
		log.Fatalf("Unexpected block for Case A: %v", err)
	} else {
		logger.Info("Success", "result", res)
	}

	// Case B: Denied Operation (Delete Critical Pod)
	logger.Info("--- Case B: Delete Critical Pod (Denied) ---")
	reqB := KubernetesRequest{
		Operation:  "DELETE",
		IsPeakHour: "false",
		Metadata: Metadata{
			Namespace: "default",
			Labels:    map[string]string{"tier": "critical"},
		},
	}
	if _, err := action.Run(ctx, reqB); err == nil {
		log.Fatalf("Unexpected success for Case B (Should be blocked)")
	} else {
		// Expect AlignmentError
		var pve *core.AlignmentError
		if errors.As(err, &pve) {
			logger.Warn("Blocked as expected", "error", err)
		} else {
			log.Fatalf("Expected AlignmentError but got: %v", err)
		}
	}

	// Case C: Denied Operation (Write in Production during Peak Hour)
	logger.Info("--- Case C: Update in Production during Peak Hour (Denied) ---")
	reqC := KubernetesRequest{
		Operation:  "UPDATE",
		IsPeakHour: "true",
		Metadata: Metadata{
			Namespace: "production",
			Labels:    map[string]string{"app": "web"},
		},
	}
	if _, err := action.Run(ctx, reqC); err == nil {
		log.Fatalf("Unexpected success for Case C (Should be blocked)")
	} else {
		logger.Warn("Blocked as expected", "error", err)
	}
}
