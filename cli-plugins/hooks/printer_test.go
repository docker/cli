package hooks_test

import (
	"strings"
	"testing"

	"github.com/docker/cli/cli-plugins/hooks"
	"gotest.tools/v3/assert"
)

func TestPrintMessages(t *testing.T) {
	const header = "\n\x1b[1mWhat's next:\x1b[0m\n"

	for _, tc := range []struct {
		doc      string
		messages []hooks.EvaluatedMessage
		expected string
	}{
		{doc: "no messages"},
		{
			doc: "no lines",
			messages: []hooks.EvaluatedMessage{
				{Type: hooks.GenericMessage},
				{Type: hooks.NextSteps, Lines: []string{}},
			},
		},
		{
			doc: "generic message",
			messages: []hooks.EvaluatedMessage{
				{Type: hooks.GenericMessage, Lines: []string{"Session active"}},
			},
			expected: "\nSession active\n",
		},
		{
			doc: "preserve message formatting",
			messages: []hooks.EvaluatedMessage{
				{Type: hooks.GenericMessage, Lines: []string{"Session active", "", "  Details"}},
			},
			expected: "\nSession active\n\n  Details\n",
		},
		{
			doc: "next steps",
			messages: []hooks.EvaluatedMessage{
				{Type: hooks.NextSteps, Lines: []string{"Try another command"}},
			},
			expected: header + "    Try another command\n",
		},
		{
			doc: "group interleaved types and preserve order within each type",
			messages: []hooks.EvaluatedMessage{
				{Type: hooks.NextSteps, Lines: []string{"First suggestion"}},
				{Type: hooks.GenericMessage, Lines: []string{"Session active"}},
				{Type: hooks.NextSteps, Lines: []string{"Second suggestion", "Third suggestion"}},
				{Type: hooks.GenericMessage, Lines: []string{"Another status"}},
			},
			expected: "\nSession active\nAnother status\n" + header + "    First suggestion\n    Second suggestion\n    Third suggestion\n",
		},
		{
			doc: "unknown type",
			messages: []hooks.EvaluatedMessage{
				{Type: hooks.ResponseType(99), Lines: []string{"Ignored"}},
			},
		},
		{
			doc: "unknown type does not affect known messages",
			messages: []hooks.EvaluatedMessage{
				{Type: hooks.ResponseType(99), Lines: []string{"Ignored"}},
				{Type: hooks.GenericMessage, Lines: []string{"Session active"}},
				{Type: hooks.NextSteps, Lines: []string{"Try another command"}},
			},
			expected: "\nSession active\n" + header + "    Try another command\n",
		},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			var w strings.Builder
			hooks.PrintMessages(&w, tc.messages)
			assert.Equal(t, w.String(), tc.expected)
		})
	}
}

func TestPrintNextSteps(t *testing.T) {
	const header = "\n\x1b[1mWhat's next:\x1b[0m\n"

	tests := []struct {
		doc            string
		messages       []string
		expectedOutput string
	}{
		{
			doc:            "no messages",
			messages:       nil,
			expectedOutput: "",
		},
		{
			doc:      "single message",
			messages: []string{"Bork!"},
			expectedOutput: header +
				"    Bork!\n",
		},
		{
			doc:      "multiple messages",
			messages: []string{"Foo", "bar"},
			expectedOutput: header +
				"    Foo\n" +
				"    bar\n",
		},
		{
			doc:            "preserve empty and indented lines",
			messages:       []string{"", "  Details", ""},
			expectedOutput: header + "    \n      Details\n    \n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			var w strings.Builder
			hooks.PrintNextSteps(&w, tc.messages)
			assert.Equal(t, w.String(), tc.expectedOutput)
		})
	}
}
