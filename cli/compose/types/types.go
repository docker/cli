// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package types

import (
	"time"

	compose "github.com/compose-spec/compose-go/v2/types"
)

// UnsupportedProperties not yet supported by this implementation of the compose file
var UnsupportedProperties = map[string]string{
	"Build": "build",
	// "cgroupns_mode",
	"CgroupParent":  "cgroup_parent",
	"Devices":       "devices",
	"DomainName":    "domainname",
	"ExternalLinks": "external_links",
	"Ipc":           "ipc",
	"Links":         "links",
	"MacAddress":    "mac_address",
	"NetworkMode":   "network_mode",
	"Pid":           "pid",
	"Privileged":    "privileged",
	"Restart":       "restart",
	"SecurityOpt":   "security_opt",
	"ShmSize":       "shm_size",
	"UserNSMode":    "userns_mode",
}

// DeprecatedProperties that were removed from the v3 format, but their
// use should not impact the behaviour of the application.
var DeprecatedProperties = map[string]Pair[string, string]{
	"ContainerName": NewPair("container_name", "Setting the container name is not supported."),
	"Expose":        NewPair("expose", "Exposing ports is unnecessary - services on the same network can access each other's containers on any port."),
}

// ForbiddenProperties that are not supported in this implementation of the
// compose file.
var ForbiddenProperties = map[string]Pair[string, string]{
	"VolumeDriver": NewPair(
		"volume_driver",
		"Instead of setting the volume driver on the service, define a volume using the top-level `volumes` option and specify the driver there.",
	),
	"VolumesFrom": NewPair(
		"volumes_from",
		"To share a volume between services, define it using the top-level `volumes` option and reference it from each service that shares it using the service-level `volumes` option.",
	),
	"CPUQuota":     NewPair("cpu_quota", "Set resource limits using deploy.resources"),
	"CPUShares":    NewPair("cpu_shares", "Set resource limits using deploy.resources"),
	"CPUSet":       NewPair("cpuset", "Set resource limits using deploy.resources"),
	"MemLimit":     NewPair("mem_limit", "Set resource limits using deploy.resources"),
	"MemSwapLimit": NewPair("memswap_limit", "Set resource limits using deploy.resources"),
}

// ConvertDurationPtr converts a typedefined Duration pointer to a time.Duration pointer with the same value.
func ConvertDurationPtr(d *compose.Duration) *time.Duration {
	if d == nil {
		return nil
	}
	res := time.Duration(*d)
	return &res
}

// ClusterVolumeSpec defines all the configuration and options specific to a
// cluster (CSI) volume.
type ClusterVolumeSpec struct {
	Group                     string               `yaml:",omitempty" json:"group,omitempty"`
	AccessMode                *AccessMode          `mapstructure:"access_mode" yaml:"access_mode,omitempty" json:"access_mode,omitempty"`
	AccessibilityRequirements *TopologyRequirement `mapstructure:"accessibility_requirements" yaml:"accessibility_requirements,omitempty" json:"accessibility_requirements,omitempty"`
	CapacityRange             *CapacityRange       `mapstructure:"capacity_range" yaml:"capacity_range,omitempty" json:"capacity_range,omitempty"`

	Secrets []VolumeSecret `yaml:",omitempty" json:"secrets,omitempty"`

	Availability string `yaml:",omitempty" json:"availability,omitempty"`
}

// AccessMode defines the way a cluster volume is accessed by the tasks
type AccessMode struct {
	Scope   string `yaml:",omitempty" json:"scope,omitempty"`
	Sharing string `yaml:",omitempty" json:"sharing,omitempty"`

	MountVolume *MountVolume `mapstructure:"mount_volume" yaml:"mount_volume,omitempty" json:"mount_volume,omitempty"`
	BlockVolume *BlockVolume `mapstructure:"block_volume" yaml:"block_volume,omitempty" json:"block_volume,omitempty"`
}

// MountVolume defines options for using a volume as a Mount
type MountVolume struct {
	FsType     string   `mapstructure:"fs_type" yaml:"fs_type,omitempty" json:"fs_type,omitempty"`
	MountFlags []string `mapstructure:"mount_flags" yaml:"mount_flags,omitempty" json:"mount_flags,omitempty"`
}

// BlockVolume is deliberately empty
type BlockVolume struct{}

// TopologyRequirement defines the requirements for volume placement in the
// cluster.
type TopologyRequirement struct {
	Requisite []Topology `yaml:",omitempty" json:"requisite,omitempty"`
	Preferred []Topology `yaml:",omitempty" json:"preferred,omitempty"`
}

// Topology defines a particular topology group
type Topology struct {
	Segments compose.Mapping `yaml:",omitempty" json:"segments,omitempty"`
}

// CapacityRange defines the minimum and maximum size of a volume.
type CapacityRange struct {
	RequiredBytes compose.UnitBytes `mapstructure:"required_bytes" yaml:"required_bytes,omitempty" json:"required_bytes,omitempty"`
	LimitBytes    compose.UnitBytes `mapstructure:"limit_bytes" yaml:"limit_bytes,omitempty" json:"limit_bytes,omitempty"`
}

// VolumeSecret defines a secret that needs to be passed to the CSI plugin when
// using the volume.
type VolumeSecret struct {
	Key    string `yaml:",omitempty" json:"key,omitempty"`
	Secret string `yaml:",omitempty" json:"secret,omitempty"`
}

type Pair[T, U any] struct {
	key   T
	value U
}

func (p Pair[T, U]) Key() T {
	return p.key
}

func (p Pair[T, U]) Value() U {
	return p.value
}

func NewPair[T, U any](key T, value U) Pair[T, U] {
	return Pair[T, U]{key, value}
}
