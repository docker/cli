package container

import (
	"context"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
)

func TestNewKillCommand(t *testing.T) {
	containers := []types.Container{
		{ID: "2d2a4efadc5b101f146be1ed32b6d7e01ddc9e66942332fed2a956a46b695ef9"},
		{ID: "84661ee093acd95b3fba80c3ff49b747794e404b11a5873f7c26f24f15862c82"},
		{ID: "1ac36525ba75d9c8110257270021fbd3ac70a02d0ff3d990372a7b672ce313d8"},
	}

	client := &fakeClient{
		containerKillFunc: func(ctx context.Context, killContainer, signal string) error {
			for _, container := range containers {
				if killContainer == container.ID {
					containers = containers[1:]
					return nil
				}
			}
			return errors.Errorf("response from daemon: Cannot kill container: %s", killContainer)
		},
		containerListFunc: func(clo types.ContainerListOptions) ([]types.Container, error) {
			return containers, nil
		},
	}

	cli := test.NewFakeCli(client)
	cmd := NewKillCommand(cli)
	cmd.SetOut(io.Discard)

	tests := []struct {
		name               string
		args               []string
		assertErr          bool
		err                string
		expectedContainers []types.Container
	}{
		{
			name:      "kill one - nonexistent",
			args:      []string{"ubuntu:latest"},
			assertErr: true,
			err:       "Cannot kill container",
			expectedContainers: []types.Container{
				{ID: "2d2a4efadc5b101f146be1ed32b6d7e01ddc9e66942332fed2a956a46b695ef9"},
				{ID: "84661ee093acd95b3fba80c3ff49b747794e404b11a5873f7c26f24f15862c82"},
				{ID: "1ac36525ba75d9c8110257270021fbd3ac70a02d0ff3d990372a7b672ce313d8"},
			},
		},
		{
			name: "kill one - existent",
			args: []string{"2d2a4efadc5b101f146be1ed32b6d7e01ddc9e66942332fed2a956a46b695ef9"},
			expectedContainers: []types.Container{
				{ID: "84661ee093acd95b3fba80c3ff49b747794e404b11a5873f7c26f24f15862c82"},
				{ID: "1ac36525ba75d9c8110257270021fbd3ac70a02d0ff3d990372a7b672ce313d8"},
			},
		},
		{
			name: "kill one - existent",
			args: []string{"84661ee093acd95b3fba80c3ff49b747794e404b11a5873f7c26f24f15862c82"},
			expectedContainers: []types.Container{
				{ID: "1ac36525ba75d9c8110257270021fbd3ac70a02d0ff3d990372a7b672ce313d8"},
			},
		},
		{
			name:               "kill all - existent",
			args:               []string{"--all"},
			expectedContainers: []types.Container{},
		},
		{
			name:               "kill all - nonexistent",
			args:               []string{"--all", "-s", "SIGINT"},
			assertErr:          true,
			err:                "no containers running to send SIGINT signal to",
			expectedContainers: []types.Container{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if tt.assertErr {
				assert.ErrorContains(t, err, tt.err)
			} else {
				assert.NilError(t, err)
			}
			assert.DeepEqual(t, containers, tt.expectedContainers)
		})
	}
}
