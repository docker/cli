package v1beta1 // import "github.com/docker/cli/kubernetes/compose/v1beta1"

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestScale(t *testing.T) {
	testCases := []string{
		"single-redis", "redis-nginx", "redis-with-replicas", "redis-with-memory",
	}

	for _, testCase := range testCases {
		stack := Stack{
			StackImpl: StackImpl{
				Spec: StackSpec{
					ComposeFile: string(golden.Get(t, testCase+".input.yaml")),
				},
			},
		}
		scaled, err := stack.Scale("redis", 5)
		if err != nil {
			t.Error(errors.Wrap(err, "Error while scaling "+testCase))
		}
		assert.Equal(t, string(golden.Get(t, testCase+".output.yaml")), scaled.Spec.ComposeFile)
	}
}

func TestScaleError(t *testing.T) {
	stack := Stack{
		StackImpl: StackImpl{
			Spec: StackSpec{
				ComposeFile: string(golden.Get(t, "redis-nginx.input.yaml")),
			},
		},
	}
	_, err := stack.Scale("non-existing-service", 4)
	assert.EqualError(t, err, "non-existing-service not found")
}
