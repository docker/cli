package stack

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	composetypes "github.com/docker/stacks/pkg/compose/types"
	stacktypes "github.com/docker/stacks/pkg/types"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestLoadEnvPermutations(t *testing.T) {
	workingDir, err := ioutil.TempDir("", "envfiles")
	assert.NilError(t, err)
	defer os.RemoveAll(workingDir)
	envFileName := "foo.env"
	err = ioutil.WriteFile(filepath.Join(workingDir, envFileName), []byte(`
EXPLICIT=explicit-set-in-foo.env
IMPLICIT=implicit-set-in-foo.env
UNLISTED=unlisted-set-in-foo.env
BOTH=both-set-in-foo.env
`), 0644)
	assert.NilError(t, err)
	err = ioutil.WriteFile(filepath.Join(workingDir, ".env"), []byte(`
BOTH=both-set-in.env
`), 0644)
	assert.NilError(t, err)
	setinstack := "setinstack"
	input := &stacktypes.StackCreate{
		Spec: stacktypes.StackSpec{
			Services: composetypes.Services{
				composetypes.ServiceConfig{
					EnvFile: composetypes.StringList{
						envFileName,
					},
					Environment: composetypes.MappingWithEquals{
						"EXPLICIT": &setinstack,
						"IMPLICIT": nil,
						"BOTH":     nil,
						"UNSET":    nil,
					},
				},
			},
		},
	}
	err = loadEnvFiles(input, workingDir)
	assert.NilError(t, err)
	assert.Assert(t, input.Spec.Services[0].Environment["EXPLICIT"] != nil)
	assert.Check(t, is.Equal(*input.Spec.Services[0].Environment["EXPLICIT"], "setinstack"))
	assert.Assert(t, input.Spec.Services[0].Environment["IMPLICIT"] != nil)
	assert.Equal(t, *input.Spec.Services[0].Environment["IMPLICIT"], "implicit-set-in-foo.env")
	assert.Assert(t, input.Spec.Services[0].Environment["UNLISTED"] != nil)
	assert.Equal(t, *input.Spec.Services[0].Environment["UNLISTED"], "unlisted-set-in-foo.env")
	assert.Assert(t, input.Spec.Services[0].Environment["BOTH"] != nil)
	assert.Equal(t, *input.Spec.Services[0].Environment["BOTH"], "both-set-in.env")
	assert.Assert(t, input.Spec.Services[0].Environment["UNSET"] == nil)
}
