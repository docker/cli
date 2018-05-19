package help

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/debug"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// SetFlagErrorFunc overrides the root command error function and checks every flag annotations against enabled features.
func SetFlagErrorFunc(dockerCli *command.DockerCli, cmd *cobra.Command, flags *pflag.FlagSet, opts *cliflags.ClientOptions) {
	// When invoking `docker stack --nonsense`, we need to make sure FlagErrorFunc return appropriate
	// output if the feature is not supported.
	// As above cli.SetupRootCommand(cmd) have already setup the FlagErrorFunc, we will add a pre-check before the FlagErrorFunc
	// is called.
	flagErrorFunc := cmd.FlagErrorFunc()
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		initializeDockerCli(dockerCli, flags, opts)
		if err := IsSupported(cmd, dockerCli); err != nil {
			return err
		}
		return flagErrorFunc(cmd, err)
	})
}

// SetHelpFunc overrides the root command help function and to hide every command depending annotations against enabled features.
func SetHelpFunc(dockerCli *command.DockerCli, cmd *cobra.Command, flags *pflag.FlagSet, opts *cliflags.ClientOptions) {
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(ccmd *cobra.Command, args []string) {
		initializeDockerCli(dockerCli, flags, opts)
		if err := IsSupported(ccmd, dockerCli); err != nil {
			ccmd.Println(err)
			return
		}

		hideUnsupportedFeatures(ccmd, dockerCli)
		defaultHelpFunc(ccmd, args)
	})
}

// SetValidateArgs validates all commands arguments depending annotations against enabled features.
func SetValidateArgs(dockerCli *command.DockerCli, cmd *cobra.Command, flags *pflag.FlagSet, opts *cliflags.ClientOptions) {
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
			if err := IsSupported(cmd, dockerCli); err != nil {
				return err
			}
			return cmdArgs(cmd, args)
		}
	})
}

// DockerPreRun initializes some global behaviours
func DockerPreRun(opts *cliflags.ClientOptions) {
	cliflags.SetLogLevel(opts.Common.LogLevel)

	if opts.ConfigDir != "" {
		cliconfig.SetDir(opts.ConfigDir)
	}

	if opts.Common.Debug {
		debug.Enable()
	}
}

func initializeDockerCli(dockerCli *command.DockerCli, flags *pflag.FlagSet, opts *cliflags.ClientOptions) {
	if dockerCli.Client() == nil { // when using --help, PersistentPreRun is not called, so initialization is needed.
		// flags must be the top-level command flags, not cmd.Flags()
		opts.Common.SetDefaultOptions(flags)
		DockerPreRun(opts)
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

// IsSupported checks if the command is supported depending enabled features.
func IsSupported(cmd *cobra.Command, details versionDetails) error {
	if err := areSubcommandsSupported(cmd, details); err != nil {
		return err
	}
	return areFlagsSupported(cmd, details)
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
