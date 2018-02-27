package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestMarshallConfig(t *testing.T) {
	workingDir := "/foo"
	homeDir := "/bar"
	cfg := fullExampleConfig(workingDir, homeDir)
	expected := fullExampleYAML(workingDir, homeDir)

	actual, err := yaml.Marshal(cfg)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(actual))

	// Make sure the expected still
	dict, err := ParseYAML([]byte("version: '3.6'\n" + expected))
	assert.NoError(t, err)
	_, err = Load(buildConfigDetails(dict, map[string]string{}))
	assert.NoError(t, err)
}
