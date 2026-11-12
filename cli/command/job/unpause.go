package job

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/spf13/cobra"
)

// newUnpauseCommand wires the `docker job unpause JOB` cobra command. client is
// injected explicitly, following the same pattern as newRunCommand (see
// run.go).
func newUnpauseCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpause JOB",
		Short: "Re-arm a paused job's schedule trigger",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnpause(cmd.Context(), dockerCli, client, args[0])
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}
	return cmd
}

// runUnpause unpauses jobRef via client.Resume. client.Resume has no reply
// message, so on success jobRef is printed on dockerCli.Out(), mirroring
// runRm (see rm.go) and `docker volume rm`/`docker network rm`.
func runUnpause(ctx context.Context, dockerCli command.Cli, client jobsClient, jobRef string) error {
	err := client.Resume(ctx, &jobsv0.ResumeRequest{JobRef: jobRef})
	if err != nil {
		return wrapJobsError(err)
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), jobRef)
	return nil
}
