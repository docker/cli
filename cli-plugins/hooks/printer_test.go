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
		messages []hooks.Response
		expected string
	}{
		{doc: "no messages"},
		{
			doc: "no lines",
			messages: []hooks.Response{
				{Type: hooks.GenericMessage},
				{Type: hooks.NextSteps},
			},
		},
		{
			doc: "generic message",
			messages: []hooks.Response{
				{Type: hooks.GenericMessage, Template: "Session active"},
			},
			expected: "\nSession active\n",
		},
		{
			doc: "preserve message formatting",
			messages: []hooks.Response{
				{Type: hooks.GenericMessage, Template: "Session active\n\n  Details"},
			},
			expected: "\nSession active\n\n  Details\n",
		},
		{
			doc: "next steps",
			messages: []hooks.Response{
				{Type: hooks.NextSteps, Template: "Try another command"},
			},
			expected: header + "    Try another command\n",
		},
		{
			doc: "group interleaved types and preserve order within each type",
			messages: []hooks.Response{
				{Type: hooks.NextSteps, Template: "First suggestion"},
				{Type: hooks.GenericMessage, Template: "Session active"},
				{Type: hooks.NextSteps, Template: "Second suggestion\nThird suggestion"},
				{Type: hooks.GenericMessage, Template: "Another status"},
			},
			expected: "\nSession active\nAnother status\n" + header + "    First suggestion\n    Second suggestion\n    Third suggestion\n",
		},
		{
			doc: "unknown type",
			messages: []hooks.Response{
				{Type: hooks.ResponseType(99), Template: "Ignored"},
			},
		},
		{
			doc: "unknown type does not affect known messages",
			messages: []hooks.Response{
				{Type: hooks.ResponseType(99), Template: "Ignored"},
				{Type: hooks.GenericMessage, Template: "Session active"},
				{Type: hooks.NextSteps, Template: "Try another command"},
			},
			expected: "\nSession active\n" + header + "    Try another command\n",
		},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			var w strings.Builder
			hooks.PrintMessages(&w, tc.messages, func(message hooks.Response) []string {
				if message.Template == "" {
					return nil
				}
				lines, err := hooks.ParseTemplate(message.Template, nil)
				assert.NilError(t, err)
				return lines
			})
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
