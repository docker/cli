package job

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/spf13/cobra"
)

// newPauseCommand wires the `docker job pause JOB` cobra command. client is
// injected explicitly, following the same pattern as newRunCommand (see
// run.go).
func newPauseCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause JOB",
		Short: "Suppress a job's schedule trigger",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPause(cmd.Context(), dockerCli, client, args[0])
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}
	return cmd
}

// runPause pauses jobRef via client.Pause. client.Pause has no reply
// message, so on success jobRef is printed on dockerCli.Out(), mirroring
// runRm (see rm.go) and `docker volume rm`/`docker network rm`.
func runPause(ctx context.Context, dockerCli command.Cli, client jobsClient, jobRef string) error {
	err := client.Pause(ctx, &jobsv0.PauseRequest{JobRef: jobRef})
	if err != nil {
		return wrapJobsError(err)
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), jobRef)
	return nil
}
