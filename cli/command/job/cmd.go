package job

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/commands"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newJobCommand)
}

// newJobCommand returns a cobra command for `job` subcommands.
func newJobCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job COMMAND",
		Short: "Manage jobs",
		Long: "Manage jobs, one-shot or scheduled containers run by the daemon.\n\n" +
			"This requires the daemon's jobs extension, which is disabled by default: start\n" +
			"dockerd with " + enableJobsFlag + ".",
		Args:                  cli.NoArgs,
		RunE:                  command.ShowHelp(dockerCli.Err()),
		Annotations:           map[string]string{"version": "1.57"},
		DisableFlagsInUseLine: true,
	}
	jobs := newLazyJobsClient(dockerCli)
	cmd.PersistentPostRunE = func(*cobra.Command, []string) error {
		return jobs.Close()
	}
	cmd.AddCommand(
		newListCommand(dockerCli, jobs),
		newLogsCommand(dockerCli, jobs),
	)
	return cmd
}

// lazyJobsClient defers newJobsClient(dockerCli) -- and therefore
// dockerCli.Client() -- until a jobsClient method is actually invoked from a
// subcommand's RunE. `docker job` subcommands are registered while the
// cobra command tree is built, which happens before dockerCli.Initialize();
// dialing eagerly at that point would call dockerCli.Client() too early and
// abort the whole process (Client() calls os.Exit(1) on failure), breaking
// every command, not just `job`.
//
// The dial is memoized: a single command invocation may call more than one
// jobsClient method (e.g. `docker job logs` falling back from Inspect to
// ListRuns), and each should reuse the same connection rather than dialing
// again. newJobCommand closes it after the invoked subcommand's RunE
// succeeds (cobra skips PersistentPostRunE on error, and the process exits
// right after).
type lazyJobsClient struct {
	dockerCli command.Cli

	client jobsClient
	closer func() error
}

func newLazyJobsClient(dockerCli command.Cli) *lazyJobsClient {
	return &lazyJobsClient{dockerCli: dockerCli}
}

func (c *lazyJobsClient) resolve() (jobsClient, error) {
	if c.client == nil {
		client, closer, err := newJobsClient(c.dockerCli)
		if err != nil {
			return nil, err
		}
		c.client, c.closer = client, closer
	}
	return c.client, nil
}

func (c *lazyJobsClient) Close() error {
	if c.closer == nil {
		return nil
	}
	return c.closer()
}

func (c *lazyJobsClient) List(ctx context.Context, req *jobsv0.ListRequest) (*jobsv0.ListReply, error) {
	client, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return client.List(ctx, req)
}

func (c *lazyJobsClient) Inspect(ctx context.Context, req *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
	client, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return client.Inspect(ctx, req)
}

func (c *lazyJobsClient) ListRuns(ctx context.Context, req *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
	client, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return client.ListRuns(ctx, req)
}
