package container

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/signal"
	"github.com/moby/term"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type startOptions struct {
	attach        bool
	openStdin     bool
	detachKeys    string
	checkpoint    string
	checkpointDir string

	containers []string
}

// NewStartCommand creates a new cobra.Command for `docker start`
func NewStartCommand(dockerCli command.Cli) *cobra.Command {
	var opts startOptions

	cmd := &cobra.Command{
		Use:   "start [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Start one or more stopped containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runStart(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.attach, "attach", "a", false, "Attach STDOUT/STDERR and forward signals")
	flags.BoolVarP(&opts.openStdin, "interactive", "i", false, "Attach container's STDIN")
	flags.StringVar(&opts.detachKeys, "detach-keys", "", "Override the key sequence for detaching a container")

	flags.StringVar(&opts.checkpoint, "checkpoint", "", "Restore from this checkpoint")
	flags.SetAnnotation("checkpoint", "experimental", nil)
	flags.SetAnnotation("checkpoint", "ostype", []string{"linux"})
	flags.StringVar(&opts.checkpointDir, "checkpoint-dir", "", "Use a custom checkpoint storage directory")
	flags.SetAnnotation("checkpoint-dir", "experimental", nil)
	flags.SetAnnotation("checkpoint-dir", "ostype", []string{"linux"})
	return cmd
}

// nolint: gocyclo
func runStart(dockerCli command.Cli, opts *startOptions) error {
	ctx, cancelFun := context.WithCancel(context.Background())
	defer cancelFun()

	if opts.attach || opts.openStdin {
		// We're going to attach to a container.
		// 1. Ensure we only have one container.
		if len(opts.containers) > 1 {
			return errors.New("you cannot start and attach multiple containers at once")
		}

		// 2. Attach to the container.
		container := opts.containers[0]
		c, err := dockerCli.Client().ContainerInspect(ctx, container)
		if err != nil {
			return err
		}

		// We always use c.ID instead of container to maintain consistency during `docker start`
		if !c.Config.Tty {
			sigc := ForwardAllSignals(ctx, dockerCli, c.ID)
			defer signal.StopCatch(sigc)
		}

		if opts.detachKeys != "" {
			dockerCli.ConfigFile().DetachKeys = opts.detachKeys
		}

		options := types.ContainerAttachOptions{
			Stream:     true,
			Stdin:      opts.openStdin && c.Config.OpenStdin,
			Stdout:     true,
			Stderr:     true,
			DetachKeys: dockerCli.ConfigFile().DetachKeys,
		}

		var in io.ReadCloser

		if options.Stdin {
			in = dockerCli.In()
		}

		resp, errAttach := dockerCli.Client().ContainerAttach(ctx, c.ID, options)
		if errAttach != nil {
			return errAttach
		}
		defer resp.Close()

		cErr := make(chan error, 1)

		go func() {
			cErr <- func() error {
				streamer := hijackedIOStreamer{
					streams:      dockerCli,
					inputStream:  in,
					outputStream: dockerCli.Out(),
					errorStream:  dockerCli.Err(),
					resp:         resp,
					tty:          c.Config.Tty,
					detachKeys:   options.DetachKeys,
				}

				errHijack := streamer.stream(ctx)
				if errHijack == nil {
					return errAttach
				}
				return errHijack
			}()
		}()

		// 3. We should open a channel for receiving status code of the container
		// no matter it's detached, removed on daemon side(--rm) or exit normally.
		statusChan := waitExitOrRemoved(ctx, dockerCli, c.ID, c.HostConfig.AutoRemove)
		startOptions := types.ContainerStartOptions{
			CheckpointID:  opts.checkpoint,
			CheckpointDir: opts.checkpointDir,
		}

		// 4. Start the container.
		if err := dockerCli.Client().ContainerStart(ctx, c.ID, startOptions); err != nil {
			cancelFun()
			<-cErr
			if c.HostConfig.AutoRemove {
				// wait container to be removed
				<-statusChan
			}
			return err
		}

		// 5. Wait for attachment to break.
		if c.Config.Tty && dockerCli.Out().IsTerminal() {
			if err := MonitorTtySize(ctx, dockerCli, c.ID, false); err != nil {
				fmt.Fprintln(dockerCli.Err(), "Error monitoring TTY size:", err)
			}
		}
		if attachErr := <-cErr; attachErr != nil {
			if _, ok := attachErr.(term.EscapeError); ok {
				// The user entered the detach escape sequence.
				return nil
			}
			return attachErr
		}

		if status := <-statusChan; status != 0 {
			return cli.StatusError{StatusCode: status}
		}
	} else if opts.checkpoint != "" {
		if len(opts.containers) > 1 {
			return errors.New("you cannot restore multiple containers at once")
		}
		container := opts.containers[0]
		startOptions := types.ContainerStartOptions{
			CheckpointID:  opts.checkpoint,
			CheckpointDir: opts.checkpointDir,
		}
		return dockerCli.Client().ContainerStart(ctx, container, startOptions)

	} else {
		// We're not going to attach to anything.
		// Start as many containers as we want.
		return startContainersWithoutAttachments(ctx, dockerCli, opts.containers)
	}

	return nil
}

func startContainersWithoutAttachments(ctx context.Context, dockerCli command.Cli, containers []string) error {
	var failedContainers []string
	for _, container := range containers {
		if err := dockerCli.Client().ContainerStart(ctx, container, types.ContainerStartOptions{}); err != nil {
			fmt.Fprintln(dockerCli.Err(), err)
			failedContainers = append(failedContainers, container)
			continue
		}
		fmt.Fprintln(dockerCli.Out(), container)
	}

	if len(failedContainers) > 0 {
		return errors.Errorf("Error: failed to start containers: %s", strings.Join(failedContainers, ", "))
	}
	return nil
}
