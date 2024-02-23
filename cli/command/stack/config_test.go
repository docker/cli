package stack

import (
	"io"
	"testing"

	composetypes "github.com/compose-spec/compose-go/v2/types"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func TestConfigWithEmptyComposeFile(t *testing.T) {
	cmd := newConfigCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetOut(io.Discard)

	assert.ErrorContains(t, cmd.Execute(), `Please specify a Compose file`)
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
			expected: `name: firstconfig
services:
  foo:
    command:
    - cat
    - file2.txt
    image: busybox:1.0
    networks:
      default: null
networks:
  default:
    name: firstconfig_default
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
			expected: `name: firstconfig
services:
  foo:
    command:
    - cat
    - file2.txt
    image: busybox:${VERSION}
    networks:
      default: null
networks:
  default:
    name: firstconfig_default
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := outputConfig(composetypes.ConfigDetails{
				ConfigFiles: []composetypes.ConfigFile{
					{Content: []byte(tc.fileOne), Filename: "firstConfig"},
					{Content: []byte(tc.fileTwo), Filename: "secondConfig"},
				},
				Environment: map[string]string{
					"VERSION": "1.0",
				},
			}, tc.skipInterpolation)
			assert.Check(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
