// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package volume

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type createOptions struct {
	name       string
	driver     string
	driverOpts opts.MapOpts
	labels     opts.ListOpts

	// options for cluster volumes only
	cluster           bool
	group             string
	scope             string
	sharing           string
	availability      string
	secrets           opts.MapOpts
	requiredBytes     opts.MemBytes
	limitBytes        opts.MemBytes
	accessType        string
	requisiteTopology opts.ListOpts
	preferredTopology opts.ListOpts
}

func newCreateCommand(dockerCLI command.Cli) *cobra.Command {
	options := createOptions{
		driverOpts:        *opts.NewMapOpts(nil, nil),
		labels:            opts.NewListOpts(opts.ValidateLabel),
		secrets:           *opts.NewMapOpts(nil, nil),
		requisiteTopology: opts.NewListOpts(nil),
		preferredTopology: opts.NewListOpts(nil),
	}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] [VOLUME]",
		Short: "Create a volume",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if options.name != "" {
					return errors.New("conflicting options: cannot specify a volume-name through both --name and as a positional arg")
				}
				options.name = args[0]
			}
			options.cluster = hasClusterVolumeOptionSet(cmd.Flags())
			return runCreate(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.StringVarP(&options.driver, "driver", "d", "local", "Specify volume driver name")
	flags.StringVar(&options.name, "name", "", "Specify volume name")
	flags.Lookup("name").Hidden = true
	flags.VarP(&options.driverOpts, "opt", "o", "Set driver specific options")
	flags.Var(&options.labels, "label", "Set metadata for a volume")

	// flags for cluster volumes only
	flags.StringVar(&options.group, "group", "", "Cluster Volume group (cluster volumes)")
	flags.SetAnnotation("group", "version", []string{"1.42"})
	flags.SetAnnotation("group", "swarm", []string{"manager"})
	flags.StringVar(&options.scope, "scope", "single", `Cluster Volume access scope ("single", "multi")`)
	flags.SetAnnotation("scope", "version", []string{"1.42"})
	flags.SetAnnotation("scope", "swarm", []string{"manager"})
	flags.StringVar(&options.sharing, "sharing", "none", `Cluster Volume access sharing ("none", "readonly", "onewriter", "all")`)
	flags.SetAnnotation("sharing", "version", []string{"1.42"})
	flags.SetAnnotation("sharing", "swarm", []string{"manager"})
	flags.StringVar(&options.availability, "availability", "active", `Cluster Volume availability ("active", "pause", "drain")`)
	flags.SetAnnotation("availability", "version", []string{"1.42"})
	flags.SetAnnotation("availability", "swarm", []string{"manager"})
	flags.StringVar(&options.accessType, "type", "block", `Cluster Volume access type ("mount", "block")`)
	flags.SetAnnotation("type", "version", []string{"1.42"})
	flags.SetAnnotation("type", "swarm", []string{"manager"})
	flags.Var(&options.secrets, "secret", "Cluster Volume secrets")
	flags.SetAnnotation("secret", "version", []string{"1.42"})
	flags.SetAnnotation("secret", "swarm", []string{"manager"})
	flags.Var(&options.limitBytes, "limit-bytes", "Minimum size of the Cluster Volume in bytes")
	flags.SetAnnotation("limit-bytes", "version", []string{"1.42"})
	flags.SetAnnotation("limit-bytes", "swarm", []string{"manager"})
	flags.Var(&options.requiredBytes, "required-bytes", "Maximum size of the Cluster Volume in bytes")
	flags.SetAnnotation("required-bytes", "version", []string{"1.42"})
	flags.SetAnnotation("required-bytes", "swarm", []string{"manager"})
	flags.Var(&options.requisiteTopology, "topology-required", "A topology that the Cluster Volume must be accessible from")
	flags.SetAnnotation("topology-required", "version", []string{"1.42"})
	flags.SetAnnotation("topology-required", "swarm", []string{"manager"})
	flags.Var(&options.preferredTopology, "topology-preferred", "A topology that the Cluster Volume would be preferred in")
	flags.SetAnnotation("topology-preferred", "version", []string{"1.42"})
	flags.SetAnnotation("topology-preferred", "swarm", []string{"manager"})

	return cmd
}

// hasClusterVolumeOptionSet returns true if any of the cluster-specific
// options are set.
func hasClusterVolumeOptionSet(flags *pflag.FlagSet) bool {
	return flags.Changed("group") || flags.Changed("scope") ||
		flags.Changed("sharing") || flags.Changed("availability") ||
		flags.Changed("type") || flags.Changed("secrets") ||
		flags.Changed("limit-bytes") || flags.Changed("required-bytes")
}

func runCreate(ctx context.Context, dockerCli command.Cli, options createOptions) error {
	res, err := dockerCli.Client().VolumeCreate(ctx, client.VolumeCreateOptions{
		Driver:            options.driver,
		DriverOpts:        options.driverOpts.GetAll(),
		Name:              options.name,
		Labels:            opts.ConvertKVStringsToMap(options.labels.GetSlice()),
		ClusterVolumeSpec: clusterVolumeSpec(options),
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), res.Volume.Name)
	return nil
}

func clusterVolumeSpec(options createOptions) *volume.ClusterVolumeSpec {
	if !options.cluster {
		return nil
	}

	var secrets []volume.Secret
	for key, secret := range options.secrets.GetAll() {
		secrets = append(secrets, volume.Secret{Key: key, Secret: secret})
	}
	slices.SortFunc(secrets, func(a, b volume.Secret) int {
		return cmp.Compare(a.Key, b.Key)
	})

	accessMode := &volume.AccessMode{
		Scope:   volume.Scope(options.scope),
		Sharing: volume.SharingMode(options.sharing),
	}
	switch options.accessType {
	case "mount":
		accessMode.MountVolume = &volume.TypeMount{}
	case "block":
		accessMode.BlockVolume = &volume.TypeBlock{}
	}

	return &volume.ClusterVolumeSpec{
		Group:      options.group,
		AccessMode: accessMode,
		AccessibilityRequirements: &volume.TopologyRequirement{
			Requisite: parseTopologies(options.requisiteTopology.GetSlice()),
			Preferred: parseTopologies(options.preferredTopology.GetSlice()),
		},
		CapacityRange: &volume.CapacityRange{
			RequiredBytes: max(options.requiredBytes.Value(), 0),
			LimitBytes:    max(options.limitBytes.Value(), 0),
		},
		Secrets:      secrets,
		Availability: volume.Availability(options.availability),
	}
}

func parseTopologies(values []string) []volume.Topology {
	topologies := make([]volume.Topology, 0, len(values))
	for _, top := range values {
		// TODO(dperny): validate topology syntax
		topologies = append(topologies, volume.Topology{
			Segments: opts.ConvertKVStringsToMap(strings.Split(top, ",")),
		})
	}
	return topologies
}
