package sdk_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSREGuardrail(t *testing.T) {
	ctx := context.Background()

	// 1. Initialize Manglekit Client with the safety policy
	client, err := sdk.NewClient(ctx, sdk.WithPolicyPath("../policies/safety.dl"))
	require.NoError(t, err)

	// Mock fn:is_peak_hour() by loading a static fact
	err = client.Engine().LoadFacts([]string{`is_peak_hour("true")`})
	require.NoError(t, err)

	// Dummy handler
	noopHandler := func(ctx context.Context, input *v1.Pod) (*v1.Pod, error) {
		return input, nil
	}

	t.Run("Test Case 1: FAIL (High-Risk DELETE)", func(t *testing.T) {
		// Input Data: Critical Pod in Development
		podInput := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "development",
				Labels: map[string]string{
					"app":  "db-proxy",
					"tier": "critical",
				},
			},
		}

		// Create Protected Action "DELETE"
		action := sdk.ProtectFunc(client, "DELETE", noopHandler)

		// Execute
		_, err := sdk.Call[*v1.Pod](ctx, action, podInput)

		// Assertion
		assert.ErrorIs(t, err, core.ErrPolicyViolation, "Critical pod deletion should be denied")
	})

	t.Run("Test Case 2: FAIL (Production Write during Peak Hour)", func(t *testing.T) {
		// Input Data: Non-critical Pod in Production
		podInput := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "production",
				Labels: map[string]string{
					"app": "frontend",
				},
			},
		}

		// Create Protected Action "RESTART" (Not READ)
		action := sdk.ProtectFunc(client, "RESTART", noopHandler)

		// Execute
		_, err := sdk.Call[*v1.Pod](ctx, action, podInput)

		// Assertion
		assert.ErrorIs(t, err, core.ErrPolicyViolation, "Production write during peak hour should be denied")
	})

	t.Run("Test Case 3: PASS (Safe Operation)", func(t *testing.T) {
		// Input Data: Non-critical Pod in Development
		podInput := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "development",
				Labels: map[string]string{
					"app": "frontend",
				},
			},
		}

		// Create Protected Action "DELETE"
		action := sdk.ProtectFunc(client, "DELETE", noopHandler)

		// Execute
		_, err := sdk.Call[*v1.Pod](ctx, action, podInput)

		// Assertion
		assert.NoError(t, err, "Safe operation should be allowed")
	})

	t.Run("Test Case 4: PASS (Production READ during Peak Hour)", func(t *testing.T) {
		// Input Data: Pod in Production
		podInput := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "production",
				Labels: map[string]string{
					"app": "frontend",
				},
			},
		}

		// Create Protected Action "READ"
		action := sdk.ProtectFunc(client, "READ", noopHandler)

		// Execute
		_, err := sdk.Call[*v1.Pod](ctx, action, podInput)

		// Assertion
		assert.NoError(t, err, "READ operation in production should be allowed during peak hour")
	})
}
