package system

import (
	"context"
	"errors"
	"testing"

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
				*builders.Container("foo"),
				*builders.Container("bar"),
			}, nil
		},
	})

	completions, directive := completeFilters(cli)(NewEventsCommand(cli), nil, "container=")

	assert.DeepEqual(t, completions, []string{"container=foo", "container=bar"})
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
