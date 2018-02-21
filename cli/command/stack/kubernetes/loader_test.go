package kubernetes

import (
	"testing"

	composetypes "github.com/docker/cli/cli/compose/types"
	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	require.Equal(t, &apiv1beta1.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		Spec: apiv1beta1.StackSpec{
			ComposeFile: string(`version: "3.1"
services:
  bar:
    image: bar
  foo:
    image: foo
networks: {}
volumes: {}
secrets: {}
configs: {}
`),
		},
	}, s)
}
