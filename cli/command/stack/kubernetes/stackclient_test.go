package kubernetes

import (
	"io/ioutil"
	"testing"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/gotestyourself/gotestyourself/assert"
)

func TestFromCompose(t *testing.T) {
	stackClient := &stackV1Beta1{}
	s, err := stackClient.FromCompose(ioutil.Discard, "foo", &composetypes.Config{
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
	assert.Equal(t, "foo", s.name)
	assert.Equal(t, string(`version: "3.1"
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
