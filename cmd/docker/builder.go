package main

import (
	"fmt"
	"os"
	"strconv"

	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	builderDefaultPlugin = "buildx"
	buildxMissingWarning = `DEPRECATED: The legacy builder is deprecated and will be removed in a future release.
            Install the buildx component to build images with BuildKit:
            https://docs.docker.com/go/buildx/
`

	buildxMissingError = `ERROR: BuildKit is enabled but the buildx component is missing or broken.
       Install the buildx component to build images with BuildKit:
       https://docs.docker.com/go/buildx/
`
)

func newBuilderError(warn bool, err error) error {
	var errorMsg string
	if warn {
		errorMsg = buildxMissingWarning
	} else {
		errorMsg = buildxMissingError
	}
	if pluginmanager.IsNotFound(err) {
		return errors.New(errorMsg)
	}
	if err != nil {
		return fmt.Errorf("%w\n\n%s", err, errorMsg)
	}
	return fmt.Errorf("%s", errorMsg)
}

func processBuilder(dockerCli command.Cli, cmd *cobra.Command, args, osargs []string) ([]string, []string, error) {
	var useLegacy, useBuilder bool

	// check DOCKER_BUILDKIT env var is present and
	// if not assume we want to use the builder component
	if v, ok := os.LookupEnv("DOCKER_BUILDKIT"); ok {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return args, osargs, errors.Wrap(err, "DOCKER_BUILDKIT environment variable expects boolean value")
		}
		if !enabled {
			useLegacy = true
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
		builderAlias = v
	}

	// is this a build that should be forwarded to the builder?
	fwargs, fwosargs, forwarded := forwardBuilder(builderAlias, args, osargs)
	if !forwarded {
		return args, osargs, nil
	}

	// wcow build command must use the legacy builder
	// if not opt-in through a builder component
	if !useBuilder && dockerCli.ServerInfo().OSType == "windows" {
		return args, osargs, nil
	}

	if useLegacy {
		// display warning if not wcow and continue
		if dockerCli.ServerInfo().OSType != "windows" {
			_, _ = fmt.Fprintln(dockerCli.Err(), newBuilderError(true, nil))
		}
		return args, osargs, nil
	}

	// check plugin is available if cmd forwarded
	plugin, perr := pluginmanager.GetPlugin(builderAlias, dockerCli, cmd.Root())
	if perr == nil && plugin != nil {
		perr = plugin.Err
	}
	if perr != nil {
		// if builder enforced with DOCKER_BUILDKIT=1, cmd must fail if plugin missing or broken
		if useBuilder {
			return args, osargs, newBuilderError(false, perr)
		}
		// otherwise, display warning and continue
		_, _ = fmt.Fprintln(dockerCli.Err(), newBuilderError(true, perr))
		return args, osargs, nil
	}

	return fwargs, fwosargs, nil
}

func forwardBuilder(alias string, args, osargs []string) ([]string, []string, bool) {
	aliases := [][2][]string{
		{
			{"builder"},
			{alias},
		},
		{
			{"build"},
			{alias, "build"},
		},
		{
			{"image", "build"},
			{alias, "build"},
		},
	}
	for _, al := range aliases {
		if fwargs, changed := command.StringSliceReplaceAt(args, al[0], al[1], 0); changed {
			fwosargs, _ := command.StringSliceReplaceAt(osargs, al[0], al[1], -1)
			return fwargs, fwosargs, true
		}
	}
	return args, osargs, false
}
