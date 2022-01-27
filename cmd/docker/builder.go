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
	return fmt.Errorf("%w\n\n%s", err, errorMsg)
}

func processBuilder(dockerCli command.Cli, cmd *cobra.Command, args, osargs []string) ([]string, []string, error) {
	// check DOCKER_BUILDKIT env var is present and
	// if not assume we want to use a builder
	var enforcedBuilder bool
	if v, ok := os.LookupEnv("DOCKER_BUILDKIT"); ok {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return args, osargs, errors.Wrap(err, "DOCKER_BUILDKIT environment variable expects boolean value")
		}
		if !enabled {
			return args, osargs, nil
		}
		enforcedBuilder = true
	}

	// if a builder alias is defined, use it instead
	// of the default one
	isAlias := false
	builderAlias := builderDefaultPlugin
	aliasMap := dockerCli.ConfigFile().Aliases
	if v, ok := aliasMap[keyBuilderAlias]; ok {
		isAlias = true
		builderAlias = v
	}

	// wcow build command must use the legacy builder for buildx
	// if not opt-in through a builder alias
	if !isAlias && dockerCli.ServerInfo().OSType == "windows" {
		return args, osargs, nil
	}

	// are we using a cmd that should be forwarded to the builder?
	fwargs, fwosargs, forwarded := forwardBuilder(builderAlias, args, osargs)
	if !forwarded {
		return args, osargs, nil
	}

	// check plugin is available if cmd forwarded
	plugin, perr := pluginmanager.GetPlugin(builderAlias, dockerCli, cmd.Root())
	if perr == nil && plugin != nil {
		perr = plugin.Err
	}
	if perr != nil {
		// if builder enforced with DOCKER_BUILDKIT=1, cmd fails if plugin missing or broken
		if enforcedBuilder {
			return fwargs, fwosargs, newBuilderError(false, perr)
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
