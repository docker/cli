package container

import (
	"context"
	"io"
	"os"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
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
	clean      bool

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
			return runLogs(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.follow, "follow", "f", false, "Follow log output")
	flags.StringVar(&opts.since, "since", "", "Show logs since timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	flags.StringVar(&opts.until, "until", "", "Show logs before a timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	flags.SetAnnotation("until", "version", []string{"1.35"})
	flags.BoolVarP(&opts.timestamps, "timestamps", "t", false, "Show timestamps")
	flags.BoolVar(&opts.details, "details", false, "Show extra details provided to logs")
	flags.StringVarP(&opts.tail, "tail", "n", "all", "Number of lines to show from the end of the logs")
	flags.BoolVar(&opts.clean, "clean", false, "limpa os logs")
	return cmd
}

func runLogs(dockerCli command.Cli, opts *logsOptions) error {
	ctx := context.Background()

	c, err := dockerCli.Client().ContainerInspect(ctx, opts.container)
	if err != nil {
		return err
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      opts.since,
		Until:      opts.until,
		Timestamps: opts.timestamps,
		Follow:     opts.follow,
		Tail:       opts.tail,
		Details:    opts.details,
		Clean:      opts.clean,
	}
	responseBody, err := dockerCli.Client().ContainerLogs(ctx, c.ID, options)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	if c.Config.Tty {
		_, err = io.Copy(dockerCli.Out(), responseBody)
	} else {
		_, err = stdcopy.StdCopy(dockerCli.Out(), dockerCli.Err(), responseBody)
	}
	
	if opts.clean {
		logSizeBefore, err := os.Stat(c.LogPath)
		if err != nil {
			return err
		}
		
		if logSizeBefore.Size() == 0 {
			fmt.Println("The log is empty")
			return err
		}

		fmt.Println("Cleaning logs...")

		err = os.Truncate(c.LogPath, 0)
		if err != nil {
        	return err
    	}
		logSizeAfter, err := os.Stat(c.LogPath)
		if err != nil {
			return err
		}
		if logSizeBefore.Size() > logSizeAfter.Size() {
			fmt.Println("...the log was cleared")
		}
		return err
	}
	return err
}
