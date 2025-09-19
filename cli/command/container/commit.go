package container

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type commitOptions struct {
	container string
	reference string

	pause   bool
	noPause bool
	comment string
	author  string
	changes opts.ListOpts
}

// newCommitCommand creates a new cobra.Command for `docker commit`
func newCommitCommand(dockerCLI command.Cli) *cobra.Command {
	var options commitOptions

	cmd := &cobra.Command{
		Use:   "commit [OPTIONS] CONTAINER [REPOSITORY[:TAG]]",
		Short: "Create a new image from a container's changes",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.container = args[0]
			if len(args) > 1 {
				options.reference = args[1]
			}
			if cmd.Flag("pause").Changed {
				if cmd.Flag("no-pause").Changed {
					return errors.New("conflicting options: --no-pause and --pause cannot be used together")
				}
				options.noPause = !options.pause
			}
			return runCommit(cmd.Context(), dockerCLI, &options)
		},
		Annotations: map[string]string{
			"aliases": "docker container commit, docker commit",
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, false),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	// TODO(thaJeztah): Deprecated: the --pause flag was deprecated in v29 and can be removed in v30.
	flags.BoolVarP(&options.pause, "pause", "p", true, "Pause container during commit (deprecated: use --no-pause instead)")
	_ = flags.MarkDeprecated("pause", "and enabled by default. Use --no-pause to disable pausing during commit.")

	flags.BoolVar(&options.noPause, "no-pause", false, "Disable pausing container during commit")
	flags.StringVarP(&options.comment, "message", "m", "", "Commit message")
	flags.StringVarP(&options.author, "author", "a", "", `Author (e.g., "John Hannibal Smith <hannibal@a-team.com>")`)

	options.changes = opts.NewListOpts(nil)
	flags.VarP(&options.changes, "change", "c", "Apply Dockerfile instruction to the created image")

	return cmd
}

func runCommit(ctx context.Context, dockerCli command.Cli, options *commitOptions) error {
	response, err := dockerCli.Client().ContainerCommit(ctx, options.container, client.ContainerCommitOptions{
		Reference: options.reference,
		Comment:   options.comment,
		Author:    options.author,
		Changes:   options.changes.GetSlice(),
		NoPause:   options.noPause,
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), response.ID)
	return nil
}
