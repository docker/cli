package stack

import (
	"os"
	"testing"

	"github.com/docker/docker/pkg/stringid"
	stacktypes "github.com/docker/stacks/pkg/types"

	"gotest.tools/assert"
)

func TestBuildEnvironmentMissing(t *testing.T) {
	randomVariableKey := stringid.GenerateRandomID()

	stackCreate := &stacktypes.StackCreate{
		Spec: stacktypes.StackSpec{
			PropertyValues: []string{
				randomVariableKey,
			},
		},
	}

	err := buildEnvironment(stackCreate)
	assert.ErrorContains(t, err, "you must specify a value for")
}

func TestBuildEnvironmentDefault(t *testing.T) {
	randomVariableKey := stringid.GenerateRandomID()

	stackCreate := &stacktypes.StackCreate{
		Spec: stacktypes.StackSpec{
			PropertyValues: []string{
				randomVariableKey + "=foo",
			},
		},
	}

	err := buildEnvironment(stackCreate)
	assert.NilError(t, err)
	assert.Equal(t, stackCreate.Spec.PropertyValues[0], randomVariableKey+"=foo")
}
func TestBuildEnvironmentFromEnv(t *testing.T) {
	randomVariableKey := stringid.GenerateRandomID()
	os.Setenv(randomVariableKey, "bar")

	stackCreate := &stacktypes.StackCreate{
		Spec: stacktypes.StackSpec{
			PropertyValues: []string{
				randomVariableKey + "=foo",
			},
		},
	}

	err := buildEnvironment(stackCreate)
	assert.NilError(t, err)
	assert.Equal(t, stackCreate.Spec.PropertyValues[0], randomVariableKey+"=bar")

}
