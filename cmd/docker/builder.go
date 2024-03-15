package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	builderDefaultPlugin = "buildx"
	buildxMissingWarning = `DEPRECATED: The legacy builder is deprecated and will be removed in a future release.
            Install the buildx component to build images with BuildKit:
            https://docs.docker.com/go/buildx/`

	buildkitDisabledWarning = `DEPRECATED: The legacy builder is deprecated and will be removed in a future release.
            BuildKit is currently disabled; enable it by removing the DOCKER_BUILDKIT=0
            environment-variable.`

	buildxMissingError = `ERROR: BuildKit is enabled but the buildx component is missing or broken.
       Install the buildx component to build images with BuildKit:
       https://docs.docker.com/go/buildx/`
)

func newBuilderError(errorMsg string, pluginLoadErr error) error {
	if pluginmanager.IsNotFound(pluginLoadErr) {
		return errors.New(errorMsg)
	}
	if pluginLoadErr != nil {
		return fmt.Errorf("%w\n\n%s", pluginLoadErr, errorMsg)
	}
	return errors.New(errorMsg)
}

//nolint:gocyclo
func processBuilder(dockerCli command.Cli, cmd *cobra.Command, args, osargs []string) ([]string, []string, []string, error) {
	var buildKitDisabled, useBuilder, useAlias bool
	var envs []string

	// check DOCKER_BUILDKIT env var is not empty
	// if it is assume we want to use the builder component
	if v := os.Getenv("DOCKER_BUILDKIT"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return args, osargs, nil, errors.Wrap(err, "DOCKER_BUILDKIT environment variable expects boolean value")
		}
		if !enabled {
			buildKitDisabled = true
		} else {
			useBuilder = true
		}
	}

	// if a builder alias is defined, use it instead
	// of the default one
	builderAlias := builderDefaultPlugin
	aliasMap := dockerCli.ConfigFile().Aliases
	if v, ok := aliasMap[keyBuilderAlias]; ok {
		useBuilder = true
		useAlias = true
		builderAlias = v
	}

	// is this a build that should be forwarded to the builder?
	fwargs, fwosargs, fwcmdpath, forwarded := forwardBuilder(builderAlias, args, osargs)
	if !forwarded {
		return args, osargs, nil, nil
	}

	// wcow build command must use the legacy builder
	// if not opt-in through a builder component
	if !useBuilder && dockerCli.ServerInfo().OSType == "windows" {
		return args, osargs, nil, nil
	}

	if buildKitDisabled {
		// display warning if not wcow and continue
		if dockerCli.ServerInfo().OSType != "windows" {
			_, _ = fmt.Fprintf(dockerCli.Err(), "%s\n\n", buildkitDisabledWarning)
		}
		return args, osargs, nil, nil
	}

	// check plugin is available if cmd forwarded
	plugin, perr := pluginmanager.GetPlugin(builderAlias, dockerCli, cmd.Root())
	if perr == nil && plugin != nil {
		perr = plugin.Err
	}
	if perr != nil {
		// if builder is enforced with DOCKER_BUILDKIT=1, cmd must fail
		// if the plugin is missing or broken.
		if useBuilder {
			return args, osargs, nil, newBuilderError(buildxMissingError, perr)
		}
		// otherwise, display warning and continue
		_, _ = fmt.Fprintf(dockerCli.Err(), "%s\n\n", newBuilderError(buildxMissingWarning, perr))
		return args, osargs, nil, nil
	}

	// If build subcommand is forwarded, user would expect "docker build" to
	// always create a local docker image (default context builder). This is
	// for better backward compatibility in case where a user could switch to
	// a docker container builder with "docker buildx --use foo" which does
	// not --load by default. Also makes sure that an arbitrary builder name
	// is not being set in the command line or in the environment before
	// setting the default context and keep "buildx install" behavior if being
	// set (builder alias).
	if forwarded && !useAlias && !hasBuilderName(args, os.Environ()) {
		envs = append([]string{"BUILDX_BUILDER=" + dockerCli.CurrentContext()}, envs...)
	}

	// overwrite the command path for this plugin using the alias name.
	cmd.Annotations[pluginmanager.CommandAnnotationPluginCommandPath] = strings.Join(append([]string{cmd.CommandPath()}, fwcmdpath...), " ")

	return fwargs, fwosargs, envs, nil
}

func forwardBuilder(alias string, args, osargs []string) ([]string, []string, []string, bool) {
	aliases := [][3][]string{
		{
			{"builder"},
			{alias},
			{"builder"},
		},
		{
			{"build"},
			{alias, "build"},
			{},
		},
		{
			{"image", "build"},
			{alias, "build"},
			{"image"},
		},
	}
	for _, al := range aliases {
		if fwargs, changed := command.StringSliceReplaceAt(args, al[0], al[1], 0); changed {
			fwosargs, _ := command.StringSliceReplaceAt(osargs, al[0], al[1], -1)
			fwcmdpath := al[2]
			return fwargs, fwosargs, fwcmdpath, true
		}
	}
	return args, osargs, nil, false
}

// hasBuilderName checks if a builder name is defined in args or env vars
func hasBuilderName(args []string, envs []string) bool {
	var builder string
	flagset := pflag.NewFlagSet("buildx", pflag.ContinueOnError)
	flagset.Usage = func() {}
	flagset.SetOutput(io.Discard)
	flagset.StringVar(&builder, "builder", "", "")
	_ = flagset.Parse(args)
	if builder != "" {
		return true
	}
	for _, e := range envs {
		if strings.HasPrefix(e, "BUILDX_BUILDER=") && e != "BUILDX_BUILDER=" {
			return true
		}
	}
	return false
}
