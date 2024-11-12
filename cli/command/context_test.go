// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package command

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDockerContextMetadataKeepAdditionalFields(t *testing.T) {
	c := DockerContext{
		Description: "test",
		AdditionalFields: map[string]any{
			"foo": "bar",
		},
	}
	jsonBytes, err := json.Marshal(c)
	assert.NilError(t, err)
	const expected = `{"Description":"test","foo":"bar"}`
	assert.Equal(t, string(jsonBytes), expected)

	var c2 DockerContext
	assert.NilError(t, json.Unmarshal(jsonBytes, &c2))
	assert.Equal(t, c2.AdditionalFields["foo"], "bar")
	assert.Equal(t, c2.Description, "test")
}
