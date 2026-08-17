package exitcode

import (
	"fmt"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
)

func TestCodeFor(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, Success},
		{
			"policy violation",
			fmt.Errorf("wrapped: %w", core.NewPolicyViolationError("hard", "RULE_1", "blocked", "")),
			PolicyDeny,
		},
		{
			"alignment",
			fmt.Errorf("wrapped: %w", core.NewAlignmentError("no", "RULE_2")),
			PolicyDeny,
		},
		{"usage", UsageErrorf("must provide either --data or --knowledge"), Usage},
		{"usage wrapped", fmt.Errorf("outer: %w", &UsageError{Err: fmt.Errorf("inner")}), Usage},
		{"runtime", fmt.Errorf("failed to read policy file"), Runtime},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, CodeFor(tc.err))
		})
	}
}

func TestIsUsageError(t *testing.T) {
	assert.True(t, IsUsageError(UsageErrorf("bad flag")))
	assert.False(t, IsUsageError(fmt.Errorf("bad flag")))
	assert.False(t, IsUsageError(nil))
}
