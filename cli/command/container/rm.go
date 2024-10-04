package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type rmOptions struct {
	rmVolumes bool
	rmLink    bool
	force     bool
	all       bool // <-- Added this field for the --all option

	containers []string
}

// NewRmCommand creates a new cobra.Command for `docker rm`
func NewRmCommand(dockerCli command.Cli) *cobra.Command {
	var opts rmOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] CONTAINER [CONTAINER...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more containers",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runRm(cmd.Context(), dockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container rm, docker container remove, docker rm",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, true),
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.rmVolumes, "volumes", "v", false, "Remove anonymous volumes associated with the container")
	flags.BoolVarP(&opts.rmLink, "link", "l", false, "Remove the specified link")
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the removal of a running container (uses SIGKILL)")
	flags.BoolVarP(&opts.all, "all", "a", false, "Remove all containers (both running and stopped)") // <-- Added this line

	return cmd
}

func runRm(ctx context.Context, dockerCli command.Cli, opts *rmOptions) error {
	var errs []string
	var containersToRemove []string

	// If --all is specified, fetch all container IDs
	if opts.all {
		// Use Docker API to get all containers (use the appropriate options)
		containers, err := dockerCli.Client().ContainerList(ctx, container.ListOptions{All: true})
		if err != nil {
			return err
		}
		for _, ctr := range containers {
			containersToRemove = append(containersToRemove, ctr.ID) // Add container ID to the list
		}
	} else {
		containersToRemove = opts.containers // Use specified containers
	}

	// Use the same parallel operation for removing containers
	errChan := parallelOperation(ctx, containersToRemove, func(ctx context.Context, ctrID string) error {
		ctrID = strings.Trim(ctrID, "/")
		if ctrID == "" {
			return errors.New("Container name cannot be empty")
		}
		return dockerCli.Client().ContainerRemove(ctx, ctrID, container.RemoveOptions{
			RemoveVolumes: opts.rmVolumes,
			RemoveLinks:   opts.rmLink,
			Force:         opts.force,
		})
	})

	for _, name := range containersToRemove {
		if err := <-errChan; err != nil {
			if opts.force && errdefs.IsNotFound(err) {
				fmt.Fprintln(dockerCli.Err(), err)
				continue
			}
			errs = append(errs, err.Error())
			continue
		}
		fmt.Fprintln(dockerCli.Out(), name)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
