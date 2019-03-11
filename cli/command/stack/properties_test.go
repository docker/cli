package stack

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	stacktypes "github.com/docker/stacks/pkg/types"
	"gotest.tools/assert"
)

func TestSubstitutePropertiesValid(t *testing.T) {
	workingDir, err := ioutil.TempDir("", "envfiles")
	assert.NilError(t, err)
	defer os.RemoveAll(workingDir)
	err = ioutil.WriteFile(filepath.Join(workingDir, ".env"), []byte(`
WEB_SCALE=2
`), 0644)
	assert.NilError(t, err)
	input := &stacktypes.ComposeInput{
		ComposeFiles: []string{
			`version: "3.0"
services:
  svc1:
    image: busybox
    command: sleep $$TIME
    environment:
      - TIME=1h
    deploy:
      replicas: ${LB_SCALE:-1}
  svc2:
    image: busybox
    command: sleep 1h
    deploy:
      replicas: ${WEB_SCALE}
`,
		},
	}
	expected := `version: "3.0"
services:
  svc1:
    image: busybox
    command: sleep $$TIME
    environment:
      - TIME=1h
    deploy:
      replicas: 1
  svc2:
    image: busybox
    command: sleep 1h
    deploy:
      replicas: 2
`
	err = substituteProperties(input, workingDir)
	assert.NilError(t, err)
	assert.Assert(t, input.ComposeFiles[0] == expected)
}
