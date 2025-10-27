package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/moby/sys/signal"
	"github.com/moby/term"
	"github.com/spf13/cobra"
)

// StartOptions group options for `start` command
type StartOptions struct {
	Attach        bool
	OpenStdin     bool
	DetachKeys    string
	Checkpoint    string
	CheckpointDir string

	Containers []string
}

// newStartCommand creates a new cobra.Command for "docker container start".
func newStartCommand(dockerCLI command.Cli) *cobra.Command {
	var opts StartOptions

	cmd := &cobra.Command{
		Use:   "start [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Start one or more stopped containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Containers = args
			return RunStart(cmd.Context(), dockerCLI, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container start, docker start",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCLI, true, func(ctr container.Summary) bool {
			return ctr.State == container.StateExited || ctr.State == container.StateCreated
		}),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.Attach, "attach", "a", false, "Attach STDOUT/STDERR and forward signals")
	flags.BoolVarP(&opts.OpenStdin, "interactive", "i", false, "Attach container's STDIN")
	flags.StringVar(&opts.DetachKeys, "detach-keys", "", "Override the key sequence for detaching a container")

	flags.StringVar(&opts.Checkpoint, "checkpoint", "", "Restore from this checkpoint")
	flags.SetAnnotation("checkpoint", "experimental", nil)
	flags.SetAnnotation("checkpoint", "ostype", []string{"linux"})
	flags.StringVar(&opts.CheckpointDir, "checkpoint-dir", "", "Use a custom checkpoint storage directory")
	flags.SetAnnotation("checkpoint-dir", "experimental", nil)
	flags.SetAnnotation("checkpoint-dir", "ostype", []string{"linux"})
	return cmd
}

// RunStart executes a `start` command
//
//nolint:gocyclo
func RunStart(ctx context.Context, dockerCli command.Cli, opts *StartOptions) error {
	ctx, cancelFun := context.WithCancel(ctx)
	defer cancelFun()

	switch {
	case opts.Attach || opts.OpenStdin:
		// We're going to attach to a container.
		// 1. Ensure we only have one container.
		if len(opts.Containers) > 1 {
			return errors.New("you cannot start and attach multiple containers at once")
		}

		// 2. Attach to the container.
		ctr := opts.Containers[0]
		c, err := dockerCli.Client().ContainerInspect(ctx, ctr, client.ContainerInspectOptions{})
		if err != nil {
			return err
		}

		// We always use c.ID instead of container to maintain consistency during `docker start`
		if !c.Container.Config.Tty {
			sigc := notifyAllSignals()
			bgCtx := context.WithoutCancel(ctx)
			go ForwardAllSignals(bgCtx, dockerCli.Client(), c.Container.ID, sigc)
			defer signal.StopCatch(sigc)
		}

		detachKeys := dockerCli.ConfigFile().DetachKeys
		if opts.DetachKeys != "" {
			detachKeys = opts.DetachKeys
		}

		options := client.ContainerAttachOptions{
			Stream:     true,
			Stdin:      opts.OpenStdin && c.Container.Config.OpenStdin,
			Stdout:     true,
			Stderr:     true,
			DetachKeys: detachKeys,
		}

		var in io.ReadCloser

		if options.Stdin {
			in = dockerCli.In()
		}

		resp, errAttach := dockerCli.Client().ContainerAttach(ctx, c.Container.ID, options)
		if errAttach != nil {
			return errAttach
		}
		defer resp.HijackedResponse.Close()

		cErr := make(chan error, 1)

		go func() {
			cErr <- func() error {
				streamer := hijackedIOStreamer{
					streams:      dockerCli,
					inputStream:  in,
					outputStream: dockerCli.Out(),
					errorStream:  dockerCli.Err(),
					resp:         resp.HijackedResponse,
					tty:          c.Container.Config.Tty,
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
		statusChan := waitExitOrRemoved(ctx, dockerCli.Client(), c.Container.ID, c.Container.HostConfig.AutoRemove)

		// 4. Start the container.
		_, err = dockerCli.Client().ContainerStart(ctx, c.Container.ID, client.ContainerStartOptions{
			CheckpointID:  opts.Checkpoint,
			CheckpointDir: opts.CheckpointDir,
		})
		if err != nil {
			cancelFun()
			<-cErr
			if c.Container.HostConfig.AutoRemove {
				// wait container to be removed
				<-statusChan
			}
			return err
		}

		// 5. Wait for attachment to break.
		if c.Container.Config.Tty && dockerCli.Out().IsTerminal() {
			if err := MonitorTtySize(ctx, dockerCli, c.Container.ID, false); err != nil {
				_, _ = fmt.Fprintln(dockerCli.Err(), "Error monitoring TTY size:", err)
			}
		}
		if attachErr := <-cErr; attachErr != nil {
			var escapeError term.EscapeError
			if errors.As(attachErr, &escapeError) {
				// The user entered the detach escape sequence.
				return nil
			}
			return attachErr
		}

		if status := <-statusChan; status != 0 {
			return cli.StatusError{StatusCode: status}
		}
		return nil
	case opts.Checkpoint != "":
		if len(opts.Containers) > 1 {
			return errors.New("you cannot restore multiple containers at once")
		}
		ctr := opts.Containers[0]
		_, err := dockerCli.Client().ContainerStart(ctx, ctr, client.ContainerStartOptions{
			CheckpointID:  opts.Checkpoint,
			CheckpointDir: opts.CheckpointDir,
		})
		return err
	default:
		// We're not going to attach to anything.
		// Start as many containers as we want.
		return startContainersWithoutAttachments(ctx, dockerCli, opts.Containers)
	}
}

func startContainersWithoutAttachments(ctx context.Context, dockerCli command.Cli, containers []string) error {
	var failedContainers []string
	for _, ctr := range containers {
		if _, err := dockerCli.Client().ContainerStart(ctx, ctr, client.ContainerStartOptions{}); err != nil {
			_, _ = fmt.Fprintln(dockerCli.Err(), err)
			failedContainers = append(failedContainers, ctr)
			continue
		}
		_, _ = fmt.Fprintln(dockerCli.Out(), ctr)
	}

	if len(failedContainers) > 0 {
		return fmt.Errorf("failed to start containers: %s", strings.Join(failedContainers, ", "))
	}
	return nil
}
