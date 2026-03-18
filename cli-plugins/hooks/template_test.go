package hooks_test

import (
	"testing"

	"github.com/docker/cli/cli-plugins/hooks"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

// TestParseTemplate tests parsing templates as returned by plugins.
//
// It uses fixed string fixtures to lock in compatibility with existing
// plugin templates, so older formats continue to work even if we add new
// template forms.
//
// For helper-backed cases, it also verifies that templates produced by the
// current TemplateReplace* helpers parse to the same output. This lets us
// evolve the emitted template format without breaking older plugins.
func TestParseTemplate(t *testing.T) {
	type testFlag struct {
		name  string
		value string
	}
	tests := []struct {
		doc            string
		template       string // compatibility fixture; keep even if helpers emit a newer form
		templateFunc   func() string
		flags          []testFlag
		args           []string
		expectedOutput []string
	}{
		{
			doc:            "empty template",
			template:       "",
			expectedOutput: []string{""},
		},
		{
			doc:            "plain message",
			template:       "a plain template message",
			expectedOutput: []string{"a plain template message"},
		},
		{
			doc:          "subcommand name",
			template:     "hello {{.Name}}", // NOTE: fixture; do not modify without considering plugin compatibility
			templateFunc: func() string { return "hello " + hooks.TemplateReplaceSubcommandName() },

			expectedOutput: []string{"hello pull"},
		},
		{
			doc:          "single flag",
			template:     `{{flag . "tag"}}`, // NOTE: fixture; do not modify without considering plugin compatibility
			templateFunc: func() string { return hooks.TemplateReplaceFlagValue("tag") },
			flags: []testFlag{
				{name: "tag", value: "my-tag"},
			},
			expectedOutput: []string{"my-tag"},
		},
		{
			doc:      "multiple flags",
			template: `{{flag . "test-one"}} {{flag . "test2"}}`, // NOTE: fixture; do not modify without considering plugin compatibility
			templateFunc: func() string {
				return hooks.TemplateReplaceFlagValue("test-one") + " " + hooks.TemplateReplaceFlagValue("test2")
			},
			flags: []testFlag{
				{
					name:  "test-one",
					value: "value",
				},
				{
					name:  "test2",
					value: "value2",
				},
			},
			expectedOutput: []string{"value value2"},
		},
		{
			doc:            "multiple args",
			template:       `{{arg . 0}} {{arg . 1}}`, // NOTE: fixture; do not modify without considering plugin compatibility
			templateFunc:   func() string { return hooks.TemplateReplaceArg(0) + " " + hooks.TemplateReplaceArg(1) },
			args:           []string{"zero", "one"},
			expectedOutput: []string{"zero one"},
		},
		{
			doc:            "arg in sentence",
			template:       "You just pulled {{arg . 0}}", // NOTE: fixture; do not modify without considering plugin compatibility
			templateFunc:   func() string { return "You just pulled " + hooks.TemplateReplaceArg(0) },
			args:           []string{"alpine"},
			expectedOutput: []string{"You just pulled alpine"},
		},
		{
			doc:            "multiline output",
			template:       "one line\nanother line!",
			expectedOutput: []string{"one line", "another line!"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			testCmd := &cobra.Command{
				Use:  "pull",
				Args: cobra.ExactArgs(len(tc.args)),
			}
			for _, f := range tc.flags {
				_ = testCmd.Flags().String(f.name, "", "")
				err := testCmd.Flag(f.name).Value.Set(f.value)
				assert.NilError(t, err)
			}
			err := testCmd.Flags().Parse(tc.args)
			assert.NilError(t, err)

			// Validate using fixtures.
			out, err := hooks.ParseTemplate(tc.template, testCmd)
			assert.NilError(t, err)
			assert.DeepEqual(t, out, tc.expectedOutput)

			if tc.templateFunc != nil {
				// Validate using the current template function equivalent.
				out, err = hooks.ParseTemplate(tc.templateFunc(), testCmd)
				assert.NilError(t, err)
				assert.DeepEqual(t, out, tc.expectedOutput)
			}
		})
	}
}
