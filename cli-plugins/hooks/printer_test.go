package hooks_test

import (
	"strings"
	"testing"

	"github.com/docker/cli/cli-plugins/hooks"
	"gotest.tools/v3/assert"
)

func TestPrintHookMessages(t *testing.T) {
	const header = "\x1b[1m\nWhat's next:\x1b[0m\n"

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
	}
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			var w strings.Builder
			hooks.PrintNextSteps(&w, tc.messages)
			assert.Equal(t, w.String(), tc.expectedOutput)
		})
	}
}
