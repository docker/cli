package system

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types/system"

	"github.com/docker/docker/api/types/network"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

// Successful completion lists all container names, prefixed with "container=".
// Filtering the completions by the current word is delegated to the completion script.
func TestCompleteEventFilterContainer(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
			return []container.Summary{
				*builders.Container("c1"),
				*builders.Container("c2"),
			}, nil
		},
	})

	completions, directive := completeFilters(cli)(NewEventsCommand(cli), nil, "container=")

	assert.DeepEqual(t, completions, []string{"container=c1", "container=c2"})
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)
}

// In case of API errors, no completions are returned.
func TestCompleteEventFilterContainerAPIError(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
			return nil, errors.New("API error")
		},
	})

	completions, directive := completeFilters(cli)(NewEventsCommand(cli), nil, "container=")

	assert.DeepEqual(t, completions, []string{})
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)
}

// Successful completion lists the name and id of the current Docker daemon, prefixed with "daemon=".
// Filtering the completions by the current word is delegated to the completion script.
func TestCompleteEventFilterDaemon(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		infoFunc: func(ctx context.Context) (system.Info, error) {
			return system.Info{
				ID:   "daemon-id",
				Name: "daemon-name",
			}, nil
		},
	})

	completions, directive := completeFilters(cli)(NewEventsCommand(cli), nil, "daemon=")

	assert.DeepEqual(t, completions, []string{"daemon=daemon-name", "daemon=daemon-id"})
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)
}

// In case of API errors, no completions are returned.
func TestCompleteEventFilterDaemonAPIError(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		infoFunc: func(ctx context.Context) (system.Info, error) {
			return system.Info{}, errors.New("API error")
		},
	})

	completions, directive := completeFilters(cli)(NewEventsCommand(cli), nil, "daemon=")

	assert.DeepEqual(t, completions, []string{})
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)
}

// Successful completion lists all network names, prefixed with "network=".
// Filtering the completions by the current word is delegated to the completion script.
func TestCompleteEventFilterNetwork(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		networkListFunc: func(_ context.Context, _ network.ListOptions) ([]network.Summary, error) {
			return []network.Summary{
				*builders.NetworkResource(builders.NetworkResourceName("nw1")),
				*builders.NetworkResource(builders.NetworkResourceName("nw2")),
			}, nil
		},
	})

	completions, directive := completeFilters(cli)(NewEventsCommand(cli), nil, "network=")

	assert.DeepEqual(t, completions, []string{"network=nw1", "network=nw2"})
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)
}

// In case of API errors, no completions are returned.
func TestCompleteEventFilterNetworkAPIError(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		networkListFunc: func(_ context.Context, _ network.ListOptions) ([]network.Summary, error) {
			return nil, errors.New("API error")
		},
	})

	completions, directive := completeFilters(cli)(NewEventsCommand(cli), nil, "network=")

	assert.DeepEqual(t, completions, []string{})
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)
}
