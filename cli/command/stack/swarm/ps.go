package swarm

import (
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/task"
	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
)

// RunPS is the swarm implementation of docker stack ps
func RunPS(dockerCli command.Cli, opts options.PS) error {
	namespace := opts.Namespace
	client := dockerCli.Client()
	ctx := context.Background()

	filter := getStackFilterFromOpt(opts.Namespace, opts.Filter)

	tasks, err := client.TaskList(ctx, types.TaskListOptions{Filters: filter})
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		return fmt.Errorf("nothing found in stack: %s", namespace)
	}

	format := opts.Format
	if len(format) == 0 {
		format = task.DefaultFormat(dockerCli.ConfigFile(), opts.Quiet)
	}

	return task.Print(ctx, dockerCli, tasks, idresolver.New(client, opts.NoResolve), !opts.NoTrunc, opts.Quiet, format)
}
