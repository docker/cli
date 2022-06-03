package volume

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
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

func newCreateCommand(dockerCli command.Cli) *cobra.Command {
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
					return errors.Errorf("conflicting options: either specify --name or provide positional arg, not both")
				}
				options.name = args[0]
			}
			options.cluster = hasClusterVolumeOptionSet(cmd.Flags())
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

	// flags for cluster volumes only
	flags.StringVar(&options.group, "group", "", "Cluster Volume group (cluster volumes)")
	flags.SetAnnotation("group", "version", []string{"1.42"})
	flags.SetAnnotation("group", "swarm", []string{"manager"})
	flags.StringVar(&options.scope, "scope", "single", `Cluster Volume access scope ("single"|"multi")`)
	flags.SetAnnotation("scope", "version", []string{"1.42"})
	flags.SetAnnotation("scope", "swarm", []string{"manager"})
	flags.StringVar(&options.sharing, "sharing", "none", `Cluster Volume access sharing ("none"|"readonly"|"onewriter"|"all")`)
	flags.SetAnnotation("sharing", "version", []string{"1.42"})
	flags.SetAnnotation("sharing", "swarm", []string{"manager"})
	flags.StringVar(&options.availability, "availability", "active", `Cluster Volume availability ("active"|"pause"|"drain")`)
	flags.SetAnnotation("availability", "version", []string{"1.42"})
	flags.SetAnnotation("availability", "swarm", []string{"manager"})
	flags.StringVar(&options.accessType, "type", "block", `Cluster Volume access type ("mount"|"block")`)
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

func runCreate(dockerCli command.Cli, options createOptions) error {
	volOpts := volume.CreateOptions{
		Driver:     options.driver,
		DriverOpts: options.driverOpts.GetAll(),
		Name:       options.name,
		Labels:     opts.ConvertKVStringsToMap(options.labels.GetAll()),
	}
	if options.cluster {
		volOpts.ClusterVolumeSpec = &volume.ClusterVolumeSpec{
			Group: options.group,
			AccessMode: &volume.AccessMode{
				Scope:   volume.Scope(options.scope),
				Sharing: volume.SharingMode(options.sharing),
			},
			Availability: volume.Availability(options.availability),
		}

		if options.accessType == "mount" {
			volOpts.ClusterVolumeSpec.AccessMode.MountVolume = &volume.TypeMount{}
		} else if options.accessType == "block" {
			volOpts.ClusterVolumeSpec.AccessMode.BlockVolume = &volume.TypeBlock{}
		}

		vcr := &volume.CapacityRange{}
		if r := options.requiredBytes.Value(); r >= 0 {
			vcr.RequiredBytes = r
		}

		if l := options.limitBytes.Value(); l >= 0 {
			vcr.LimitBytes = l
		}
		volOpts.ClusterVolumeSpec.CapacityRange = vcr

		for key, secret := range options.secrets.GetAll() {
			volOpts.ClusterVolumeSpec.Secrets = append(
				volOpts.ClusterVolumeSpec.Secrets,
				volume.Secret{
					Key:    key,
					Secret: secret,
				},
			)
		}
		sort.SliceStable(volOpts.ClusterVolumeSpec.Secrets, func(i, j int) bool {
			return volOpts.ClusterVolumeSpec.Secrets[i].Key < volOpts.ClusterVolumeSpec.Secrets[j].Key
		})

		// TODO(dperny): ignore if no topology specified
		topology := &volume.TopologyRequirement{}
		for _, top := range options.requisiteTopology.GetAll() {
			// each topology takes the form segment=value,segment=value
			// comma-separated list of equal separated maps
			segments := map[string]string{}
			for _, segment := range strings.Split(top, ",") {
				parts := strings.SplitN(segment, "=", 2)
				// TODO(dperny): validate topology syntax
				segments[parts[0]] = parts[1]
			}
			topology.Requisite = append(
				topology.Requisite,
				volume.Topology{Segments: segments},
			)
		}

		for _, top := range options.preferredTopology.GetAll() {
			// each topology takes the form segment=value,segment=value
			// comma-separated list of equal separated maps
			segments := map[string]string{}
			for _, segment := range strings.Split(top, ",") {
				parts := strings.SplitN(segment, "=", 2)
				// TODO(dperny): validate topology syntax
				segments[parts[0]] = parts[1]
			}

			topology.Preferred = append(
				topology.Preferred,
				volume.Topology{Segments: segments},
			)
		}

		volOpts.ClusterVolumeSpec.AccessibilityRequirements = topology
	}

	vol, err := dockerCli.Client().VolumeCreate(context.Background(), volOpts)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), vol.Name)
	return nil
}
