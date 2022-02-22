package command

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDockerContextMetadataKeepAdditionalFields(t *testing.T) {
	c := DockerContext{
		Description: "test",
		AdditionalFields: map[string]interface{}{
			"foo": "bar",
		},
	}
	jsonBytes, err := json.Marshal(c)
	assert.NilError(t, err)
	assert.Equal(t, `{"Description":"test","foo":"bar"}`, string(jsonBytes))

	var c2 DockerContext
	assert.NilError(t, json.Unmarshal(jsonBytes, &c2))
	assert.Equal(t, c2.AdditionalFields["foo"], "bar")
	assert.Equal(t, c2.Description, "test")
}
