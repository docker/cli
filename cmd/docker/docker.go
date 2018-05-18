package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/commands"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/debug"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/term"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newDockerCommand(dockerCli *command.DockerCli) *cobra.Command {
	opts := cliflags.NewClientOptions()
	var flags *pflag.FlagSet

	cmd := &cobra.Command{
		Use:              "docker [OPTIONS] COMMAND [ARG...]",
		Short:            "A self-sufficient runtime for containers",
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true,
		Args:             noArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// flags must be the top-level command flags, not cmd.Flags()
			opts.Common.SetDefaultOptions(flags)
			dockerPreRun(opts)
			if err := dockerCli.Initialize(opts); err != nil {
				return err
			}
			return isSupported(cmd, dockerCli)
		},
		Version:               fmt.Sprintf("%s, build %s", cli.Version, cli.GitCommit),
		DisableFlagsInUseLine: true,
	}
	cli.SetupRootCommand(cmd)

	flags = cmd.Flags()
	flags.BoolP("version", "v", false, "Print version information and quit")
	flags.StringVar(&opts.ConfigDir, "config", cliconfig.Dir(), "Location of client config files")
	opts.Common.InstallFlags(flags)

	// Install persistent flags
	persistentFlags := cmd.PersistentFlags()
	persistentFlags.StringVar(&opts.Common.Orchestrator, "orchestrator", "", "Orchestrator to use (swarm|kubernetes|all)")

	setFlagErrorFunc(dockerCli, cmd, flags, opts)

	setHelpFunc(dockerCli, cmd, flags, opts)

	cmd.SetOutput(dockerCli.Out())
	commands.AddCommands(cmd, dockerCli)

	disableFlagsInUseLine(cmd)
	setValidateArgs(dockerCli, cmd, flags, opts)

	return cmd
}

func disableFlagsInUseLine(cmd *cobra.Command) {
	visitAll(cmd, func(ccmd *cobra.Command) {
		// do not add a `[flags]` to the end of the usage line.
		ccmd.DisableFlagsInUseLine = true
	})
}

func setFlagErrorFunc(dockerCli *command.DockerCli, cmd *cobra.Command, flags *pflag.FlagSet, opts *cliflags.ClientOptions) {
	// When invoking `docker stack --nonsense`, we need to make sure FlagErrorFunc return appropriate
	// output if the feature is not supported.
	// As above cli.SetupRootCommand(cmd) have already setup the FlagErrorFunc, we will add a pre-check before the FlagErrorFunc
	// is called.
	flagErrorFunc := cmd.FlagErrorFunc()
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		initializeDockerCli(dockerCli, flags, opts)
		if err := isSupported(cmd, dockerCli); err != nil {
			return err
		}
		return flagErrorFunc(cmd, err)
	})
}

func setHelpFunc(dockerCli *command.DockerCli, cmd *cobra.Command, flags *pflag.FlagSet, opts *cliflags.ClientOptions) {
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(ccmd *cobra.Command, args []string) {
		initializeDockerCli(dockerCli, flags, opts)
		if err := isSupported(ccmd, dockerCli); err != nil {
			ccmd.Println(err)
			return
		}

		hideUnsupportedFeatures(ccmd, dockerCli)
		defaultHelpFunc(ccmd, args)
	})
}

func setValidateArgs(dockerCli *command.DockerCli, cmd *cobra.Command, flags *pflag.FlagSet, opts *cliflags.ClientOptions) {
	// The Args is handled by ValidateArgs in cobra, which does not allows a pre-hook.
	// As a result, here we replace the existing Args validation func to a wrapper,
	// where the wrapper will check to see if the feature is supported or not.
	// The Args validation error will only be returned if the feature is supported.
	visitAll(cmd, func(ccmd *cobra.Command) {
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
			initializeDockerCli(dockerCli, flags, opts)
			if err := isSupported(cmd, dockerCli); err != nil {
				return err
			}
			return cmdArgs(cmd, args)
		}
	})
}

func initializeDockerCli(dockerCli *command.DockerCli, flags *pflag.FlagSet, opts *cliflags.ClientOptions) {
	if dockerCli.Client() == nil { // when using --help, PersistentPreRun is not called, so initialization is needed.
		// flags must be the top-level command flags, not cmd.Flags()
		opts.Common.SetDefaultOptions(flags)
		dockerPreRun(opts)
		dockerCli.Initialize(opts)
	}
}

// visitAll will traverse all commands from the root.
// This is different from the VisitAll of cobra.Command where only parents
// are checked.
func visitAll(root *cobra.Command, fn func(*cobra.Command)) {
	for _, cmd := range root.Commands() {
		visitAll(cmd, fn)
	}
	fn(root)
}

func noArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return fmt.Errorf(
		"docker: '%s' is not a docker command.\nSee 'docker --help'", args[0])
}

func main() {
	// Set terminal emulation based on platform as required.
	stdin, stdout, stderr := term.StdStreams()
	logrus.SetOutput(stderr)

	dockerCli := command.NewDockerCli(stdin, stdout, stderr, contentTrustEnabled())
	cmd := newDockerCommand(dockerCli)

	if err := cmd.Execute(); err != nil {
		if sterr, ok := err.(cli.StatusError); ok {
			if sterr.Status != "" {
				fmt.Fprintln(stderr, sterr.Status)
			}
			// StatusError should only be used for errors, and all errors should
			// have a non-zero exit status, so never exit with 0
			if sterr.StatusCode == 0 {
				os.Exit(1)
			}
			os.Exit(sterr.StatusCode)
		}
		fmt.Fprintln(stderr, err)
		os.Exit(1)
	}
}

func contentTrustEnabled() bool {
	if e := os.Getenv("DOCKER_CONTENT_TRUST"); e != "" {
		if t, err := strconv.ParseBool(e); t || err != nil {
			// treat any other value as true
			return true
		}
	}
	return false
}

func dockerPreRun(opts *cliflags.ClientOptions) {
	cliflags.SetLogLevel(opts.Common.LogLevel)

	if opts.ConfigDir != "" {
		cliconfig.SetDir(opts.ConfigDir)
	}

	if opts.Common.Debug {
		debug.Enable()
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

func hideUnsupportedFeatures(cmd *cobra.Command, details versionDetails) {
	clientVersion := details.Client().ClientVersion()
	osType := details.ServerInfo().OSType
	hasExperimental := details.ServerInfo().HasExperimental
	hasExperimentalCLI := details.ClientInfo().HasExperimental
	hasKubernetes := details.ClientInfo().HasKubernetes()

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		hideFeatureFlag(f, hasExperimental, "experimental")
		hideFeatureFlag(f, hasExperimentalCLI, "experimentalCLI")
		hideFeatureFlag(f, hasKubernetes, "kubernetes")
		hideFeatureFlag(f, !hasKubernetes, "swarm")
		// hide flags not supported by the server
		if !isOSTypeSupported(f, osType) || !isVersionSupported(f, clientVersion) {
			f.Hidden = true
		}
	})

	for _, subcmd := range cmd.Commands() {
		hideFeatureSubCommand(subcmd, hasExperimental, "experimental")
		hideFeatureSubCommand(subcmd, hasExperimentalCLI, "experimentalCLI")
		hideFeatureSubCommand(subcmd, hasKubernetes, "kubernetes")
		hideFeatureSubCommand(subcmd, !hasKubernetes, "swarm")
		// hide subcommands not supported by the server
		if subcmdVersion, ok := subcmd.Annotations["version"]; ok && versions.LessThan(clientVersion, subcmdVersion) {
			subcmd.Hidden = true
		}
	}
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
	hasKubernetes := details.ClientInfo().HasKubernetes()
	hasExperimentalCLI := details.ClientInfo().HasExperimental

	errs := []string{}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			if !isVersionSupported(f, clientVersion) {
				errs = append(errs, fmt.Sprintf("\"--%s\" requires API version %s, but the Docker daemon API version is %s", f.Name, getFlagAnnotation(f, "version"), clientVersion))
				return
			}
			if !isOSTypeSupported(f, osType) {
				errs = append(errs, fmt.Sprintf("\"--%s\" requires the Docker daemon to run on %s, but the Docker daemon is running on %s", f.Name, getFlagAnnotation(f, "ostype"), osType))
				return
			}
			if _, ok := f.Annotations["experimental"]; ok && !hasExperimental {
				errs = append(errs, fmt.Sprintf("\"--%s\" is only supported on a Docker daemon with experimental features enabled", f.Name))
			}
			if _, ok := f.Annotations["experimentalCLI"]; ok && !hasExperimentalCLI {
				errs = append(errs, fmt.Sprintf("\"--%s\" is only supported when experimental cli features are enabled", f.Name))
			}
			_, isKubernetesAnnotated := f.Annotations["kubernetes"]
			_, isSwarmAnnotated := f.Annotations["swarm"]
			if isKubernetesAnnotated && !isSwarmAnnotated && !hasKubernetes {
				errs = append(errs, fmt.Sprintf("\"--%s\" is only supported on a Docker cli with kubernetes features enabled", f.Name))
			}
			if isSwarmAnnotated && !isKubernetesAnnotated && hasKubernetes {
				errs = append(errs, fmt.Sprintf("\"--%s\" is only supported on a Docker cli with swarm features enabled", f.Name))
			}
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
	hasExperimental := details.ServerInfo().HasExperimental
	hasExperimentalCLI := details.ClientInfo().HasExperimental
	hasKubernetes := details.ClientInfo().HasKubernetes()

	// Check recursively so that, e.g., `docker stack ls` returns the same output as `docker stack`
	for curr := cmd; curr != nil; curr = curr.Parent() {
		if cmdVersion, ok := curr.Annotations["version"]; ok && versions.LessThan(clientVersion, cmdVersion) {
			return fmt.Errorf("%s requires API version %s, but the Docker daemon API version is %s", cmd.CommandPath(), cmdVersion, clientVersion)
		}
		if _, ok := curr.Annotations["experimental"]; ok && !hasExperimental {
			return fmt.Errorf("%s is only supported on a Docker daemon with experimental features enabled", cmd.CommandPath())
		}
		if _, ok := curr.Annotations["experimentalCLI"]; ok && !hasExperimentalCLI {
			return fmt.Errorf("%s is only supported when experimental cli features are enabled", cmd.CommandPath())
		}
		_, isKubernetesAnnotated := curr.Annotations["kubernetes"]
		_, isSwarmAnnotated := curr.Annotations["swarm"]

		if isKubernetesAnnotated && !isSwarmAnnotated && !hasKubernetes {
			return fmt.Errorf("%s is only supported on a Docker cli with kubernetes features enabled", cmd.CommandPath())
		}
		if isSwarmAnnotated && !isKubernetesAnnotated && hasKubernetes {
			return fmt.Errorf("%s is only supported on a Docker cli with swarm features enabled", cmd.CommandPath())
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
