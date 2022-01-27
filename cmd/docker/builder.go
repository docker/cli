package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	builderDefaultPlugin     = "buildx"
	builderDefaultInstallMsg = `To install buildx, see https://docs.docker.com/go/buildx/`
	builderErrorMsg          = `%s: Required builder component %s is missing or broken.`
)

type builderError struct {
	warn    bool
	builder string
	err     error
}

func newBuilderError(warn bool, builder string, err error) error {
	return &builderError{
		warn:    warn,
		builder: builder,
		err:     err,
	}
}

func (e *builderError) Error() string {
	var errorMsg bytes.Buffer
	if e.warn {
		errorMsg.WriteString(fmt.Sprintf(builderErrorMsg, "WARNING", e.builder))
	} else {
		errorMsg.WriteString(fmt.Sprintf(builderErrorMsg, "ERROR", e.builder))
	}
	if e.builder == builderDefaultPlugin {
		errorMsg.WriteString(" ")
		errorMsg.WriteString(builderDefaultInstallMsg)
	}
	if pluginmanager.IsNotFound(e.err) {
		return errors.New(errorMsg.String()).Error()
	}
	return errors.Errorf("%v\n\n%s", e.err, errorMsg.String()).Error()
}

func (e *builderError) Unwrap() error {
	return e.err
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
			return fwargs, fwosargs, newBuilderError(false, builderAlias, perr)
		}
		// otherwise, display warning and continue
		_, _ = fmt.Fprintln(dockerCli.Err(), newBuilderError(true, builderAlias, perr).Error())
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
