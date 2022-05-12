package volume

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type createOptions struct {
	name       string
	driver     string
	driverOpts opts.MapOpts
	labels     opts.ListOpts
}

func newCreateCommand(dockerCli command.Cli) *cobra.Command {
	options := createOptions{
		driverOpts: *opts.NewMapOpts(nil, nil),
		labels:     opts.NewListOpts(opts.ValidateLabel),
	}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] [VOLUME]",
		Short: "Create a volume",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if options.name != "" {
					return errors.Errorf("conflicting options: either specify --name or provide positional arg, not both")
				}
				options.name = args[0]
			}
			return runCreate(dockerCli, options)
		},
		ValidArgsFunction: completion.NoComplete,
	}
	flags := cmd.Flags()
	flags.StringVarP(&options.driver, "driver", "d", "local", "Specify volume driver name")
	flags.StringVar(&options.name, "name", "", "Specify volume name")
	flags.Lookup("name").Hidden = true
	flags.VarP(&options.driverOpts, "opt", "o", "Set driver specific options")
	flags.Var(&options.labels, "label", "Set metadata for a volume")

	return cmd
}

func runCreate(dockerCli command.Cli, options createOptions) error {
	vol, err := dockerCli.Client().VolumeCreate(context.Background(), volume.CreateOptions{
		Driver:     options.driver,
		DriverOpts: options.driverOpts.GetAll(),
		Name:       options.name,
		Labels:     opts.ConvertKVStringsToMap(options.labels.GetAll()),
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), vol.Name)
	return nil
}
