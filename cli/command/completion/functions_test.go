package completion

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/filters"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/env"
)

type fakeCLI struct {
	*fakeClient
}

// Client implements [APIClientProvider].
func (c fakeCLI) Client() client.APIClient {
	return c.fakeClient
}

type fakeClient struct {
	client.Client
	containerListFunc func(options client.ContainerListOptions) ([]container.Summary, error)
	imageListFunc     func(options client.ImageListOptions) ([]image.Summary, error)
	networkListFunc   func(ctx context.Context, options client.NetworkListOptions) ([]network.Summary, error)
	volumeListFunc    func(filter filters.Args) (volume.ListResponse, error)
}

func (c *fakeClient) ContainerList(_ context.Context, options client.ContainerListOptions) ([]container.Summary, error) {
	if c.containerListFunc != nil {
		return c.containerListFunc(options)
	}
	return []container.Summary{}, nil
}

func (c *fakeClient) ImageList(_ context.Context, options client.ImageListOptions) ([]image.Summary, error) {
	if c.imageListFunc != nil {
		return c.imageListFunc(options)
	}
	return []image.Summary{}, nil
}

func (c *fakeClient) NetworkList(ctx context.Context, options client.NetworkListOptions) ([]network.Summary, error) {
	if c.networkListFunc != nil {
		return c.networkListFunc(ctx, options)
	}
	return []network.Summary{}, nil
}

func (c *fakeClient) VolumeList(_ context.Context, options client.VolumeListOptions) (volume.ListResponse, error) {
	if c.volumeListFunc != nil {
		return c.volumeListFunc(options.Filters)
	}
	return volume.ListResponse{}, nil
}

func TestCompleteContainerNames(t *testing.T) {
	tests := []struct {
		doc              string
		showAll, showIDs bool
		filters          []func(container.Summary) bool
		containers       []container.Summary
		expOut           []string
		expOpts          client.ContainerListOptions
		expDirective     cobra.ShellCompDirective
	}{
		{
			doc:          "no results",
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:     "all containers",
			showAll: true,
			containers: []container.Summary{
				{ID: "id-c", State: container.StateRunning, Names: []string{"/container-c", "/container-c/link-b"}},
				{ID: "id-b", State: container.StateCreated, Names: []string{"/container-b"}},
				{ID: "id-a", State: container.StateExited, Names: []string{"/container-a"}},
			},
			expOut:       []string{"container-c", "container-c/link-b", "container-b", "container-a"},
			expOpts:      client.ContainerListOptions{All: true},
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:     "all containers with ids",
			showAll: true,
			showIDs: true,
			containers: []container.Summary{
				{ID: "id-c", State: container.StateRunning, Names: []string{"/container-c", "/container-c/link-b"}},
				{ID: "id-b", State: container.StateCreated, Names: []string{"/container-b"}},
				{ID: "id-a", State: container.StateExited, Names: []string{"/container-a"}},
			},
			expOut:       []string{"id-c", "container-c", "container-c/link-b", "id-b", "container-b", "id-a", "container-a"},
			expOpts:      client.ContainerListOptions{All: true},
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:     "only running containers",
			showAll: false,
			containers: []container.Summary{
				{ID: "id-c", State: container.StateRunning, Names: []string{"/container-c", "/container-c/link-b"}},
			},
			expOut:       []string{"container-c", "container-c/link-b"},
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:     "with filter",
			showAll: true,
			filters: []func(container.Summary) bool{
				func(ctr container.Summary) bool { return ctr.State == container.StateCreated },
			},
			containers: []container.Summary{
				{ID: "id-c", State: container.StateRunning, Names: []string{"/container-c", "/container-c/link-b"}},
				{ID: "id-b", State: container.StateCreated, Names: []string{"/container-b"}},
				{ID: "id-a", State: container.StateExited, Names: []string{"/container-a"}},
			},
			expOut:       []string{"container-b"},
			expOpts:      client.ContainerListOptions{All: true},
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:     "multiple filters",
			showAll: true,
			filters: []func(container.Summary) bool{
				func(ctr container.Summary) bool { return ctr.ID == "id-a" },
				func(ctr container.Summary) bool { return ctr.State == container.StateCreated },
			},
			containers: []container.Summary{
				{ID: "id-c", State: container.StateRunning, Names: []string{"/container-c", "/container-c/link-b"}},
				{ID: "id-b", State: container.StateCreated, Names: []string{"/container-b"}},
				{ID: "id-a", State: container.StateCreated, Names: []string{"/container-a"}},
			},
			expOut:       []string{"container-a"},
			expOpts:      client.ContainerListOptions{All: true},
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:          "with error",
			expDirective: cobra.ShellCompDirectiveError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			if tc.showIDs {
				t.Setenv("DOCKER_COMPLETION_SHOW_CONTAINER_IDS", "yes")
			}
			comp := ContainerNames(fakeCLI{&fakeClient{
				containerListFunc: func(opts client.ContainerListOptions) ([]container.Summary, error) {
					assert.Check(t, is.DeepEqual(opts, tc.expOpts, cmpopts.IgnoreUnexported(client.ContainerListOptions{}, filters.Args{})))
					if tc.expDirective == cobra.ShellCompDirectiveError {
						return nil, errors.New("some error occurred")
					}
					return tc.containers, nil
				},
			}}, tc.showAll, tc.filters...)

			containers, directives := comp(&cobra.Command{}, nil, "")
			assert.Check(t, is.Equal(directives&tc.expDirective, tc.expDirective))
			assert.Check(t, is.DeepEqual(containers, tc.expOut))
		})
	}
}

func TestCompleteEnvVarNames(t *testing.T) {
	env.PatchAll(t, map[string]string{
		"ENV_A": "hello-a",
		"ENV_B": "hello-b",
	})
	values, directives := EnvVarNames()(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")

	sort.Strings(values)
	expected := []string{"ENV_A", "ENV_B"}
	assert.Check(t, is.DeepEqual(values, expected))
}

func TestCompleteFileNames(t *testing.T) {
	values, directives := FileNames()(nil, nil, "")
	assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveDefault))
	assert.Check(t, is.Len(values, 0))
}

func TestCompleteFromList(t *testing.T) {
	expected := []string{"one", "two", "three"}

	values, directives := FromList(expected...)(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Check(t, is.DeepEqual(values, expected))
}

func TestCompleteImageNames(t *testing.T) {
	tests := []struct {
		doc          string
		images       []image.Summary
		expOut       []string
		expDirective cobra.ShellCompDirective
	}{
		{
			doc:          "no results",
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc: "with results",
			images: []image.Summary{
				{RepoTags: []string{"image-c:latest", "image-c:other"}},
				{RepoTags: []string{"image-b:latest", "image-b:other"}},
				{RepoTags: []string{"image-a:latest", "image-a:other"}},
			},
			expOut:       []string{"image-c:latest", "image-c:other", "image-b:latest", "image-b:other", "image-a:latest", "image-a:other"},
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:          "with error",
			expDirective: cobra.ShellCompDirectiveError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			comp := ImageNames(fakeCLI{&fakeClient{
				imageListFunc: func(options client.ImageListOptions) ([]image.Summary, error) {
					if tc.expDirective == cobra.ShellCompDirectiveError {
						return nil, errors.New("some error occurred")
					}
					return tc.images, nil
				},
			}}, -1)

			volumes, directives := comp(&cobra.Command{}, nil, "")
			assert.Check(t, is.Equal(directives&tc.expDirective, tc.expDirective))
			assert.Check(t, is.DeepEqual(volumes, tc.expOut))
		})
	}
}

func TestCompleteNetworkNames(t *testing.T) {
	tests := []struct {
		doc          string
		networks     []network.Summary
		expOut       []string
		expDirective cobra.ShellCompDirective
	}{
		{
			doc:          "no results",
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc: "with results",
			networks: []network.Summary{
				{
					Network: network.Network{
						ID:   "nw-c",
						Name: "network-c",
					},
				},
				{
					Network: network.Network{
						ID:   "nw-b",
						Name: "network-b",
					},
				},
				{
					Network: network.Network{
						ID:   "nw-a",
						Name: "network-a",
					},
				},
			},
			expOut:       []string{"network-c", "network-b", "network-a"},
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:          "with error",
			expDirective: cobra.ShellCompDirectiveError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			comp := NetworkNames(fakeCLI{&fakeClient{
				networkListFunc: func(ctx context.Context, options client.NetworkListOptions) ([]network.Summary, error) {
					if tc.expDirective == cobra.ShellCompDirectiveError {
						return nil, errors.New("some error occurred")
					}
					return tc.networks, nil
				},
			}})

			volumes, directives := comp(&cobra.Command{}, nil, "")
			assert.Check(t, is.Equal(directives&tc.expDirective, tc.expDirective))
			assert.Check(t, is.DeepEqual(volumes, tc.expOut))
		})
	}
}

func TestCompletePlatforms(t *testing.T) {
	values, directives := Platforms()(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Check(t, is.DeepEqual(values, commonPlatforms))
}

func TestCompleteVolumeNames(t *testing.T) {
	tests := []struct {
		doc          string
		volumes      []*volume.Volume
		expOut       []string
		expDirective cobra.ShellCompDirective
	}{
		{
			doc:          "no results",
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc: "with results",
			volumes: []*volume.Volume{
				{Name: "volume-c"},
				{Name: "volume-b"},
				{Name: "volume-a"},
			},
			expOut:       []string{"volume-c", "volume-b", "volume-a"},
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:          "with error",
			expDirective: cobra.ShellCompDirectiveError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			comp := VolumeNames(fakeCLI{&fakeClient{
				volumeListFunc: func(filter filters.Args) (volume.ListResponse, error) {
					if tc.expDirective == cobra.ShellCompDirectiveError {
						return volume.ListResponse{}, errors.New("some error occurred")
					}
					return volume.ListResponse{Volumes: tc.volumes}, nil
				},
			}})

			volumes, directives := comp(&cobra.Command{}, nil, "")
			assert.Check(t, is.Equal(directives&tc.expDirective, tc.expDirective))
			assert.Check(t, is.DeepEqual(volumes, tc.expOut))
		})
	}
}
