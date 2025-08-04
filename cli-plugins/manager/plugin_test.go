package manager

import (
	"encoding/json"
	"errors"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestPluginMarshal(t *testing.T) {
	const jsonWithError = `{"Name":"some-plugin","Err":"something went wrong"}`
	const jsonNoError = `{"Name":"some-plugin"}`

	tests := []struct {
		doc      string
		error    error
		expected string
	}{
		{
			doc:      "no error",
			expected: jsonNoError,
		},
		{
			doc:      "regular error",
			error:    errors.New("something went wrong"),
			expected: jsonWithError,
		},
		{
			doc:      "custom error",
			error:    newPluginError("something went wrong"),
			expected: jsonWithError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			actual, err := json.Marshal(&Plugin{Name: "some-plugin", Err: tc.error})
			assert.NilError(t, err)
			assert.Check(t, is.Equal(string(actual), tc.expected))
		})
	}
}
