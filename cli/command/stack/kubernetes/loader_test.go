package kubernetes

import (
	"testing"

	composetypes "github.com/docker/cli/cli/compose/types"
	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLoadStack(t *testing.T) {
	s, err := LoadStack("foo", composetypes.Config{
		Version:  "3.1",
		Filename: "banana",
		Services: []composetypes.ServiceConfig{
			{
				Name:  "foo",
				Image: "foo",
			},
			{
				Name:  "bar",
				Image: "bar",
			},
		},
	})
	assert.NilError(t, err)
	assert.DeepEqual(t, &apiv1beta1.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		Spec: apiv1beta1.StackSpec{
			ComposeFile: `version: "3.1"
services:
  bar:
    image: bar
  foo:
    image: foo
networks: {}
volumes: {}
secrets: {}
configs: {}
`,
		},
	}, s, cmpKubeAPITime)
}

// TODO: this can be removed when k8s.io/apimachinery is updated to > 1.9.0
var cmpKubeAPITime = cmp.Comparer(func(x, y *metav1.Time) bool {
	if x == nil || y == nil {
		return x == y
	}
	return x.Time.Equal(y.Time)
})
