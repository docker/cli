package stack

import (
	"fmt"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/loader"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func newConfigCommand(dockerCli command.Cli) *cobra.Command {
	var opts options.Deploy

	cmd := &cobra.Command{
		Use:   "config [OPTIONS]",
		Short: "Parse and validate a stack configuration",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := loader.LoadComposefile(dockerCli, opts)
			if err != nil {
				return err
			}
			marshal, err := yaml.Marshal(config)
			if err != nil {
				return err
			}
			fmt.Println(string(marshal))
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringSliceVarP(&opts.Composefiles, "compose-file", "c", []string{}, `Path to a Compose file, or "-" to read from stdin`)
	flags.SetAnnotation("compose-file", "version", []string{"1.25"})
	flags.BoolVar(&opts.SendRegistryAuth, "with-registry-auth", false, "Send registry authentication details to Swarm agents")
	flags.SetAnnotation("with-registry-auth", "swarm", nil)
	flags.BoolVar(&opts.Prune, "prune", false, "Prune services that are no longer referenced")
	flags.SetAnnotation("prune", "version", []string{"1.27"})
	flags.SetAnnotation("prune", "swarm", nil)
	flags.StringVar(&opts.ResolveImage, "resolve-image", swarm.ResolveImageAlways,
		`Query the registry to resolve image digest and supported platforms ("`+swarm.ResolveImageAlways+`"|"`+swarm.ResolveImageChanged+`"|"`+swarm.ResolveImageNever+`")`)
	flags.SetAnnotation("resolve-image", "version", []string{"1.30"})
	flags.SetAnnotation("resolve-image", "swarm", nil)
	return cmd
}
