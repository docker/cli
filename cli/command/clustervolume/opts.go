package clustervolume

import (
	"github.com/docker/cli/opts"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"

	"github.com/spf13/pflag"
)

const (
	flagLabel         = "label"
	flagGroup         = "group"
	flagScope         = "scope"
	flagSharing       = "sharing"
	flagAvailability  = "availability"
	flagDriver        = "driver"
	flagDriverOpts    = "driver-opts"
	flagSecrets       = "secret"
	flagRequiredBytes = "required-bytes"
	flagLimitBytes    = "limit-bytes"
	flagType          = "type"
	// TODO(dperny): add these
	// flagFsType = "fstype"
	// flagMountFlags = "mount-flag"
)

func addVolumeFlags(flags *pflag.FlagSet, opts *clusterVolumeOptions) {
	flags.VarP(&opts.labels, flagLabel, "l", "Cluster Volume labels")
	flags.StringVarP(&opts.group, flagGroup, "g", "", "Volume group")
	flags.StringVar(&opts.scope, flagScope, "single", `Volume access scope ("single"|"multi")`)
	flags.StringVar(&opts.sharing, flagSharing, "none", `Volume access sharing ("none"|"readonly"|"onewriter"|"all")`)
	flags.StringVar(&opts.availability, flagAvailability, "active", `Volume availability ("active"|"pause"|"drain")`)
	flags.StringVar(&opts.driver, flagDriver, "", "Volume driver")
	flags.Var(&opts.driverOpts, flagDriverOpts, "Volume driver options")
	flags.Var(&opts.secrets, flagSecrets, "Volume secrets")
	flags.Uint64Var(&opts.requiredBytes, flagRequiredBytes, 0, "Minimum size of the volume in bytes (default 0 for undefined)")
	flags.Uint64Var(&opts.limitBytes, flagLimitBytes, 0, "Maximum size of the volume in bytes (default 0 for undefined)")
	flags.StringVar(&opts.accessType, flagType, "block", `Volume access type ("mount"|"block")`)
}

// TODO(dperny): figure out how to specify topology constraints...
type clusterVolumeOptions struct {
	name   string
	labels opts.ListOpts

	group        string
	scope        string
	sharing      string
	availability string

	driver     string
	driverOpts opts.MapOpts

	secrets opts.MapOpts

	// TODO(dperny): allow specifying human-readable bytes (like MB/GB/etc)
	requiredBytes uint64
	limitBytes    uint64

	accessType string
}

func newClusterVolumeOptions() *clusterVolumeOptions {
	return &clusterVolumeOptions{
		labels:     opts.NewListOpts(opts.ValidateLabel),
		driverOpts: *opts.NewMapOpts(nil, nil),
		secrets:    *opts.NewMapOpts(nil, nil),
	}
}

func (vopts clusterVolumeOptions) ToVolumeSpec() swarm.VolumeSpec {
	spec := swarm.VolumeSpec{
		Annotations: swarm.Annotations{
			Name:   vopts.name,
			Labels: opts.ConvertKVStringsToMap(vopts.labels.GetAll()),
		},
		Group: vopts.group,
		AccessMode: &swarm.VolumeAccessMode{
			Scope:   swarm.VolumeScope(vopts.scope),
			Sharing: swarm.VolumeSharing(vopts.sharing),
		},
		Driver: &mount.Driver{
			Name:    vopts.driver,
			Options: vopts.driverOpts.GetAll(),
		},
		Availability: swarm.VolumeAvailability(vopts.availability),
		CapacityRange: &swarm.CapacityRange{
			RequiredBytes: vopts.requiredBytes,
			LimitBytes:    vopts.limitBytes,
		},
	}

	switch vopts.accessType {
	case "mount":
		spec.AccessMode.MountVolume = &swarm.VolumeTypeMount{}
	case "block":
		spec.AccessMode.BlockVolume = &swarm.VolumeTypeBlock{}
	}

	var secrets []swarm.VolumeSecret
	for key, secret := range vopts.secrets.GetAll() {
		secrets = append(secrets, swarm.VolumeSecret{
			Key:    key,
			Secret: secret,
		})
	}
	spec.Secrets = secrets

	return spec
}
