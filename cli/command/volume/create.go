package volume

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type createOptions struct {
	name       string
	driver     string
	driverOpts opts.MapOpts
	labels     opts.ListOpts

	// options for cluster voluems only
	group         string
	scope         string
	sharing       string
	availability  string
	secrets       opts.MapOpts
	requiredBytes opts.MemBytes
	limitBytes    opts.MemBytes
	accessType    string
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
					return errors.Errorf("Conflicting options: either specify --name or provide positional arg, not both\n")
				}
				options.name = args[0]
			}
			return runCreate(dockerCli, options)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&options.driver, "driver", "d", "local", "Specify volume driver name")
	flags.StringVar(&options.name, "name", "", "Specify volume name")
	flags.Lookup("name").Hidden = true
	flags.VarP(&options.driverOpts, "opt", "o", "Set driver specific options")
	flags.Var(&options.labels, "label", "Set metadata for a volume")

	// flags for cluster volumes only
	flags.StringVarP(&options.group, "group", "g", "", "Cluster Volume group (cluster volumes)")
	flags.StringVar(&options.scope, "scope", "single", `Cluster Volume access scope ("single"|"multi")`)
	flags.StringVar(&options.sharing, "sharing", "none", `Cluster Volume access sharing ("none"|"readonly"|"onewriter"|"all")`)
	flags.StringVar(&options.availability, "availability", "active", `Cluster Volume availability ("active"|"pause"|"drain")`)
	flags.StringVar(&options.accessType, "type", "block", `Cluster Volume access type ("mount"|"block")`)
	flags.Var(&options.secrets, "secret", "Cluster Volume secrets")
	flags.Var(&options.requiredBytes, "limit-bytes", "Minimum size of the Cluster Volume in bytes (default 0 for undefined)")
	flags.Var(&options.limitBytes, "required-bytes", "Maximum size of the Cluster Volume in bytes (default 0 for undefined)")

	return cmd
}

func runCreate(dockerCli command.Cli, options createOptions) error {
	client := dockerCli.Client()

	volReq := volumetypes.VolumeCreateBody{
		Driver:     options.driver,
		DriverOpts: options.driverOpts.GetAll(),
		Name:       options.name,
		Labels:     opts.ConvertKVStringsToMap(options.labels.GetAll()),
	}

	vol, err := client.VolumeCreate(context.Background(), volReq)
	if err != nil {
		return err
	}

	fmt.Fprintf(dockerCli.Out(), "%s\n", vol.Name)
	return nil
}
