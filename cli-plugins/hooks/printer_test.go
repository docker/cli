package hooks

import (
	"bytes"
	"testing"

	"github.com/morikuni/aec"
	"gotest.tools/v3/assert"
)

func TestPrintHookMessages(t *testing.T) {
	testCases := []struct {
		messages       []string
		expectedOutput string
	}{
		{
			messages:       []string{},
			expectedOutput: "",
		},
		{
			messages: []string{"Bork!"},
			expectedOutput: aec.Bold.Apply("\nWhat's next:") + "\n" +
				"    Bork!\n",
		},
		{
			messages: []string{"Foo", "bar"},
			expectedOutput: aec.Bold.Apply("\nWhat's next:") + "\n" +
				"    Foo\n" +
				"    bar\n",
		},
	}

	for _, tc := range testCases {
		w := bytes.Buffer{}
		PrintNextSteps(&w, tc.messages)
		assert.Equal(t, w.String(), tc.expectedOutput)
	}
}
