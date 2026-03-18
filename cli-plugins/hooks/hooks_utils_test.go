package hooks_test

import (
	"testing"

	"github.com/docker/cli/cli-plugins/hooks"
)

func TestTemplateHelpers(t *testing.T) {
	tests := []struct {
		doc  string
		got  func() string
		want string
	}{
		{
			doc:  "subcommand name",
			got:  hooks.TemplateReplaceSubcommandName,
			want: `{{.Name}}`,
		},
		{
			doc: "flag value",
			got: func() string {
				return hooks.TemplateReplaceFlagValue("name")
			},
			want: `{{flag . "name"}}`,
		},
		{
			doc: "arg",
			got: func() string {
				return hooks.TemplateReplaceArg(0)
			},
			want: `{{arg . 0}}`,
		},
		{
			doc: "arg",
			got: func() string {
				return hooks.TemplateReplaceArg(3)
			},
			want: `{{arg . 3}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			if got := tc.got(); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
