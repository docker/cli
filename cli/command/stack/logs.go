package stack

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// logsOptions holds docker stack logs options
type logsOptions struct {
	namespace  string
	follow     bool
	since      string
	tail       string
	timestamps bool
	noColor    bool
}

func newLogsCommand(dockerCLI command.Cli) *cobra.Command {
	var opts logsOptions

	cmd := &cobra.Command{
		Use:   "logs [OPTIONS] STACK",
		Short: "Fetch aggregated logs of all services in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.namespace = args[0]
			if err := validateStackName(opts.namespace); err != nil {
				return err
			}
			return runLogs(cmd.Context(), dockerCLI, opts)
		},
		Annotations:           map[string]string{"version": "1.29"},
		ValidArgsFunction:     completeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.follow, "follow", "f", false, "Follow log output")
	flags.StringVar(&opts.since, "since", "", `Show logs since timestamp (e.g. "2013-01-02T13:23:37Z") or relative (e.g. "42m" for 42 minutes)`)
	flags.StringVarP(&opts.tail, "tail", "n", "all", "Number of lines to show from the end of the logs (per service)")
	flags.BoolVarP(&opts.timestamps, "timestamps", "t", false, "Show timestamps")
	flags.BoolVar(&opts.noColor, "no-color", false, "Produce monochrome output")
	return cmd
}

// runLogs is the swarm implementation of docker stack logs. It streams the
// logs of every service in the stack concurrently, prefixing each line with
// the (colored) service name, similar to "docker compose logs".
func runLogs(ctx context.Context, dockerCLI command.Cli, opts logsOptions) error {
	apiClient := dockerCLI.Client()

	res, err := getStackServices(ctx, apiClient, opts.namespace)
	if err != nil {
		return err
	}
	if len(res.Items) == 0 {
		return fmt.Errorf("nothing found in stack: %s", opts.namespace)
	}

	maxLen := 0
	for _, s := range res.Items {
		if n := len(s.Spec.Name); n > maxLen {
			maxLen = n
		}
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex // serializes output lines across services
	)
	errs := make([]error, len(res.Items))
	for i, service := range res.Items {
		wg.Add(1)
		go func(idx int, s swarm.Service) {
			defer wg.Done()
			if err := streamServiceLogs(ctx, apiClient, dockerCLI, s, idx, maxLen, &mu, opts); err != nil {
				errs[idx] = fmt.Errorf("%s: %w", s.Spec.Name, err)
			}
		}(i, service)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func streamServiceLogs(ctx context.Context, apiClient client.APIClient, dockerCLI command.Cli, s swarm.Service, idx, maxLen int, mu *sync.Mutex, opts logsOptions) error {
	body, err := apiClient.ServiceLogs(ctx, s.ID, client.ServiceLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.follow,
		Since:      opts.since,
		Tail:       opts.tail,
		Timestamps: opts.timestamps,
	})
	if err != nil {
		return err
	}
	defer body.Close()

	prefix := logPrefix(s.Spec.Name, idx, maxLen, opts.noColor)
	stdout := newPrefixWriter(dockerCLI.Out(), prefix, mu)
	stderr := newPrefixWriter(dockerCLI.Err(), prefix, mu)
	// Service logs are always multiplexed (services with a TTY are not
	// supported by the service logs endpoint without --raw).
	_, err = stdcopy.StdCopy(stdout, stderr, body)
	return err
}
