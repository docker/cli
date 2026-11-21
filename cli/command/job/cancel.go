package job

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/spf13/cobra"
)

// newCancelCommand wires the `docker job cancel JOB` cobra command. client is
// injected explicitly, following the same pattern as newRunCommand (see
// run.go).
func newCancelCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel JOB",
		Short: "Cancel a job's in-flight run",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCancel(cmd.Context(), dockerCli, client, args[0])
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}
	return cmd
}

// runCancel cancels jobRef's in-flight run via client.Cancel. If a run was
// cancelled, its ID is printed on dockerCli.Out(), the same convention as
// runRun (see run.go). If no run was in flight, a message is printed instead
// and nil is returned: cancelling an idle job is a valid, idempotent
// request, not a failure.
func runCancel(ctx context.Context, dockerCli command.Cli, client jobsClient, jobRef string) error {
	reply, err := client.Cancel(ctx, &jobsv0.CancelRequest{JobRef: jobRef})
	if err != nil {
		return wrapJobsError(err)
	}

	if reply.RunID == "" {
		_, _ = fmt.Fprintf(dockerCli.Out(), "no run in flight for job %q\n", jobRef)
		return nil
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), reply.RunID)
	return nil
}
