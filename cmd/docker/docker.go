package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/socket"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/commands"
	"github.com/docker/cli/cli/debug"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/cli/version"
	platformsignals "github.com/docker/cli/cmd/docker/internal/signals"
	"github.com/docker/docker/api/types/versions"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
)

type errCtxSignalTerminated struct {
	signal os.Signal
}

func (errCtxSignalTerminated) Error() string {
	return ""
}

func main() {
	err := dockerMain(context.Background())
	if errors.As(err, &errCtxSignalTerminated{}) {
		os.Exit(getExitCode(err))
	}

	if err != nil && !cerrdefs.IsCanceled(err) {
		if err.Error() != "" {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(getExitCode(err))
	}
}

func notifyContext(ctx context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)

	ctxCause, cancel := context.WithCancelCause(ctx)

	go func() {
		select {
		case <-ctx.Done():
			signal.Stop(ch)
			return
		case sig := <-ch:
			cancel(errCtxSignalTerminated{
				signal: sig,
			})
			signal.Stop(ch)
			return
		}
	}()

	return ctxCause, func() {
		signal.Stop(ch)
		cancel(nil)
	}
}

func dockerMain(ctx context.Context) error {
	ctx, cancelNotify := notifyContext(ctx, platformsignals.TerminationSignals...)
	defer cancelNotify()

	dockerCli, err := command.NewDockerCli(command.WithBaseContext(ctx))
	if err != nil {
		return err
	}
	logrus.SetOutput(dockerCli.Err())
	otel.SetErrorHandler(debug.OTELErrorHandler)

	return runDocker(ctx, dockerCli)
}

// getExitCode returns the exit-code to use for the given error.
// If err is a [cli.StatusError] and has a StatusCode set, it uses the
// status-code from it, otherwise it returns "1" for any error.
func getExitCode(err error) int {
	if err == nil {
		return 0
	}

	var userTerminatedErr errCtxSignalTerminated
	if errors.As(err, &userTerminatedErr) {
		s, ok := userTerminatedErr.signal.(syscall.Signal)
		if !ok {
			return 1
		}
		return 128 + int(s)
	}

	var stErr cli.StatusError
	if errors.As(err, &stErr) && stErr.StatusCode != 0 { // FIXME(thaJeztah): StatusCode should never be used with a zero status-code. Check if we do this anywhere.
		return stErr.StatusCode
	}

	// No status-code provided; all errors should have a non-zero exit code.
	return 1
}

func newDockerCommand(dockerCli *command.DockerCli) *cli.TopLevelCommand {
	var (
		opts    *cliflags.ClientOptions
		helpCmd *cobra.Command
	)

	cmd := &cobra.Command{
		Use:              "docker [OPTIONS] COMMAND [ARG...]",
		Short:            "A self-sufficient runtime for containers",
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return command.ShowHelp(dockerCli.Err())(cmd, args)
			}
			return fmt.Errorf("docker: unknown command: docker %s\n\nRun 'docker --help' for more information", args[0])
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return isSupported(cmd, dockerCli)
		},
		Version:               fmt.Sprintf("%s, build %s", version.Version, version.GitCommit),
		DisableFlagsInUseLine: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   false,
			HiddenDefaultCmd:    true,
			DisableDescriptions: os.Getenv("DOCKER_CLI_DISABLE_COMPLETION_DESCRIPTION") != "",
		},
	}
	cmd.SetIn(dockerCli.In())
	cmd.SetOut(dockerCli.Out())
	cmd.SetErr(dockerCli.Err())

	opts, helpCmd = cli.SetupRootCommand(cmd)

	// TODO(thaJeztah): move configuring completion for these flags to where the flags are added.
	_ = cmd.RegisterFlagCompletionFunc("context", completeContextNames(dockerCli))
	_ = cmd.RegisterFlagCompletionFunc("log-level", completeLogLevels)

	cmd.Flags().BoolP("version", "v", false, "Print version information and quit")
	setFlagErrorFunc(dockerCli, cmd)

	setupHelpCommand(dockerCli, cmd, helpCmd)
	setHelpFunc(dockerCli, cmd)

	cmd.SetOut(dockerCli.Out())
	commands.AddCommands(cmd, dockerCli)

	cli.DisableFlagsInUseLine(cmd)
	setValidateArgs(dockerCli, cmd)

	// flags must be the top-level command flags, not cmd.Flags()
	return cli.NewTopLevelCommand(cmd, dockerCli, opts, cmd.Flags())
}

func setFlagErrorFunc(dockerCli command.Cli, cmd *cobra.Command) {
	// When invoking `docker stack --nonsense`, we need to make sure FlagErrorFunc return appropriate
	// output if the feature is not supported.
	// As above cli.SetupRootCommand(cmd) have already setup the FlagErrorFunc, we will add a pre-check before the FlagErrorFunc
	// is called.
	flagErrorFunc := cmd.FlagErrorFunc()
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err := pluginmanager.AddPluginCommandStubs(dockerCli, cmd.Root()); err != nil {
			return err
		}
		if err := isSupported(cmd, dockerCli); err != nil {
			return err
		}
		if err := hideUnsupportedFeatures(cmd, dockerCli); err != nil {
			return err
		}
		return flagErrorFunc(cmd, err)
	})
}

func setupHelpCommand(dockerCli command.Cli, rootCmd, helpCmd *cobra.Command) {
	origRun := helpCmd.Run
	origRunE := helpCmd.RunE

	helpCmd.Run = nil
	helpCmd.RunE = func(c *cobra.Command, args []string) error {
		if len(args) > 0 {
			helpcmd, err := pluginmanager.PluginRunCommand(dockerCli, args[0], rootCmd)
			if err == nil {
				return helpcmd.Run()
			}
			if !pluginmanager.IsNotFound(err) {
				return fmt.Errorf("unknown help topic: %v", strings.Join(args, " "))
			}
		}
		if origRunE != nil {
			return origRunE(c, args)
		}
		origRun(c, args)
		return nil
	}
}

func tryRunPluginHelp(dockerCli command.Cli, ccmd *cobra.Command, cargs []string) error {
	root := ccmd.Root()

	cmd, _, err := root.Traverse(cargs)
	if err != nil {
		return err
	}
	helpcmd, err := pluginmanager.PluginRunCommand(dockerCli, cmd.Name(), root)
	if err != nil {
		return err
	}
	return helpcmd.Run()
}

func setHelpFunc(dockerCli command.Cli, cmd *cobra.Command) {
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(ccmd *cobra.Command, args []string) {
		if err := pluginmanager.AddPluginCommandStubs(dockerCli, ccmd.Root()); err != nil {
			ccmd.Println(err)
			return
		}

		if len(args) >= 1 {
			err := tryRunPluginHelp(dockerCli, ccmd, args)
			if err == nil {
				return
			}
			if !pluginmanager.IsNotFound(err) {
				ccmd.Println(err)
				return
			}
		}

		// FIXME(thaJeztah): need a better way for this; hiding the command here, so that it's present" by default for generating docs etc.
		if c, _, err := ccmd.Find([]string{"buildx"}); c == nil || err != nil {
			if b, _, _ := ccmd.Find([]string{"bake"}); b != nil {
				b.Hidden = true
			}
		}
		if err := isSupported(ccmd, dockerCli); err != nil {
			ccmd.Println(err)
			return
		}
		if err := hideUnsupportedFeatures(ccmd, dockerCli); err != nil {
			ccmd.Println(err)
			return
		}

		defaultHelpFunc(ccmd, args)
	})
}

func setValidateArgs(dockerCli command.Cli, cmd *cobra.Command) {
	// The Args is handled by ValidateArgs in cobra, which does not allows a pre-hook.
	// As a result, here we replace the existing Args validation func to a wrapper,
	// where the wrapper will check to see if the feature is supported or not.
	// The Args validation error will only be returned if the feature is supported.
	cli.VisitAll(cmd, func(ccmd *cobra.Command) {
		// if there is no tags for a command or any of its parent,
		// there is no need to wrap the Args validation.
		if !hasTags(ccmd) {
			return
		}

		if ccmd.Args == nil {
			return
		}

		cmdArgs := ccmd.Args
		ccmd.Args = func(cmd *cobra.Command, args []string) error {
			if err := isSupported(cmd, dockerCli); err != nil {
				return err
			}
			return cmdArgs(cmd, args)
		}
	})
}

func tryPluginRun(ctx context.Context, dockerCli command.Cli, cmd *cobra.Command, subcommand string, envs []string) error {
	plugincmd, err := pluginmanager.PluginRunCommand(dockerCli, subcommand, cmd)
	if err != nil {
		return err
	}

	// Establish the plugin socket, adding it to the environment under a
	// well-known key if successful.
	srv, err := socket.NewPluginServer(nil)
	if err == nil {
		plugincmd.Env = append(plugincmd.Env, socket.EnvKey+"="+srv.Addr().String())
		defer func() {
			// Close the server when plugin execution is over, so that in case
			// it's still open, any sockets on the filesystem are cleaned up.
			_ = srv.Close()
		}()
	}

	// Set additional environment variables specified by the caller.
	plugincmd.Env = append(plugincmd.Env, envs...)

	// Background signal handling logic: block on the signals channel, and
	// notify the plugin via the PluginServer (or signal) as appropriate.
	const exitLimit = 2

	tryTerminatePlugin := func(force bool) {
		// If stdin is a TTY, the kernel will forward
		// signals to the subprocess because the shared
		// pgid makes the TTY a controlling terminal.
		//
		// The plugin should have it's own copy of this
		// termination logic, and exit after 3 retries
		// on it's own.
		if dockerCli.Out().IsTerminal() {
			return
		}

		// Terminate the plugin server, which will
		// close all connections with plugin
		// subprocesses, and signal them to exit.
		//
		// Repeated invocations will result in EINVAL,
		// or EBADF; but that is fine for our purposes.
		if srv != nil {
			_ = srv.Close()
		}

		// force the process to terminate if it hasn't already
		if force {
			_ = plugincmd.Process.Kill()
			_, _ = fmt.Fprint(dockerCli.Err(), "got 3 SIGTERM/SIGINTs, forcefully exiting\n")

			// Restore terminal in case it was in raw mode.
			restoreTerminal(dockerCli)
			os.Exit(1)
		}
	}

	go func() {
		retries := 0
		force := false
		// catch the first signal through context cancellation
		<-ctx.Done()
		tryTerminatePlugin(force)

		// register subsequent signals
		signals := make(chan os.Signal, exitLimit)
		signal.Notify(signals, platformsignals.TerminationSignals...)

		for range signals {
			retries++
			// If we're still running after 3 interruptions
			// (SIGINT/SIGTERM), send a SIGKILL to the plugin as a
			// final attempt to terminate, and exit.
			if retries >= exitLimit {
				force = true
			}
			tryTerminatePlugin(force)
		}
	}()

	if err := plugincmd.Run(); err != nil {
		statusCode := 1
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return err
		}
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			statusCode = ws.ExitStatus()
		}
		return cli.StatusError{
			StatusCode: statusCode,
		}
	}
	return nil
}

// forceExitAfter3TerminationSignals waits for the first termination signal
// to be caught and the context to be marked as done, then registers a new
// signal handler for subsequent signals. It forces the process to exit
// after 3 SIGTERM/SIGINT signals.
func forceExitAfter3TerminationSignals(ctx context.Context, streams command.Streams) {
	// wait for the first signal to be caught and the context to be marked as done
	<-ctx.Done()
	// register a new signal handler for subsequent signals
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, platformsignals.TerminationSignals...)

	// once we have received a total of 3 signals we force exit the cli
	for i := 0; i < 2; i++ {
		<-sig
	}
	_, _ = fmt.Fprint(streams.Err(), "\ngot 3 SIGTERM/SIGINTs, forcefully exiting\n")

	// Restore terminal in case it was in raw mode.
	restoreTerminal(streams)
	os.Exit(1)
}

// restoreTerminal restores the terminal if it was in raw mode; this prevents
// local echo from being disabled for the current terminal after forceful
// termination. It's a no-op if there's no prior state to restore.
func restoreTerminal(streams command.Streams) {
	streams.In().RestoreTerminal()
	streams.Out().RestoreTerminal()
	streams.Err().RestoreTerminal()
}

//nolint:gocyclo
func runDocker(ctx context.Context, dockerCli *command.DockerCli) error {
	tcmd := newDockerCommand(dockerCli)

	cmd, args, err := tcmd.HandleGlobalFlags()
	if err != nil {
		return err
	}

	if err := tcmd.Initialize(command.WithEnableGlobalMeterProvider(), command.WithEnableGlobalTracerProvider()); err != nil {
		return err
	}

	mp := dockerCli.MeterProvider()
	if mp, ok := mp.(command.MeterProvider); ok {
		defer func() {
			if err := mp.Shutdown(ctx); err != nil {
				otel.Handle(err)
			}
		}()
	} else {
		fmt.Fprint(dockerCli.Err(), "Warning: Unexpected OTEL error, metrics may not be flushed")
	}

	dockerCli.InstrumentCobraCommands(ctx, cmd)

	var envs []string
	args, os.Args, envs, err = processAliases(dockerCli, cmd, args, os.Args)
	if err != nil {
		return err
	}

	if cli.HasCompletionArg(args) {
		// We add plugin command stubs early only for completion. We don't
		// want to add them for normal command execution as it would cause
		// a significant performance hit.
		err = pluginmanager.AddPluginCommandStubs(dockerCli, cmd)
		if err != nil {
			return err
		}
	}

	var subCommand *cobra.Command
	if len(args) > 0 {
		ccmd, _, err := cmd.Find(args)
		subCommand = ccmd
		if err != nil || pluginmanager.IsPluginCommand(ccmd) {
			err := tryPluginRun(ctx, dockerCli, cmd, args[0], envs)
			if err == nil {
				if ccmd != nil && dockerCli.Out().IsTerminal() && dockerCli.HooksEnabled() {
					pluginmanager.RunPluginHooks(ctx, dockerCli, cmd, ccmd, args)
				}
				return nil
			}
			if !pluginmanager.IsNotFound(err) {
				// For plugin not found we fall through to
				// cmd.Execute() which deals with reporting
				// "command not found" in a consistent way.
				return err
			}
		}
	}

	// This is a fallback for the case where the command does not exit
	// based on context cancellation.
	go forceExitAfter3TerminationSignals(ctx, dockerCli)

	// We've parsed global args already, so reset args to those
	// which remain.
	cmd.SetArgs(args)
	err = cmd.ExecuteContext(ctx)

	// If the command is being executed in an interactive terminal
	// and hook are enabled, run the plugin hooks.
	if subCommand != nil && dockerCli.Out().IsTerminal() && dockerCli.HooksEnabled() {
		var errMessage string
		if err != nil {
			errMessage = err.Error()
		}
		pluginmanager.RunCLICommandHooks(ctx, dockerCli, cmd, subCommand, errMessage)
	}

	return err
}

type versionDetails interface {
	CurrentVersion() string
	ServerInfo() command.ServerInfo
}

func hideFlagIf(f *pflag.Flag, condition func(string) bool, annotation string) {
	if f.Hidden {
		return
	}
	var val string
	if values, ok := f.Annotations[annotation]; ok {
		if len(values) > 0 {
			val = values[0]
		}
		if condition(val) {
			f.Hidden = true
		}
	}
}

func hideSubcommandIf(subcmd *cobra.Command, condition func(string) bool, annotation string) {
	if subcmd.Hidden {
		return
	}
	if v, ok := subcmd.Annotations[annotation]; ok {
		if condition(v) {
			subcmd.Hidden = true
		}
	}
}

func hideUnsupportedFeatures(cmd *cobra.Command, details versionDetails) error {
	var (
		notExperimental = func(_ string) bool { return !details.ServerInfo().HasExperimental }
		notOSType       = func(v string) bool { return details.ServerInfo().OSType != "" && v != details.ServerInfo().OSType }
		notSwarmStatus  = func(v string) bool {
			s := details.ServerInfo().SwarmStatus
			if s == nil {
				// engine did not return swarm status header
				return false
			}
			switch v {
			case "manager":
				// requires the node to be a manager
				return !s.ControlAvailable
			case "active":
				// requires swarm to be active on the node (e.g. for swarm leave)
				// only hide the command if we're sure the node is "inactive"
				// for any other status, assume the "leave" command can still
				// be used.
				return s.NodeState == "inactive"
			case "":
				// some swarm commands, such as "swarm init" and "swarm join"
				// are swarm-related, but do not require swarm to be active
				return false
			default:
				// ignore any other value for the "swarm" annotation
				return false
			}
		}
		versionOlderThan = func(v string) bool { return versions.LessThan(details.CurrentVersion(), v) }
	)

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// hide flags not supported by the server
		// root command shows all top-level flags
		if cmd.Parent() != nil {
			if cmds, ok := f.Annotations["top-level"]; ok {
				f.Hidden = !findCommand(cmd, cmds)
			}
			if f.Hidden {
				return
			}
		}

		hideFlagIf(f, notExperimental, "experimental")
		hideFlagIf(f, notOSType, "ostype")
		hideFlagIf(f, notSwarmStatus, "swarm")
		hideFlagIf(f, versionOlderThan, "version")
	})

	for _, subcmd := range cmd.Commands() {
		hideSubcommandIf(subcmd, notExperimental, "experimental")
		hideSubcommandIf(subcmd, notOSType, "ostype")
		hideSubcommandIf(subcmd, notSwarmStatus, "swarm")
		hideSubcommandIf(subcmd, versionOlderThan, "version")
	}
	return nil
}

// Checks if a command or one of its ancestors is in the list
func findCommand(cmd *cobra.Command, cmds []string) bool {
	if cmd == nil {
		return false
	}
	for _, c := range cmds {
		if c == cmd.Name() {
			return true
		}
	}
	return findCommand(cmd.Parent(), cmds)
}

func isSupported(cmd *cobra.Command, details versionDetails) error {
	if err := areSubcommandsSupported(cmd, details); err != nil {
		return err
	}
	return areFlagsSupported(cmd, details)
}

func areFlagsSupported(cmd *cobra.Command, details versionDetails) error {
	var errs []error

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed || len(f.Annotations) == 0 {
			return
		}
		// Important: in the code below, calls to "details.CurrentVersion()" and
		// "details.ServerInfo()" are deliberately executed inline to make them
		// be executed "lazily". This is to prevent making a connection with the
		// daemon to perform a "ping" (even for flags that do not require a
		// daemon connection).
		//
		// See commit b39739123b845f872549e91be184cc583f5b387c for details.

		if _, ok := f.Annotations["version"]; ok && !isVersionSupported(f, details.CurrentVersion()) {
			errs = append(errs, fmt.Errorf(`"--%s" requires API version %s, but the Docker daemon API version is %s`, f.Name, getFlagAnnotation(f, "version"), details.CurrentVersion()))
			return
		}
		if _, ok := f.Annotations["ostype"]; ok && !isOSTypeSupported(f, details.ServerInfo().OSType) {
			errs = append(errs, fmt.Errorf(
				`"--%s" is only supported on a Docker daemon running on %s, but the Docker daemon is running on %s`,
				f.Name,
				getFlagAnnotation(f, "ostype"), details.ServerInfo().OSType),
			)
			return
		}
		if _, ok := f.Annotations["experimental"]; ok && !details.ServerInfo().HasExperimental {
			errs = append(errs, fmt.Errorf(`"--%s" is only supported on a Docker daemon with experimental features enabled`, f.Name))
		}
		// buildkit-specific flags are noop when buildkit is not enabled, so we do not add an error in that case
	})
	return errors.Join(errs...)
}

// Check recursively so that, e.g., `docker stack ls` returns the same output as `docker stack`
func areSubcommandsSupported(cmd *cobra.Command, details versionDetails) error {
	// Check recursively so that, e.g., `docker stack ls` returns the same output as `docker stack`
	for curr := cmd; curr != nil; curr = curr.Parent() {
		// Important: in the code below, calls to "details.CurrentVersion()" and
		// "details.ServerInfo()" are deliberately executed inline to make them
		// be executed "lazily". This is to prevent making a connection with the
		// daemon to perform a "ping" (even for commands that do not require a
		// daemon connection).
		//
		// See commit b39739123b845f872549e91be184cc583f5b387c for details.

		if cmdVersion, ok := curr.Annotations["version"]; ok && versions.LessThan(details.CurrentVersion(), cmdVersion) {
			return fmt.Errorf("%s requires API version %s, but the Docker daemon API version is %s", cmd.CommandPath(), cmdVersion, details.CurrentVersion())
		}
		if ost, ok := curr.Annotations["ostype"]; ok && details.ServerInfo().OSType != "" && ost != details.ServerInfo().OSType {
			return fmt.Errorf("%s is only supported on a Docker daemon running on %s, but the Docker daemon is running on %s", cmd.CommandPath(), ost, details.ServerInfo().OSType)
		}
		if _, ok := curr.Annotations["experimental"]; ok && !details.ServerInfo().HasExperimental {
			return fmt.Errorf("%s is only supported on a Docker daemon with experimental features enabled", cmd.CommandPath())
		}
	}
	return nil
}

func getFlagAnnotation(f *pflag.Flag, annotation string) string {
	if value, ok := f.Annotations[annotation]; ok && len(value) == 1 {
		return value[0]
	}
	return ""
}

func isVersionSupported(f *pflag.Flag, clientVersion string) bool {
	if v := getFlagAnnotation(f, "version"); v != "" {
		return versions.GreaterThanOrEqualTo(clientVersion, v)
	}
	return true
}

func isOSTypeSupported(f *pflag.Flag, osType string) bool {
	if v := getFlagAnnotation(f, "ostype"); v != "" && osType != "" {
		return osType == v
	}
	return true
}

// hasTags return true if any of the command's parents has tags
func hasTags(cmd *cobra.Command) bool {
	for curr := cmd; curr != nil; curr = curr.Parent() {
		if len(curr.Annotations) > 0 {
			return true
		}
	}

	return false
}
