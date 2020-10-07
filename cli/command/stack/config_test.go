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

var configMergeTests = []struct {
	name              string
	skipInterpolation bool
	first             string
	second            string
	merged            string
}{
	{
		name:              "With Interpolation",
		skipInterpolation: false,
		first: `version: "3.7"
services:
  foo:
    image: busybox:latest
    command: cat file1.txt
`,
		second: `version: "3.7"
services:
  foo:
    image: busybox:${VERSION}
    command: cat file2.txt
`,
		merged: `version: "3.7"
services:
  foo:
    command:
    - cat
    - file2.txt
    image: busybox:1.0
`,
	},
	{
		name:              "Without Interpolation",
		skipInterpolation: true,
		first: `version: "3.7"
services:
  foo:
    image: busybox:latest
    command: cat file1.txt
`,
		second: `version: "3.7"
services:
  foo:
    image: busybox:${VERSION}
    command: cat file2.txt
`,
		merged: `version: "3.7"
services:
  foo:
    command:
    - cat
    - file2.txt
    image: busybox:${VERSION}
`,
	},
}

func TestConfigMergeInterpolation(t *testing.T) {

	for _, tt := range configMergeTests {
		t.Run(tt.name, func(t *testing.T) {
			firstConfig := []byte(tt.first)
			secondConfig := []byte(tt.second)

			firstConfigData, err := loader.ParseYAML(firstConfig)
			assert.NilError(t, err)
			secondConfigData, err := loader.ParseYAML(secondConfig)
			assert.NilError(t, err)

			env := map[string]string{
				"VERSION": "1.0",
			}

			cfg, err := outputConfig(composetypes.ConfigDetails{
				ConfigFiles: []composetypes.ConfigFile{
					{Config: firstConfigData, Filename: "firstConfig"},
					{Config: secondConfigData, Filename: "secondConfig"},
				},
				Environment: env,
			}, tt.skipInterpolation)
			assert.NilError(t, err)

			assert.Equal(t, cfg, tt.merged)
		})
	}

}
