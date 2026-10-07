package stack

import (
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func TestConfigWithEmptyComposeFile(t *testing.T) {
	cmd := newConfigCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	assert.ErrorContains(t, cmd.Execute(), `specify a Compose file`)
}

func TestConfigMergeInterpolation(t *testing.T) {
	tests := []struct {
		name              string
		skipInterpolation bool
		fileOne           string
		fileTwo           string
		expected          string
	}{
		{
			name:              "With Interpolation",
			skipInterpolation: false,
			fileOne: `version: "3.7"
services:
  foo:
    image: busybox:latest
    command: cat file1.txt
`,
			fileTwo: `version: "3.7"
services:
  foo:
    image: busybox:${VERSION}
    command: cat file2.txt
`,
			expected: `version: "3.7"
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
			fileOne: `version: "3.7"
services:
  foo:
    image: busybox:latest
    command: cat file1.txt
`,
			fileTwo: `version: "3.7"
services:
  foo:
    image: busybox:${VERSION}
    command: cat file2.txt
`,
			expected: `version: "3.7"
services:
  foo:
    command:
      - cat
      - file2.txt
    image: busybox:${VERSION}
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			firstConfigData, err := loader.ParseYAML([]byte(tc.fileOne))
			assert.Check(t, err)
			secondConfigData, err := loader.ParseYAML([]byte(tc.fileTwo))
			assert.Check(t, err)

			actual, err := outputConfig(composetypes.ConfigDetails{
				ConfigFiles: []composetypes.ConfigFile{
					{Config: firstConfigData, Filename: "firstConfig"},
					{Config: secondConfigData, Filename: "secondConfig"},
				},
				Environment: map[string]string{
					"VERSION": "1.0",
				},
			}, tc.skipInterpolation, nil)
			assert.Check(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestConfigProfiles(t *testing.T) {
	dict, err := loader.ParseYAML([]byte(`version: "3.8"
services:
  web:
    image: busybox:latest
  debug:
    image: busybox:latest
    profiles:
      - debug
`))
	assert.NilError(t, err)
	details := composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{Config: dict, Filename: "compose.yaml"},
		},
	}

	withoutProfile, err := outputConfig(details, false, nil)
	assert.NilError(t, err)
	assert.Check(t, !strings.Contains(withoutProfile, "debug:"))
	assert.Check(t, strings.Contains(withoutProfile, "web:"))

	withProfile, err := outputConfig(details, false, []string{"debug"})
	assert.NilError(t, err)
	assert.Check(t, strings.Contains(withProfile, "debug:"))
	assert.Check(t, strings.Contains(withProfile, "web:"))
}
