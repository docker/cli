package container

import (
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/assert"
)

func TestRunLabel(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(_ *container.Config, _ *container.HostConfig, _ *network.NetworkingConfig, _ string) (container.ContainerCreateCreatedBody, error) {
			return container.ContainerCreateCreatedBody{
				ID: "id",
			}, nil
		},
		Version: "1.36",
	})
	cmd := NewRunCommand(cli)
	cmd.Flags().Set("detach", "true")
	cmd.SetArgs([]string{"--label", "foo", "busybox"})
	assert.NoError(t, cmd.Execute())
}
