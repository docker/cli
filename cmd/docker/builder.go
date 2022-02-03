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
	builderDefaultInstallMsg = `To install buildx, see
       https://docs.docker.com/go/buildx. You can also fallback to the
       legacy builder by setting DOCKER_BUILDKIT=0`

	builderErrorMsg = "ERROR: Missing builder component %s."
)

type builderError struct {
	builder string
	err     error
}

func newBuilderError(builder string, err error) error {
	return &builderError{
		builder: builder,
		err:     err,
	}
}

func (e *builderError) Error() string {
	var errorMsg bytes.Buffer
	errorMsg.WriteString(fmt.Sprintf(builderErrorMsg, e.builder))
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

func processBuilder(dockerCli command.Cli, cmd *cobra.Command, args, osArgs []string) ([]string, []string, error) {
	// check DOCKER_BUILDKIT env var is present and
	// if not assume we want to use a builder
	if v, ok := os.LookupEnv("DOCKER_BUILDKIT"); ok {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return args, osArgs, errors.Wrap(err, "DOCKER_BUILDKIT environment variable expects boolean value")
		}
		if !enabled {
			return args, osArgs, nil
		}
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
		return args, osArgs, nil
	}

	// builder aliases
	aliases := [][2][]string{
		{
			{"builder"},
			{builderAlias},
		},
		{
			{"build"},
			{builderAlias, "build"},
		},
		{
			{"image", "build"},
			{builderAlias, "build"},
		},
	}

	// are we using a cmd that should be forwarded to the builder?
	var forwarded bool
	for _, al := range aliases {
		var didChange bool
		args, didChange = command.StringSliceReplaceAt(args, al[0], al[1], 0)
		if didChange {
			forwarded = true
			osArgs, _ = command.StringSliceReplaceAt(osArgs, al[0], al[1], -1)
			break
		}
	}
	if !forwarded {
		return args, osArgs, nil
	}

	// check plugin is available if cmd forwarded
	plugin, perr := pluginmanager.GetPlugin(builderAlias, dockerCli, cmd.Root())
	if perr == nil && plugin != nil {
		perr = plugin.Err
	}
	if perr != nil {
		return args, osArgs, newBuilderError(builderAlias, perr)
	}

	return args, osArgs, nil
}
