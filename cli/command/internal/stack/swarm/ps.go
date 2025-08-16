package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/stack/options"
	"github.com/docker/cli/cli/command/task"
	"github.com/moby/moby/api/types/swarm"
)

// RunPS is the swarm implementation of docker stack ps
//
// Deprecated: This function will be removed from the Docker CLI's
// public facing API. External code should avoid relying on it.
func RunPS(ctx context.Context, dockerCLI cli.Cli, opts options.PS) error {
	return runPS(ctx, dockerCLI, opts)
}

func runPS(ctx context.Context, dockerCLI cli.Cli, opts options.PS) error {
	filter := getStackFilterFromOpt(opts.Namespace, opts.Filter)

	client := dockerCLI.Client()
	tasks, err := client.TaskList(ctx, swarm.TaskListOptions{Filters: filter})
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

	return task.Print(ctx, dockerCLI, tasks, idresolver.New(client, opts.NoResolve), !opts.NoTrunc, opts.Quiet, format)
}
