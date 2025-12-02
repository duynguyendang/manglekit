package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/sdk"
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
	// We assume safety.dl is in the current directory
	ctx := context.Background()
	client, err := sdk.NewDefault()
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}
	if err := client.Engine().LoadFromPath("examples/sre_guardrail/safety.dl"); err != nil {
		log.Fatalf("Failed to load policy: %v", err)
	}

	// 2. Define the high-risk operation
	// This function simulates an action on a Kubernetes cluster.
	deletePod := func(ctx context.Context, req KubernetesRequest) (string, error) {
		return fmt.Sprintf("Executed %s on pod in %s", req.Operation, req.Metadata.Namespace), nil
	}

	// 3. Protect the operation
	// This wraps the function with the policy engine.
	safeDeletePod := sdk.ProtectFunc(client, "k8s_guardrail", deletePod)

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
	if res, err := sdk.Call[string](ctx, safeDeletePod, reqA); err != nil {
		logger.Warn("Blocked", "error", err)
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
	if res, err := sdk.Call[string](ctx, safeDeletePod, reqB); err != nil {
		logger.Warn("Blocked", "error", err)
	} else {
		logger.Info("Success", "result", res)
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
	if res, err := sdk.Call[string](ctx, safeDeletePod, reqC); err != nil {
		logger.Warn("Blocked", "error", err)
	} else {
		logger.Info("Success", "result", res)
	}
}
