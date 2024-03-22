package hooks

import (
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TestParseTemplate(t *testing.T) {
	type testFlag struct {
		name  string
		value string
	}
	testCases := []struct {
		template       string
		flags          []testFlag
		args           []string
		expectedOutput string
	}{
		{
			template:       "",
			expectedOutput: "",
		},
		{
			template:       "a plain template message",
			expectedOutput: "a plain template message",
		},
		{
			template: TemplateReplaceFlagValue("tag"),
			flags: []testFlag{
				{
					name:  "tag",
					value: "my-tag",
				},
			},
			expectedOutput: "my-tag",
		},
		{
			template: TemplateReplaceFlagValue("test-one") + " " + TemplateReplaceFlagValue("test2"),
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
			expectedOutput: "value value2",
		},
		{
			template:       TemplateReplaceArg(0) + " " + TemplateReplaceArg(1),
			args:           []string{"zero", "one"},
			expectedOutput: "zero one",
		},
		{
			template:       "You just pulled " + TemplateReplaceArg(0),
			args:           []string{"alpine"},
			expectedOutput: "You just pulled alpine",
		},
	}

	for _, tc := range testCases {
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

		out, err := ParseTemplate(tc.template, testCmd)
		assert.NilError(t, err)
		assert.Equal(t, out, tc.expectedOutput)
	}
}
