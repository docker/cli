package container

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type rmOptions struct {
	rmVolumes bool
	rmLink    bool
	force     bool

	containers []string
}

// newRmCommand creates a new cobra.Command for "docker container rm".
func newRmCommand(dockerCLI command.Cli) *cobra.Command {
	var opts rmOptions

	cmd := &cobra.Command{
		Use:   "rm [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Remove one or more containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runRm(cmd.Context(), dockerCLI, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container rm, docker container remove, docker rm",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCLI, true, func(ctr container.Summary) bool {
			return opts.force || ctr.State == container.StateExited || ctr.State == container.StateCreated
		}),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.rmVolumes, "volumes", "v", false, "Remove anonymous volumes associated with the container")
	flags.BoolVarP(&opts.rmLink, "link", "l", false, "Remove the specified link")
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the removal of a running container (uses SIGKILL)")
	return cmd
}

// newRemoveCommand adds subcommands for "docker container"; unlike the
// top-level "docker rm", it also adds a "remove" alias to support
// "docker container remove" in addition to "docker container rm".
func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	cmd := *newRmCommand(dockerCli)
	cmd.Aliases = []string{"rm", "remove"}
	return &cmd
}

func runRm(ctx context.Context, dockerCLI command.Cli, opts *rmOptions) error {
	apiClient := dockerCLI.Client()
	errChan := parallelOperation(ctx, opts.containers, func(ctx context.Context, ctrID string) error {
		ctrID = strings.Trim(ctrID, "/")
		if ctrID == "" {
			return errors.New("container name cannot be empty")
		}
		_, err := apiClient.ContainerRemove(ctx, ctrID, client.ContainerRemoveOptions{
			RemoveVolumes: opts.rmVolumes,
			RemoveLinks:   opts.rmLink,
			Force:         opts.force,
		})
		return err
	})

	var errs []error
	for _, name := range opts.containers {
		if err := <-errChan; err != nil {
			if opts.force && errdefs.IsNotFound(err) {
				_, _ = fmt.Fprintln(dockerCLI.Err(), err)
				continue
			}
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}
	return errors.Join(errs...)
}
