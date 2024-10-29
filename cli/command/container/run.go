package container

import (
	"context"
	"fmt"
	"io"
	"strings"
	"syscall"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/container"
	"github.com/moby/sys/signal"
	"github.com/moby/term"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type runOptions struct {
	createOptions
	detach     bool
	sigProxy   bool
	detachKeys string
}

// NewRunCommand create a new `docker run` command
func NewRunCommand(dockerCli command.Cli) *cobra.Command {
	var options runOptions
	var copts *containerOptions

	cmd := &cobra.Command{
		Use:   "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
		Short: "Create and run a new container from an image",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			copts.Image = args[0]
			if len(args) > 1 {
				copts.Args = args[1:]
			}
			return runRun(cmd.Context(), dockerCli, cmd.Flags(), &options, copts)
		},
		ValidArgsFunction: completion.ImageNames(dockerCli),
		Annotations: map[string]string{
			"category-top": "1",
			"aliases":      "docker container run, docker run",
		},
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	// These are flags not stored in Config/HostConfig
	flags.BoolVarP(&options.detach, "detach", "d", false, "Run container in background and print container ID")
	flags.BoolVar(&options.sigProxy, "sig-proxy", true, "Proxy received signals to the process")
	flags.StringVar(&options.name, "name", "", "Assign a name to the container")
	flags.StringVar(&options.detachKeys, "detach-keys", "", "Override the key sequence for detaching a container")
	flags.StringVar(&options.pull, "pull", PullImageMissing, `Pull image before running ("`+PullImageAlways+`", "`+PullImageMissing+`", "`+PullImageNever+`")`)
	_ = cmd.RegisterFlagCompletionFunc("pull", completion.FromList(PullImageAlways, PullImageMissing, PullImageNever))
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Suppress the pull output")

	// Add an explicit help that doesn't have a `-h` to prevent the conflict
	// with hostname
	flags.Bool("help", false, "Print usage")

	command.AddPlatformFlag(flags, &options.platform)
	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)
	command.AddTrustVerificationFlags(flags, &options.untrusted, dockerCli.ContentTrustEnabled())
	copts = addContainerCreateFlags(cmd, dockerCli)
	return cmd
}

func runRun(ctx context.Context, dockerCli command.Cli, flags *pflag.FlagSet, ropts *runOptions, copts *containerOptions) error {
	if err := validatePullOpt(ropts.pull); err != nil {
		return cli.StatusError{
			Status:     withHelp(err, "run").Error(),
			StatusCode: 125,
		}
	}
	proxyConfig := dockerCli.ConfigFile().ParseProxyConfig(dockerCli.Client().DaemonHost(), opts.ConvertKVStringsToMapWithNil(copts.env.GetAll()))
	newEnv := []string{}
	for k, v := range proxyConfig {
		if v == nil {
			newEnv = append(newEnv, k)
		} else {
			newEnv = append(newEnv, k+"="+*v)
		}
	}
	copts.env = *opts.NewListOptsRef(&newEnv, nil)
	containerCfg, err := parse(flags, copts, dockerCli.ServerInfo().OSType)
	// just in case the parse does not exit
	if err != nil {
		return cli.StatusError{
			Status:     withHelp(err, "run").Error(),
			StatusCode: 125,
		}
	}
	if err = validateAPIVersion(containerCfg, dockerCli.CurrentVersion()); err != nil {
		return cli.StatusError{
			Status:     withHelp(err, "run").Error(),
			StatusCode: 125,
		}
	}
	return runContainer(ctx, dockerCli, ropts, copts, containerCfg)
}

//nolint:gocyclo
func runContainer(ctx context.Context, dockerCli command.Cli, runOpts *runOptions, copts *containerOptions, containerCfg *containerConfig) error {
	ctx = context.WithoutCancel(ctx)

	config := containerCfg.Config
	stdout, stderr := dockerCli.Out(), dockerCli.Err()
	apiClient := dockerCli.Client()

	config.ArgsEscaped = false

	if !runOpts.detach {
		if err := dockerCli.In().CheckTty(config.AttachStdin, config.Tty); err != nil {
			return err
		}
	} else {
		if copts.attach.Len() != 0 {
			return errors.New("conflicting options: cannot specify both --attach and --detach")
		}

		config.AttachStdin = false
		config.AttachStdout = false
		config.AttachStderr = false
		config.StdinOnce = false
	}

	ctx, cancelFun := context.WithCancel(ctx)
	defer cancelFun()

	containerID, err := createContainer(ctx, dockerCli, containerCfg, &runOpts.createOptions)
	if err != nil {
		return toStatusError(err)
	}
	if runOpts.sigProxy {
		sigc := notifyAllSignals()
		// since we're explicitly setting up signal handling here, and the daemon will
		// get notified independently of the clients ctx cancellation, we use this context
		// but without cancellation to avoid ForwardAllSignals from returning
		// before all signals are forwarded.
		bgCtx := context.WithoutCancel(ctx)
		go ForwardAllSignals(bgCtx, apiClient, containerID, sigc)
		defer signal.StopCatch(sigc)
	}

	var (
		waitDisplayID chan struct{}
		errCh         chan error
	)
	if !config.AttachStdout && !config.AttachStderr {
		// Make this asynchronous to allow the client to write to stdin before having to read the ID
		waitDisplayID = make(chan struct{})
		go func() {
			defer close(waitDisplayID)
			_, _ = fmt.Fprintln(stdout, containerID)
		}()
	}
	attach := config.AttachStdin || config.AttachStdout || config.AttachStderr
	if attach {
		detachKeys := dockerCli.ConfigFile().DetachKeys
		if runOpts.detachKeys != "" {
			detachKeys = runOpts.detachKeys
		}

		// ctx should not be cancellable here, as this would kill the stream to the container
		// and we want to keep the stream open until the process in the container exits or until
		// the user forcefully terminates the CLI.
		closeFn, err := attachContainer(ctx, dockerCli, containerID, &errCh, config, container.AttachOptions{
			Stream:     true,
			Stdin:      config.AttachStdin,
			Stdout:     config.AttachStdout,
			Stderr:     config.AttachStderr,
			DetachKeys: detachKeys,
		})
		if err != nil {
			return err
		}
		defer closeFn()
	}

	// New context here because we don't to cancel waiting on container exit/remove
	// when we cancel attach, etc.
	statusCtx, cancelStatusCtx := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelStatusCtx()
	statusChan := waitExitOrRemoved(statusCtx, apiClient, containerID, copts.autoRemove)

	// start the container
	if err := apiClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		// If we have hijackedIOStreamer, we should notify
		// hijackedIOStreamer we are going to exit and wait
		// to avoid the terminal are not restored.
		if attach {
			cancelFun()
			<-errCh
		}

		if copts.autoRemove {
			// wait container to be removed
			<-statusChan
		}
		return toStatusError(err)
	}

	if (config.AttachStdin || config.AttachStdout || config.AttachStderr) && config.Tty && dockerCli.Out().IsTerminal() {
		if err := MonitorTtySize(ctx, dockerCli, containerID, false); err != nil {
			_, _ = fmt.Fprintln(stderr, "Error monitoring TTY size:", err)
		}
	}

	if errCh != nil {
		if err := <-errCh; err != nil {
			if _, ok := err.(term.EscapeError); ok {
				// The user entered the detach escape sequence.
				return nil
			}

			logrus.Debugf("Error hijack: %s", err)
			return err
		}
	}

	// Detached mode: wait for the id to be displayed and return.
	if !config.AttachStdout && !config.AttachStderr {
		// Detached mode
		<-waitDisplayID
		return nil
	}

	status := <-statusChan
	if status != 0 {
		return cli.StatusError{StatusCode: status}
	}
	return nil
}

func attachContainer(ctx context.Context, dockerCli command.Cli, containerID string, errCh *chan error, config *container.Config, options container.AttachOptions) (func(), error) {
	resp, errAttach := dockerCli.Client().ContainerAttach(ctx, containerID, options)
	if errAttach != nil {
		return nil, errAttach
	}

	var (
		out, cerr io.Writer
		in        io.ReadCloser
	)
	if options.Stdin {
		in = dockerCli.In()
	}
	if options.Stdout {
		out = dockerCli.Out()
	}
	if options.Stderr {
		if config.Tty {
			cerr = dockerCli.Out()
		} else {
			cerr = dockerCli.Err()
		}
	}

	ch := make(chan error, 1)
	*errCh = ch

	go func() {
		ch <- func() error {
			streamer := hijackedIOStreamer{
				streams:      dockerCli,
				inputStream:  in,
				outputStream: out,
				errorStream:  cerr,
				resp:         resp,
				tty:          config.Tty,
				detachKeys:   options.DetachKeys,
			}

			if errHijack := streamer.stream(ctx); errHijack != nil {
				return errHijack
			}
			return errAttach
		}()
	}()
	return resp.Close, nil
}

// withHelp decorates the error with a suggestion to use "--help".
func withHelp(err error, commandName string) error {
	return fmt.Errorf("docker: %w\n\nRun 'docker %s --help' for more information", err, commandName)
}

// toStatusError attempts to detect specific error-conditions to assign
// an appropriate exit-code for situations where the problem originates
// from the container. It returns [cli.StatusError] with the original
// error message and the Status field set as follows:
//
// - 125: for generic failures sent back from the daemon
// - 126: if container start fails with 'permission denied' error
// - 127: if container start fails with 'not found'/'no such' error
func toStatusError(err error) error {
	// TODO(thaJeztah): some of these errors originate from the container: should we actually suggest "--help" for those?

	errMsg := err.Error()

	if strings.Contains(errMsg, "executable file not found") || strings.Contains(errMsg, "no such file or directory") || strings.Contains(errMsg, "system cannot find the file specified") {
		return cli.StatusError{
			Status:     withHelp(err, "run").Error(),
			StatusCode: 127,
		}
	}

	if strings.Contains(errMsg, syscall.EACCES.Error()) || strings.Contains(errMsg, syscall.EISDIR.Error()) {
		return cli.StatusError{
			Status:     withHelp(err, "run").Error(),
			StatusCode: 126,
		}
	}

	return cli.StatusError{
		Status:     withHelp(err, "run").Error(),
		StatusCode: 125,
	}
}
