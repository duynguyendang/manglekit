package inductor

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	i := &Inductor{}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"strips triple backtick fences",
			"```datalog\nrule(a,b).\n```",
			"rule(a,b).",
		},
		{
			"strips single backtick fence",
			"```\ncontent\n```",
			"content",
		},
		{
			"strips double slash comments",
			"// this is a comment\nrule(a,b).",
			"rule(a,b).",
		},
		{
			"strips hash comments",
			"# this is a comment\nrule(a,b).",
			"rule(a,b).",
		},
		{
			"strips blank lines",
			"rule1(a).\n\n\nrule2(b).",
			"rule1(a).\nrule2(b).",
		},
		{
			"strips leading whitespace",
			"   rule(a,b).",
			"rule(a,b).",
		},
		{
			"strips trailing whitespace",
			"rule(a,b).   ",
			"rule(a,b).",
		},
		{
			"preserves valid datalog rules",
			"allow(agent, action, resource) :- true.",
			"allow(agent, action, resource) :- true.",
		},
		{
			"handles mixed fences and comments",
			"```\n// comment\nrule(a).",
			"rule(a).",
		},
		{
			"handles multiple comment lines",
			"// comment 1\n// comment 2\nrule(a).",
			"rule(a).",
		},
		{
			"handles empty input",
			"",
			"",
		},
		{
			"handles only fences",
			"```\n```",
			"",
		},
		{
			"handles only comments",
			"// comment 1\n// comment 2\n# hash comment",
			"",
		},
		{
			"preserves multiline rules",
			"parent(X,Y) :- child(X,Z), parent(Z,Y).",
			"parent(X,Y) :- child(X,Z), parent(Z,Y).",
		},
		{
			"handles interspersed content",
			"# comment\nrule1(a).\n// another\nrule2(b).",
			"rule1(a).\nrule2(b).",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := i.sanitize(tt.input)
			if got != tt.want {
				t.Errorf("sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}