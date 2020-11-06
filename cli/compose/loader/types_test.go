package loader

import (
	"encoding/json"
	"testing"

	yaml "gopkg.in/yaml.v2"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestMarshallConfig(t *testing.T) {
	workingDir := "/foo"
	homeDir := "/bar"
	cfg := fullExampleConfig(workingDir, homeDir)
	expected := fullExampleYAML(workingDir)

	actual, err := yaml.Marshal(cfg)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(expected, string(actual)))

	// Make sure the expected still
	dict, err := ParseYAML([]byte("version: '3.9'\n" + expected))
	assert.NilError(t, err)
	_, err = Load(buildConfigDetails(dict, map[string]string{}))
	assert.NilError(t, err)
}

func TestJSONMarshallConfig(t *testing.T) {
	workingDir := "/foo"
	homeDir := "/bar"
	cfg := fullExampleConfig(workingDir, homeDir)
	expected := fullExampleJSON(workingDir)

	actual, err := json.MarshalIndent(cfg, "", "  ")
	assert.NilError(t, err)
	assert.Check(t, is.Equal(expected, string(actual)))

	dict, err := ParseYAML([]byte(expected))
	assert.NilError(t, err)
	_, err = Load(buildConfigDetails(dict, map[string]string{}))
	assert.NilError(t, err)
}
