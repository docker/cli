package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/docker/cli/cli"
	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/metadata"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	contextdocker "github.com/docker/cli/cli/context/docker"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// processCloud resolves the global --cloud[=NAME] option after configuration is
// loaded, but before telemetry and endpoint initialization.
// The option appears in help and completion only when the selected provider is
// installed, valid, and declares support for the resolver contract.
// The provider defaults to the offload plugin; features.cloud in the effective
// Docker config.json can override the plugin name, not its executable path.
// Explicit use also requires the provider's capability declaration.
// Discovery for visibility does not provision a context or connect to a daemon.
//
// # Provider contract
//
// The provider must include "Features": {"cloud-context-resolver": true} in its
// docker-cli-plugin-metadata response to declare support for this contract.
// Older plugins without this declaration leave the flag hidden and cannot be
// used for resolution.
// The CLI uses normal plugin discovery and invokes:
//
//	docker-<provider> --config=<dir> <provider> __resolve-context -- <name>
//
// Bare --cloud and --cloud= pass default as the target name.
// The effective config directory is also passed as DOCKER_CONFIG.
// The provider must provision into that context store without changing the saved
// current context or recursively forwarding --cloud.
// It inherits the environment, receives no interactive stdin, and sends progress
// to stderr.
// Stdout must contain exactly one JSON object:
//
//	{"DOCKER_CONTEXT":"provisioned-context"}
//
// The returned context must exist, must not be the virtual default context, and
// must have a Docker endpoint with a nonempty host.
// Resolver failures abort command execution without falling back to another
// context.
// Cancellation terminates the resolver process; provisioning subprocesses and
// resource cleanup remain the provider's responsibility.
// Downstream plugins receive --context=<resolved-name> instead of the global
// --cloud option.
// The provider implementation ships separately and must support this operation
// before enabling the integration.
func processCloud(ctx context.Context, dockerCli *command.DockerCli, cmd *cobra.Command, args, osArgs []string) ([]string, error) {
	if !cmd.Flags().Changed("cloud") {
		return osArgs, nil
	}
	if cmd.Flags().Changed("context") || cmd.Flags().Changed("host") {
		return osArgs, errors.New("conflicting options: cannot specify --cloud together with --context or --host")
	}

	help, err := cloudHelpRequest(cmd, args)
	if err != nil {
		return osArgs, err
	}

	var contextName string
	if !help {
		name, _ := cmd.Flags().GetString("cloud")
		if name == "" {
			name = "default"
		}

		contextName, err = resolveCloudContext(ctx, dockerCli, cmd, name)
		if err != nil {
			return osArgs, fmt.Errorf("--cloud: %w", err)
		}
		if err := cmd.Flags().Set("context", contextName); err != nil {
			return osArgs, err
		}
	}

	return cloudPluginArgs(cmd, osArgs, contextName)
}

// cloudResolverFeature is the plugin metadata feature that declares support for
// the __resolve-context contract.
const cloudResolverFeature = "cloud-context-resolver"

func cloudProvider(dockerCli config.Provider, rootCmd *cobra.Command) (*pluginmanager.Plugin, error) {
	provider := dockerCli.ConfigFile().Features["cloud"]
	if provider == "" {
		provider = "offload"
	}

	plugin, err := pluginmanager.GetPlugin(provider, dockerCli, rootCmd)
	if err != nil {
		return nil, fmt.Errorf("cloud resolver plugin %q unavailable: %w", provider, err)
	}
	if plugin.Err != nil {
		return nil, fmt.Errorf("invalid cloud resolver plugin %q: %w", provider, plugin.Err)
	}

	if supported, _ := plugin.Features[cloudResolverFeature].(bool); !supported {
		return nil, fmt.Errorf("plugin %q does not support cloud context resolution", provider)
	}

	return plugin, nil
}

// updateCloudFlagVisibility runs only for help and completion, avoiding plugin
// discovery on ordinary invocations that do not use --cloud.
func updateCloudFlagVisibility(dockerCli config.Provider, rootCmd *cobra.Command) {
	flag := rootCmd.Flags().Lookup("cloud")
	if flag == nil {
		return
	}
	_, err := cloudProvider(dockerCli, rootCmd)
	flag.Hidden = err != nil
}

func resolveCloudContext(ctx context.Context, dockerCli *command.DockerCli, rootCmd *cobra.Command, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	plugin, err := cloudProvider(dockerCli, rootCmd)
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, plugin.Path, "--config="+config.Dir(), plugin.Name, "__resolve-context", "--", name) // #nosec G204 -- executable validated through CLI plugin discovery
	cmd.Env = append(os.Environ(), config.EnvOverrideConfigDir+"="+config.Dir(), metadata.ReexecEnvvar+"="+os.Args[0])
	cmd.Stderr = dockerCli.Err()

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("cloud resolver failed: %w", err)
	}

	var response struct {
		DockerContext string `json:"DOCKER_CONTEXT"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return "", fmt.Errorf("invalid cloud resolver response: %w", err)
	}
	response.DockerContext = strings.TrimSpace(response.DockerContext)
	if response.DockerContext == "" || response.DockerContext == command.DefaultContextName {
		return "", errors.New("cloud resolver must return a non-default DOCKER_CONTEXT")
	}

	// Do not allow a missing context or endpoint to fall back to the local engine.
	meta, err := dockerCli.ContextStore().GetMetadata(response.DockerContext)
	if err != nil {
		return "", fmt.Errorf("loading resolved context %q: %w", response.DockerContext, err)
	}

	endpoint, err := contextdocker.EndpointFromContext(meta)
	if err != nil {
		return "", fmt.Errorf("invalid resolved context %q: %w", response.DockerContext, err)
	}
	if endpoint.Host == "" {
		return "", fmt.Errorf("resolved context %q has no Docker endpoint host", response.DockerContext)
	}

	return response.DockerContext, nil
}

// cloudHelpRequest avoids provisioning for help and shell completion.
// Parse known command flags rather than scanning argv: --help may be an option
// value or an argument to a container, not a request for Docker help.
// Unknown flags and missing flag values return a usage error, so a mistyped
// invocation fails without provisioning or falling back to the local engine.
// Flag values are validated only by the command's own parse, after resolution.
func cloudHelpRequest(rootCmd *cobra.Command, args []string) (bool, error) {
	help, _ := rootCmd.PersistentFlags().GetBool("help")
	version, _ := rootCmd.Flags().GetBool("version")
	if help || version || len(args) == 0 {
		return true, nil
	}

	switch args[0] {
	case "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return true, nil
	}

	cmd, remaining, err := rootCmd.Find(args)
	if err != nil || cmd == rootCmd || pluginmanager.IsPluginCommand(cmd) {
		// Plugin flags are opaque. Only recognize help immediately after the
		// plugin name; deeper help is available through "docker help PLUGIN".
		return len(args) > 1 && isHelpFlag(args[1]), nil
	}

	// "build", "bake", "builder", and "image build" are builtin commands that
	// processAliases may still rewrite into a call to the builder plugin
	// (buildx) after cloud resolution runs, so their flag set here is only the
	// legacy builder's and does not reflect what will actually parse the
	// remaining args. Treat them like a plugin command above: only recognize
	// help immediately after the resolved command path.
	if _, _, _, forwarded := forwardBuilder(builderDefaultPlugin, args, args); forwarded {
		return len(remaining) > 0 && isHelpFlag(remaining[0]), nil
	}

	cmd.InitDefaultHelpFlag()
	flags := cmd.Flags()
	flags.AddFlagSet(cmd.PersistentFlags())
	flags.AddFlagSet(cmd.InheritedFlags())

	err = flags.ParseAll(remaining, func(flag *pflag.Flag, value string) error {
		if flag.Name == "help" {
			var err error
			if help, err = strconv.ParseBool(value); err != nil {
				return fmt.Errorf("invalid argument %q for \"--help\" flag: %w", value, err)
			}
		}
		return nil
	})
	if err != nil {
		// Format the usage error directly. The root command's FlagErrorFunc
		// first checks whether the daemon supports the command, which would
		// connect to the engine selected before --cloud is resolved.
		return false, cli.FlagErrorFunc(cmd, err)
	}

	return help, nil
}

// isHelpFlag reports whether arg requests help, in any of the forms pflag
// accepts for a boolean flag.
func isHelpFlag(arg string) bool {
	return arg == "--help" || arg == "-h" || arg == "--help=true"
}

// cloudPluginArgs only rewrites the global prefix, leaving subcommand arguments
// untouched. Parsing also distinguishes --cloud from another flag's value.
func cloudPluginArgs(cmd *cobra.Command, osArgs []string, contextName string) ([]string, error) {
	flags := pflag.NewFlagSet(cmd.Name(), pflag.ContinueOnError)
	flags.SetInterspersed(false)
	flags.AddFlagSet(cmd.Flags())
	flags.AddFlagSet(cmd.PersistentFlags())

	result := []string{osArgs[0]}
	if contextName != "" {
		result = append(result, "--context="+contextName)
	}

	if err := flags.ParseAll(osArgs[1:], func(flag *pflag.Flag, value string) error {
		if flag.Name != "cloud" {
			result = append(result, "--"+flag.Name+"="+value)
		}
		return nil
	}); err != nil {
		return osArgs, err
	}

	if flags.ArgsLenAtDash() >= 0 {
		result = append(result, "--")
	}

	return append(result, flags.Args()...), nil
}
