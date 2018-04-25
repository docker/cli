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
	assert.Equal(t, string(`version: "3.5"
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

func TestFromComposeUnsupportedVersion(t *testing.T) {
	stackClient := &stackV1Beta1{}
	_, err := stackClient.FromCompose(ioutil.Discard, "foo", &composetypes.Config{
		Version:  "3.6",
		Filename: "banana",
		Services: []composetypes.ServiceConfig{
			{
				Name:  "foo",
				Image: "foo",
				Volumes: []composetypes.ServiceVolumeConfig{
					{
						Type:   "tmpfs",
						Target: "/app",
						Tmpfs: &composetypes.ServiceVolumeTmpfs{
							Size: 10000,
						},
					},
				},
			},
		},
	})
	assert.ErrorContains(t, err, "the compose yaml file is invalid with v3.5: services.foo.volumes.0 Additional property tmpfs is not allowed")
}
