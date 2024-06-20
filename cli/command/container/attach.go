package container

import (
	"context"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/moby/sys/signal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// AttachOptions group options for `attach` command
type AttachOptions struct {
	NoStdin    bool
	Proxy      bool
	DetachKeys string
}

func inspectContainerAndCheckState(ctx context.Context, apiClient client.APIClient, args string) (*types.ContainerJSON, error) {
	c, err := apiClient.ContainerInspect(ctx, args)
	if err != nil {
		return nil, err
	}
	if !c.State.Running {
		return nil, errors.New("You cannot attach to a stopped container, start it first")
	}
	if c.State.Paused {
		return nil, errors.New("You cannot attach to a paused container, unpause it first")
	}
	if c.State.Restarting {
		return nil, errors.New("You cannot attach to a restarting container, wait until it is running")
	}

	return &c, nil
}

// NewAttachCommand creates a new cobra.Command for `docker attach`
func NewAttachCommand(dockerCLI command.Cli) *cobra.Command {
	var opts AttachOptions

	cmd := &cobra.Command{
		Use:   "attach [OPTIONS] CONTAINER",
		Short: "Attach local standard input, output, and error streams to a running container",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attachStdIn := true
			if opts.NoStdin {
				// TODO(thaJeztah): this is the tricky one: can we use container.AttachOptions for this one without it being ambiguous?
				attachStdIn = false
			}

			containerID := args[0]
			disableSignalProxy := !opts.Proxy
			return RunAttach(cmd.Context(), dockerCLI, containerID, disableSignalProxy, container.AttachOptions{
				Stream:     true,
				Stdin:      attachStdIn,
				Stdout:     true,
				Stderr:     true,
				DetachKeys: opts.DetachKeys,
			})
		},
		Annotations: map[string]string{
			"aliases": "docker container attach, docker attach",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCLI, false, func(ctr types.Container) bool {
			return ctr.State != "paused"
		}),
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.NoStdin, "no-stdin", false, "Do not attach STDIN")
	// Is this feature still used?
	// It was added in https://github.com/moby/moby/commit/4918769b1ac2d38f23087b766140e6a7f8979310 to allow forwarding signals to containers
	// Changed in https://github.com/moby/moby/commit/333bc23f21e8423d3085632db99a6d1df227c5f1
	// And changed to be enabled by default in https://github.com/moby/moby/commit/e0b59ab52b87b8fc15dd5534c3231fdd74843f9f (unless a TTY is attached)
	// related: https://github.com/moby/moby/commit/e0b59ab52b87b8fc15dd5534c3231fdd74843f9f#commitcomment-25897874
	// related: https://github.com/moby/moby/issues/9098
	// related: https://github.com/docker/cli/pull/1841
	flags.BoolVar(&opts.Proxy, "sig-proxy", true, "Proxy all received signals to the process")
	flags.StringVar(&opts.DetachKeys, "detach-keys", "", "Override the key sequence for detaching a container")
	return cmd
}

// RunAttach attaches to the given container.
func RunAttach(ctx context.Context, dockerCLI command.Cli, containerID string, disableSignalProxy bool, opts container.AttachOptions) error {
	apiClient := dockerCLI.Client()

	c, err := inspectContainerAndCheckState(ctx, apiClient, containerID)
	if err != nil {
		return err
	}

	if opts.Stdin {
		if err := dockerCLI.In().CheckTty(opts.Stdin, c.Config.Tty); err != nil {
			return err
		}
		if !c.Config.OpenStdin {
			// TODO(thaJeztah): should this produce an error?
			opts.Stdin = false
		}
	}

	if opts.DetachKeys == "" {
		opts.DetachKeys = dockerCLI.ConfigFile().DetachKeys
	}

	var in io.ReadCloser
	if opts.Stdin {
		in = dockerCLI.In()
	}

	// TODO(thaJeztah): should this still depend on Config.Tty? It's unconditionally enabled on `docker exec` since https://github.com/docker/cli/pull/1841/files
	if !disableSignalProxy && !c.Config.Tty {
		sigc := notifyAllSignals()
		// since we're explicitly setting up signal handling here, and the daemon will
		// get notified independently of the clients ctx cancellation, we use this context
		// but without cancellation to avoid ForwardAllSignals from returning
		// before all signals are forwarded.
		bgCtx := context.WithoutCancel(ctx)
		go ForwardAllSignals(bgCtx, apiClient, containerID, sigc)
		defer signal.StopCatch(sigc)
	}

	resp, errAttach := apiClient.ContainerAttach(ctx, containerID, opts)
	if errAttach != nil {
		return errAttach
	}
	defer resp.Close()

	// If use docker attach command to attach to a stop container, it will return
	// "You cannot attach to a stopped container" error, it's ok, but when
	// attach to a running container, it(docker attach) use inspect to check
	// the container's state, if it pass the state check on the client side,
	// and then the container is stopped, docker attach command still attach to
	// the container and not exit.
	//
	// Recheck the container's state to avoid attach block.
	_, err = inspectContainerAndCheckState(ctx, apiClient, containerID)
	if err != nil {
		return err
	}

	if c.Config.Tty && dockerCLI.Out().IsTerminal() {
		resizeTTY(ctx, dockerCLI, containerID)
	}

	streamer := hijackedIOStreamer{
		streams:      dockerCLI,
		inputStream:  in,
		outputStream: dockerCLI.Out(),
		errorStream:  dockerCLI.Err(),
		resp:         resp,
		tty:          c.Config.Tty,
		detachKeys:   opts.DetachKeys,
	}

	if err := streamer.stream(ctx); err != nil {
		return err
	}

	// request channel to wait for client
	resultC, errC := apiClient.ContainerWait(ctx, containerID, "")
	return getExitStatus(errC, resultC)
}

func getExitStatus(errC <-chan error, resultC <-chan container.WaitResponse) error {
	select {
	case result := <-resultC:
		if result.Error != nil {
			return errors.New(result.Error.Message)
		}
		if result.StatusCode != 0 {
			return cli.StatusError{StatusCode: int(result.StatusCode)}
		}
	case err := <-errC:
		return err
	}

	return nil
}

func resizeTTY(ctx context.Context, dockerCli command.Cli, containerID string) {
	height, width := dockerCli.Out().GetTtySize()
	// To handle the case where a user repeatedly attaches/detaches without resizing their
	// terminal, the only way to get the shell prompt to display for attaches 2+ is to artificially
	// resize it, then go back to normal. Without this, every attach after the first will
	// require the user to manually resize or hit enter.
	resizeTtyTo(ctx, dockerCli.Client(), containerID, height+1, width+1, false)

	// After the above resizing occurs, the call to MonitorTtySize below will handle resetting back
	// to the actual size.
	if err := MonitorTtySize(ctx, dockerCli, containerID, false); err != nil {
		logrus.Debugf("Error monitoring TTY size: %s", err)
	}
}
