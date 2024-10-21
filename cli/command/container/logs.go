package container

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
)

type logsOptions struct {
	follow     bool
	since      string
	until      string
	timestamps bool
	details    bool
	tail       string

	detachDelay int

	container string
}

// NewLogsCommand creates a new cobra.Command for `docker logs`
func NewLogsCommand(dockerCli command.Cli) *cobra.Command {
	var opts logsOptions

	cmd := &cobra.Command{
		Use:   "logs [OPTIONS] CONTAINER",
		Short: "Fetch the logs of a container",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.container = args[0]
			return runLogs(cmd.Context(), dockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container logs, docker logs",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, true),
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.follow, "follow", "f", false, "Follow log output")
	flags.StringVar(&opts.since, "since", "", `Show logs since timestamp (e.g. "2013-01-02T13:23:37Z") or relative (e.g. "42m" for 42 minutes)`)
	flags.StringVar(&opts.until, "until", "", `Show logs before a timestamp (e.g. "2013-01-02T13:23:37Z") or relative (e.g. "42m" for 42 minutes)`)
	flags.SetAnnotation("until", "version", []string{"1.35"})
	flags.BoolVarP(&opts.timestamps, "timestamps", "t", false, "Show timestamps")
	flags.BoolVar(&opts.details, "details", false, "Show extra details provided to logs")
	flags.StringVarP(&opts.tail, "tail", "n", "all", "Number of lines to show from the end of the logs")
	flags.IntVarP(&opts.detachDelay, "delay", "d", 0, "Number of seconds to wait for container restart before exiting")
	return cmd
}

func runLogs(ctx context.Context, dockerCli command.Cli, opts *logsOptions) error {
	c, err := dockerCli.Client().ContainerInspect(ctx, opts.container)
	if err != nil {
		return err
	}

	since := opts.since
	for {
		restarting := make(chan events.Message)
		if opts.detachDelay != 0 {
			go func() {
				eventsCtx, eventsCtxCancel := context.WithCancel(ctx)
				eventC, eventErrs := dockerCli.Client().Events(eventsCtx, events.ListOptions{})
				defer eventsCtxCancel()
				for {
					select {
					case event := <-eventC:
						if event.Action == events.ActionRestart && event.Actor.ID == c.ID {
							restarting <- event
							return
						}
					case <-eventErrs:
						return
					}
				}
			}()
		}

		opts.since = since
		err = streamLogs(ctx, dockerCli, opts, c)

		if opts.detachDelay == 0 {
			return err
		}

		select {
		case restartEvent := <-restarting:
			since = strconv.FormatInt(restartEvent.Time, 10)
		case <-time.After(time.Duration(opts.detachDelay) * time.Second):
			return err
		}
	}
}

func streamLogs(ctx context.Context, dockerCli command.Cli, opts *logsOptions, c container.InspectResponse) error {
	responseBody, err := dockerCli.Client().ContainerLogs(ctx, c.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      opts.since,
		Until:      opts.until,
		Timestamps: opts.timestamps,
		Follow:     opts.follow,
		Tail:       opts.tail,
		Details:    opts.details,
	})
	if err != nil {
		return err
	}
	defer responseBody.Close()

	if c.Config.Tty {
		_, err = io.Copy(dockerCli.Out(), responseBody)
	} else {
		_, err = stdcopy.StdCopy(dockerCli.Out(), dockerCli.Err(), responseBody)
	}

	return err
}
