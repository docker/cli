package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/docker/cli/cli"
	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/commands"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/cli/version"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newDockerCommand(dockerCli *command.DockerCli) *cli.TopLevelCommand {
	var (
		opts    *cliflags.ClientOptions
		flags   *pflag.FlagSet
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
			return fmt.Errorf("docker: '%s' is not a docker command.\nSee 'docker --help'", args[0])

		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return isSupported(cmd, dockerCli)
		},
		Version:               fmt.Sprintf("%s, build %s", version.Version, version.GitCommit),
		DisableFlagsInUseLine: true,
	}
	opts, flags, helpCmd = cli.SetupRootCommand(cmd)
	flags.BoolP("version", "v", false, "Print version information and quit")

	setFlagErrorFunc(dockerCli, cmd)

	setupHelpCommand(dockerCli, cmd, helpCmd)
	setHelpFunc(dockerCli, cmd)

	cmd.SetOutput(dockerCli.Out())
	commands.AddCommands(cmd, dockerCli)

	cli.DisableFlagsInUseLine(cmd)
	setValidateArgs(dockerCli, cmd)

	// flags must be the top-level command flags, not cmd.Flags()
	return cli.NewTopLevelCommand(cmd, dockerCli, opts, flags)
}

func setFlagErrorFunc(dockerCli *command.DockerCli, cmd *cobra.Command) {
	// When invoking `docker stack --nonsense`, we need to make sure FlagErrorFunc return appropriate
	// output if the feature is not supported.
	// As above cli.SetupRootCommand(cmd) have already setup the FlagErrorFunc, we will add a pre-check before the FlagErrorFunc
	// is called.
	flagErrorFunc := cmd.FlagErrorFunc()
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err := isSupported(cmd, dockerCli); err != nil {
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
				err = helpcmd.Run()
				if err != nil {
					return err
				}
			}
			if !pluginmanager.IsNotFound(err) {
				return err
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
		// Add a stub entry for every plugin so they are
		// included in the help output and so that
		// `tryRunPluginHelp` can find them or if we fall
		// through they will be included in the default help
		// output.
		if err := pluginmanager.AddPluginCommandStubs(dockerCli, ccmd.Root()); err != nil {
			ccmd.Println(err)
			return
		}

		if len(args) >= 1 {
			err := tryRunPluginHelp(dockerCli, ccmd, args)
			if err == nil { // Successfully ran the plugin
				return
			}
			if !pluginmanager.IsNotFound(err) {
				ccmd.Println(err)
				return
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

func setValidateArgs(dockerCli *command.DockerCli, cmd *cobra.Command) {
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

func tryPluginRun(dockerCli command.Cli, cmd *cobra.Command, subcommand string) error {
	plugincmd, err := pluginmanager.PluginRunCommand(dockerCli, subcommand, cmd)
	if err != nil {
		return err
	}

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

func runDocker(dockerCli *command.DockerCli) error {
	tcmd := newDockerCommand(dockerCli)

	cmd, args, err := tcmd.HandleGlobalFlags()
	if err != nil {
		return err
	}

	if err := tcmd.Initialize(); err != nil {
		return err
	}

	if len(args) > 0 {
		if _, _, err := cmd.Find(args); err != nil {
			err := tryPluginRun(dockerCli, cmd, args[0])
			if !pluginmanager.IsNotFound(err) {
				return err
			}
			// For plugin not found we fall through to
			// cmd.Execute() which deals with reporting
			// "command not found" in a consistent way.
		}
	}

	return cmd.Execute()
}

func main() {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logrus.SetOutput(dockerCli.Err())

	if err := runDocker(dockerCli); err != nil {
		if sterr, ok := err.(cli.StatusError); ok {
			if sterr.Status != "" {
				fmt.Fprintln(dockerCli.Err(), sterr.Status)
			}
			// StatusError should only be used for errors, and all errors should
			// have a non-zero exit status, so never exit with 0
			if sterr.StatusCode == 0 {
				os.Exit(1)
			}
			os.Exit(sterr.StatusCode)
		}
		fmt.Fprintln(dockerCli.Err(), err)
		os.Exit(1)
	}
}

type versionDetails interface {
	Client() client.APIClient
	ClientInfo() command.ClientInfo
	ServerInfo() command.ServerInfo
}

func hideFeatureFlag(f *pflag.Flag, hasFeature bool, annotation string) {
	if hasFeature {
		return
	}
	if _, ok := f.Annotations[annotation]; ok {
		f.Hidden = true
	}
}

func hideFeatureSubCommand(subcmd *cobra.Command, hasFeature bool, annotation string) {
	if hasFeature {
		return
	}
	if _, ok := subcmd.Annotations[annotation]; ok {
		subcmd.Hidden = true
	}
}

func hideUnsupportedFeatures(cmd *cobra.Command, details versionDetails) error {
	clientVersion := details.Client().ClientVersion()
	osType := details.ServerInfo().OSType
	hasExperimental := details.ServerInfo().HasExperimental
	hasExperimentalCLI := details.ClientInfo().HasExperimental
	hasBuildKit, err := command.BuildKitEnabled(details.ServerInfo())
	if err != nil {
		return err
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		hideFeatureFlag(f, hasExperimental, "experimental")
		hideFeatureFlag(f, hasExperimentalCLI, "experimentalCLI")
		hideFeatureFlag(f, hasBuildKit, "buildkit")
		hideFeatureFlag(f, !hasBuildKit, "no-buildkit")
		// hide flags not supported by the server
		if !isOSTypeSupported(f, osType) || !isVersionSupported(f, clientVersion) {
			f.Hidden = true
		}
		// root command shows all top-level flags
		if cmd.Parent() != nil {
			if commands, ok := f.Annotations["top-level"]; ok {
				f.Hidden = !findCommand(cmd, commands)
			}
		}
	})

	for _, subcmd := range cmd.Commands() {
		hideFeatureSubCommand(subcmd, hasExperimental, "experimental")
		hideFeatureSubCommand(subcmd, hasExperimentalCLI, "experimentalCLI")
		hideFeatureSubCommand(subcmd, hasBuildKit, "buildkit")
		hideFeatureSubCommand(subcmd, !hasBuildKit, "no-buildkit")
		// hide subcommands not supported by the server
		if subcmdVersion, ok := subcmd.Annotations["version"]; ok && versions.LessThan(clientVersion, subcmdVersion) {
			subcmd.Hidden = true
		}
		if v, ok := subcmd.Annotations["ostype"]; ok && v != osType {
			subcmd.Hidden = true
		}
	}
	return nil
}

// Checks if a command or one of its ancestors is in the list
func findCommand(cmd *cobra.Command, commands []string) bool {
	if cmd == nil {
		return false
	}
	for _, c := range commands {
		if c == cmd.Name() {
			return true
		}
	}
	return findCommand(cmd.Parent(), commands)
}

func isSupported(cmd *cobra.Command, details versionDetails) error {
	if err := areSubcommandsSupported(cmd, details); err != nil {
		return err
	}
	return areFlagsSupported(cmd, details)
}

func areFlagsSupported(cmd *cobra.Command, details versionDetails) error {
	clientVersion := details.Client().ClientVersion()
	osType := details.ServerInfo().OSType
	hasExperimental := details.ServerInfo().HasExperimental
	hasExperimentalCLI := details.ClientInfo().HasExperimental

	errs := []string{}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			if !isVersionSupported(f, clientVersion) {
				errs = append(errs, fmt.Sprintf("\"--%s\" requires API version %s, but the Docker daemon API version is %s", f.Name, getFlagAnnotation(f, "version"), clientVersion))
				return
			}
			if !isOSTypeSupported(f, osType) {
				errs = append(errs, fmt.Sprintf("\"--%s\" is only supported on a Docker daemon running on %s, but the Docker daemon is running on %s", f.Name, getFlagAnnotation(f, "ostype"), osType))
				return
			}
			if _, ok := f.Annotations["experimental"]; ok && !hasExperimental {
				errs = append(errs, fmt.Sprintf("\"--%s\" is only supported on a Docker daemon with experimental features enabled", f.Name))
			}
			if _, ok := f.Annotations["experimentalCLI"]; ok && !hasExperimentalCLI {
				errs = append(errs, fmt.Sprintf("\"--%s\" is on a Docker cli with experimental cli features enabled", f.Name))
			}
			// buildkit-specific flags are noop when buildkit is not enabled, so we do not add an error in that case
		}
	})
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

// Check recursively so that, e.g., `docker stack ls` returns the same output as `docker stack`
func areSubcommandsSupported(cmd *cobra.Command, details versionDetails) error {
	clientVersion := details.Client().ClientVersion()
	osType := details.ServerInfo().OSType
	hasExperimental := details.ServerInfo().HasExperimental
	hasExperimentalCLI := details.ClientInfo().HasExperimental

	// Check recursively so that, e.g., `docker stack ls` returns the same output as `docker stack`
	for curr := cmd; curr != nil; curr = curr.Parent() {
		if cmdVersion, ok := curr.Annotations["version"]; ok && versions.LessThan(clientVersion, cmdVersion) {
			return fmt.Errorf("%s requires API version %s, but the Docker daemon API version is %s", cmd.CommandPath(), cmdVersion, clientVersion)
		}
		if os, ok := curr.Annotations["ostype"]; ok && os != osType {
			return fmt.Errorf("%s is only supported on a Docker daemon running on %s, but the Docker daemon is running on %s", cmd.CommandPath(), os, osType)
		}
		if _, ok := curr.Annotations["experimental"]; ok && !hasExperimental {
			return fmt.Errorf("%s is only supported on a Docker daemon with experimental features enabled", cmd.CommandPath())
		}
		if _, ok := curr.Annotations["experimentalCLI"]; ok && !hasExperimentalCLI {
			return fmt.Errorf("%s is only supported on a Docker cli with experimental cli features enabled", cmd.CommandPath())
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
