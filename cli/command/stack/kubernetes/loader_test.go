package kubernetes

import (
	"testing"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/stretchr/testify/require"
)

func TestLoadStack(t *testing.T) {
	s, err := loadStack("foo", composetypes.Config{
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
	require.Equal(t, "foo", s.name)
	require.Equal(t, string(`version: "3.1"
services:
  bar:
    image: bar
  foo:
    image: foo
networks: {}
volumes: {}
secrets: {}
configs: {}
`), s.composeFile)
}
