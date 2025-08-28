package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/task"
	"github.com/docker/docker/api/types/swarm"
)

// RunPS is the swarm implementation of docker stack ps
//
// Deprecated: this function was for internal use and will be removed in the next release.
func RunPS(ctx context.Context, dockerCLI command.Cli, opts options.PS) error {
	filter := getStackFilterFromOpt(opts.Namespace, opts.Filter)

	apiClient := dockerCLI.Client()
	tasks, err := apiClient.TaskList(ctx, swarm.TaskListOptions{Filters: filter})
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		return fmt.Errorf("nothing found in stack: %s", opts.Namespace)
	}

	format := opts.Format
	if len(format) == 0 {
		format = task.DefaultFormat(dockerCLI.ConfigFile(), opts.Quiet)
	}

	return task.Print(ctx, dockerCLI, tasks, idresolver.New(apiClient, opts.NoResolve), !opts.NoTrunc, opts.Quiet, format)
}
