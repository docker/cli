package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/stack/options"
	composeLoader "github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

func newConfigCommand(dockerCli command.Cli) *cobra.Command {
	var opts options.Config

	cmd := &cobra.Command{
		Use:   "config [OPTIONS]",
		Short: "Outputs the final config file, after doing merges and interpolations",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return command.RunSwarm(dockerCli)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.StringSliceVarP(&opts.Composefiles, "compose-file", "c", []string{}, `Path to a Compose file, or "-" to read from stdin`)
	flags.BoolVar(&opts.SkipInterpolation, "skip-interpolation", false, "Skip interpolation and output only merged config")
	return cmd
}

// outputConfig returns the merged and interpolated config file
func outputConfig(configFiles composetypes.ConfigDetails, skipInterpolation bool) (string, error) {
	optsFunc := func(options *composeLoader.Options) {
		options.SkipInterpolation = skipInterpolation
	}
	config, err := composeLoader.Load(configFiles, optsFunc)
	if err != nil {
		return "", err
	}

	d, err := yaml.Marshal(&config)
	if err != nil {
		return "", err
	}
	return string(d), nil
}
