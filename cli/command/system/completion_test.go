package system

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TestCompleteEventFilter(t *testing.T) {
	tests := []struct {
		client     *fakeClient
		toComplete string
		expected   []string
	}{
		{
			client: &fakeClient{
				containerListFunc: func(_ context.Context, _ container.ListOptions) ([]types.Container, error) {
					return []types.Container{
						*builders.Container("c1"),
						*builders.Container("c2"),
					}, nil
				},
			},
			toComplete: "container=",
			expected:   []string{"container=c1", "container=c2"},
		},
		{
			client: &fakeClient{
				containerListFunc: func(_ context.Context, _ container.ListOptions) ([]types.Container, error) {
					return nil, errors.New("API error")
				},
			},
			toComplete: "container=",
			expected:   []string{},
		},
		{
			client: &fakeClient{
				infoFunc: func(ctx context.Context) (system.Info, error) {
					return system.Info{
						ID:   "daemon-id",
						Name: "daemon-name",
					}, nil
				},
			},
			toComplete: "daemon=",
			expected:   []string{"daemon=daemon-name", "daemon=daemon-id"},
		},
		{
			client: &fakeClient{
				infoFunc: func(ctx context.Context) (system.Info, error) {
					return system.Info{}, errors.New("API error")
				},
			},
			toComplete: "daemon=",
			expected:   []string{},
		},
		{
			client: &fakeClient{
				imageListFunc: func(_ context.Context, _ image.ListOptions) ([]image.Summary, error) {
					return []image.Summary{
						{RepoTags: []string{"img:1"}},
						{RepoTags: []string{"img:2"}},
					}, nil
				},
			},
			toComplete: "image=",
			expected:   []string{"image=img:1", "image=img:2"},
		},
		{
			client: &fakeClient{
				imageListFunc: func(_ context.Context, _ image.ListOptions) ([]image.Summary, error) {
					return []image.Summary{}, errors.New("API error")
				},
			},
			toComplete: "image=",
			expected:   []string{},
		},
		{
			client: &fakeClient{
				networkListFunc: func(_ context.Context, _ network.ListOptions) ([]network.Summary, error) {
					return []network.Summary{
						*builders.NetworkResource(builders.NetworkResourceName("nw1")),
						*builders.NetworkResource(builders.NetworkResourceName("nw2")),
					}, nil
				},
			},
			toComplete: "network=",
			expected:   []string{"network=nw1", "network=nw2"},
		},
		{
			client: &fakeClient{
				networkListFunc: func(_ context.Context, _ network.ListOptions) ([]network.Summary, error) {
					return nil, errors.New("API error")
				},
			},
			toComplete: "network=",
			expected:   []string{},
		},
		{
			client: &fakeClient{
				nodeListFunc: func(_ context.Context, _ types.NodeListOptions) ([]swarm.Node, error) {
					return []swarm.Node{
						*builders.Node(builders.Hostname("n1")),
					}, nil
				},
			},
			toComplete: "node=",
			expected:   []string{"node=n1"},
		},
		{
			client: &fakeClient{
				nodeListFunc: func(_ context.Context, _ types.NodeListOptions) ([]swarm.Node, error) {
					return []swarm.Node{}, errors.New("API error")
				},
			},
			toComplete: "node=",
			expected:   []string{},
		},
		{
			client: &fakeClient{
				volumeListFunc: func(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
					return volume.ListResponse{
						Volumes: []*volume.Volume{
							builders.Volume(builders.VolumeName("v1")),
							builders.Volume(builders.VolumeName("v2")),
						},
					}, nil
				},
			},
			toComplete: "volume=",
			expected:   []string{"volume=v1", "volume=v2"},
		},
		{
			client: &fakeClient{
				volumeListFunc: func(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
					return volume.ListResponse{}, errors.New("API error")
				},
			},
			toComplete: "volume=",
			expected:   []string{},
		},
	}

	for _, tc := range tests {
		cli := test.NewFakeCli(tc.client)

		completions, directive := completeEventFilters(cli)(NewEventsCommand(cli), nil, tc.toComplete)

		assert.DeepEqual(t, completions, tc.expected)
		assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp, fmt.Sprintf("wrong directive in completion for '%s'", tc.toComplete))
	}
}
