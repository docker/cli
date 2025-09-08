package stack

import (
	"bytes"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	composeLoader "github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configOptions holds docker stack config options
type configOptions struct {
	composeFiles      []string
	skipInterpolation bool
}

func newConfigCommand(dockerCLI command.Cli) *cobra.Command {
	var opts configOptions

	cmd := &cobra.Command{
		Use:   "config [OPTIONS]",
		Short: "Outputs the final config file, after doing merges and interpolations",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configDetails, err := getConfigDetails(opts.composeFiles, dockerCLI.In())
			if err != nil {
				return err
			}

			cfg, err := outputConfig(configDetails, opts.skipInterpolation)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(dockerCLI.Out(), "%s", cfg)
			return err
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringSliceVarP(&opts.composeFiles, "compose-file", "c", []string{}, `Path to a Compose file, or "-" to read from stdin`)
	flags.BoolVar(&opts.skipInterpolation, "skip-interpolation", false, "Skip interpolation and output only merged config")
	return cmd
}

// outputConfig returns the merged and interpolated config file
func outputConfig(configFiles composetypes.ConfigDetails, skipInterpolation bool) (string, error) {
	optsFunc := func(opts *composeLoader.Options) {
		opts.SkipInterpolation = skipInterpolation
	}
	config, err := composeLoader.Load(configFiles, optsFunc)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err = enc.Encode(&config)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
