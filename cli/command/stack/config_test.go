package stack

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func TestConfigWithEmptyComposeFile(t *testing.T) {
	cmd := newConfigCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetOut(ioutil.Discard)

	assert.ErrorContains(t, cmd.Execute(), `Please specify a Compose file`)
}

func TestConfigMergeUsingInterpolation(t *testing.T) {

	firstConfig := []byte(`
version: "3.7"
services:
  foo:
    image: busybox:latest
    command: cat file1.txt
`)
	secondConfig := []byte(`
version: "3.7"
services:
  foo:
    image: busybox:${VERSION}
    command: cat file2.txt
`)

	firstConfigData, err := loader.ParseYAML(firstConfig)
	assert.NilError(t, err)
	secondConfigData, err := loader.ParseYAML(secondConfig)
	assert.NilError(t, err)

	env := map[string]string{
		"VERSION": "1.0",
	}

	cfg, err := OutputConfig(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{Config: firstConfigData, Filename: "firstConfig"},
			{Config: secondConfigData, Filename: "secondConfig"},
		},
		Environment: env,
	}, false)
	assert.NilError(t, err)

	var mergedConfig = `version: "3.7"
services:
  foo:
    command:
    - cat
    - file2.txt
    image: busybox:1.0
`
	assert.Equal(t, cfg, mergedConfig)
}

func TestConfigMergeSkipInterpolation(t *testing.T) {

	firstConfig := []byte(`
version: "3.7"
services:
  foo:
    image: busybox:latest
    command: cat file1.txt
`)
	secondConfig := []byte(`
version: "3.7"
services:
  foo:
    image: busybox:${VERSION}
    command: cat file2.txt
`)

	firstConfigData, err := loader.ParseYAML(firstConfig)
	assert.NilError(t, err)
	secondConfigData, err := loader.ParseYAML(secondConfig)
	assert.NilError(t, err)

	env := map[string]string{
		"VERSION": "1.0",
	}

	cfg, err := OutputConfig(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{Config: firstConfigData, Filename: "firstConfig"},
			{Config: secondConfigData, Filename: "secondConfig"},
		},
		Environment: env,
	}, true)
	assert.NilError(t, err)

	var mergedConfig = `version: "3.7"
services:
  foo:
    command:
    - cat
    - file2.txt
    image: busybox:${VERSION}
`
	assert.Equal(t, cfg, mergedConfig)
}
