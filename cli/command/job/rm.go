package job

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/spf13/cobra"
)

// rmOptions holds the parsed flags of `docker job rm`.
type rmOptions struct {
	job  string
	runs string
}

// newRmCommand wires the `docker job rm [--runs ...] JOB` cobra command.
// client is injected explicitly, following the same pattern as
// newRunCommand (see run.go).
func newRmCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	var opts rmOptions

	cmd := &cobra.Command{
		Use:     "rm JOB",
		Aliases: []string{"remove"},
		Short:   "Remove a job",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.job = args[0]
			return runRm(cmd.Context(), dockerCli, client, &opts)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.runs, "runs", "", `Run history retention: "keep", "remove", or "remove-finished" (default "keep")`)
	return cmd
}

// runRm removes opts.job via client.Remove. client.Remove has no reply
// message, so on success opts.job (the identifier passed by the caller) is
// printed on dockerCli.Out(), mirroring `docker volume rm`/`docker network
// rm`.
func runRm(ctx context.Context, dockerCli command.Cli, client jobsClient, opts *rmOptions) error {
	err := client.Remove(ctx, &jobsv0.RemoveRequest{JobRef: opts.job, RunsRemoval: opts.runs})
	if err != nil {
		return wrapJobsError(err)
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), opts.job)
	return nil
}
