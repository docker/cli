package loader

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestMarshallConfig(t *testing.T) {
	workingDir := "/foo"
	homeDir := "/bar"
	cfg := fullExampleConfig(workingDir, homeDir)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(cfg)
	assert.NilError(t, err)
	actual := buf.String()
	golden.Assert(t, actual, "full-example.yaml.golden")

	// Make sure the expected can be parsed.
	yamlData, err := os.ReadFile("testdata/full-example.yaml.golden")
	assert.NilError(t, err)
	dict, err := ParseYAML(yamlData)
	assert.NilError(t, err)
	_, err = Load(buildConfigDetails(dict, map[string]string{}))
	assert.NilError(t, err)
}

func TestJSONMarshallConfig(t *testing.T) {
	workingDir := "/foo"
	homeDir := "/bar"
	cfg := fullExampleConfig(workingDir, homeDir)
	actual, err := json.MarshalIndent(cfg, "", "  ")
	assert.NilError(t, err)
	golden.Assert(t, string(actual), "full-example.json.golden")

	jsonData, err := os.ReadFile("testdata/full-example.json.golden")
	assert.NilError(t, err)
	dict, err := ParseYAML(jsonData)
	assert.NilError(t, err)
	_, err = Load(buildConfigDetails(dict, map[string]string{}))
	assert.NilError(t, err)
}
