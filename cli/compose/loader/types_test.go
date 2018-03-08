package loader

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	yaml "gopkg.in/yaml.v2"
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
	dict, err := ParseYAML([]byte("version: '3.6'\n" + expected))
	assert.NilError(t, err)
	_, err = Load(buildConfigDetails(dict, map[string]string{}))
	assert.NilError(t, err)
}
