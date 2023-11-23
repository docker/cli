package convert

import (
	"strings"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"
)

type volumes map[string]composetypes.VolumeConfig

// Volumes from compose-file types to engine api types
func Volumes(serviceVolumes []composetypes.ServiceVolumeConfig, stackVolumes volumes, namespace Namespace) ([]mount.Mount, error) {
	mounts := make([]mount.Mount, 0, len(serviceVolumes))
	for _, volumeConfig := range serviceVolumes {
		mnt, err := convertVolumeToMount(volumeConfig, stackVolumes, namespace)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mnt)
	}
	return mounts, nil
}

func createMountFromVolume(volume composetypes.ServiceVolumeConfig) mount.Mount {
	return mount.Mount{
		Type:        mount.Type(volume.Type),
		Target:      volume.Target,
		ReadOnly:    volume.ReadOnly,
		Source:      volume.Source,
		Consistency: mount.Consistency(volume.Consistency),
	}
}

func handleVolumeToMount(
	volume composetypes.ServiceVolumeConfig,
	stackVolumes volumes,
	namespace Namespace,
) (mount.Mount, error) {
	result := createMountFromVolume(volume)

	if volume.Tmpfs != nil {
		return mount.Mount{}, errors.New("tmpfs options are incompatible with type volume")
	}
	if volume.Bind != nil {
		return mount.Mount{}, errors.New("bind options are incompatible with type volume")
	}
	if volume.Cluster != nil {
		return mount.Mount{}, errors.New("cluster options are incompatible with type volume")
	}
	// Anonymous volumes
	if volume.Source == "" {
		return result, nil
	}

	stackVolume, exists := stackVolumes[volume.Source]
	if !exists {
		return mount.Mount{}, errors.Errorf("undefined volume %q", volume.Source)
	}

	result.Source = namespace.Scope(volume.Source)
	result.VolumeOptions = &mount.VolumeOptions{}

	if volume.Volume != nil {
		result.VolumeOptions.NoCopy = volume.Volume.NoCopy
	}

	if stackVolume.Name != "" {
		result.Source = stackVolume.Name
	}

	// External named volumes
	if stackVolume.External.External {
		return result, nil
	}

	result.VolumeOptions.Labels = AddStackLabel(namespace, stackVolume.Labels)
	if stackVolume.Driver != "" || stackVolume.DriverOpts != nil {
		result.VolumeOptions.DriverConfig = &mount.Driver{
			Name:    stackVolume.Driver,
			Options: stackVolume.DriverOpts,
		}
	}

	return result, nil
}

func handleBindToMount(volume composetypes.ServiceVolumeConfig) (mount.Mount, error) {
	result := createMountFromVolume(volume)

	if volume.Source == "" {
		return mount.Mount{}, errors.New("invalid bind source, source cannot be empty")
	}
	if volume.Volume != nil {
		return mount.Mount{}, errors.New("volume options are incompatible with type bind")
	}
	if volume.Tmpfs != nil {
		return mount.Mount{}, errors.New("tmpfs options are incompatible with type bind")
	}
	if volume.Cluster != nil {
		return mount.Mount{}, errors.New("cluster options are incompatible with type bind")
	}
	if volume.Bind != nil {
		result.BindOptions = &mount.BindOptions{
			Propagation: mount.Propagation(volume.Bind.Propagation),
		}
	}
	return result, nil
}

func handleTmpfsToMount(volume composetypes.ServiceVolumeConfig) (mount.Mount, error) {
	result := createMountFromVolume(volume)

	if volume.Source != "" {
		return mount.Mount{}, errors.New("invalid tmpfs source, source must be empty")
	}
	if volume.Bind != nil {
		return mount.Mount{}, errors.New("bind options are incompatible with type tmpfs")
	}
	if volume.Volume != nil {
		return mount.Mount{}, errors.New("volume options are incompatible with type tmpfs")
	}
	if volume.Cluster != nil {
		return mount.Mount{}, errors.New("cluster options are incompatible with type tmpfs")
	}
	if volume.Tmpfs != nil {
		result.TmpfsOptions = &mount.TmpfsOptions{
			SizeBytes: volume.Tmpfs.Size,
		}
	}
	return result, nil
}

func handleNpipeToMount(volume composetypes.ServiceVolumeConfig) (mount.Mount, error) {
	result := createMountFromVolume(volume)

	if volume.Source == "" {
		return mount.Mount{}, errors.New("invalid npipe source, source cannot be empty")
	}
	if volume.Volume != nil {
		return mount.Mount{}, errors.New("volume options are incompatible with type npipe")
	}
	if volume.Tmpfs != nil {
		return mount.Mount{}, errors.New("tmpfs options are incompatible with type npipe")
	}
	if volume.Bind != nil {
		result.BindOptions = &mount.BindOptions{
			Propagation: mount.Propagation(volume.Bind.Propagation),
		}
	}
	return result, nil
}

func handleClusterToMount(
	volume composetypes.ServiceVolumeConfig,
	stackVolumes volumes,
	namespace Namespace,
) (mount.Mount, error) {
	if volume.Source == "" {
		return mount.Mount{}, errors.New("invalid cluster source, source cannot be empty")
	}
	if volume.Tmpfs != nil {
		return mount.Mount{}, errors.New("tmpfs options are incompatible with type cluster")
	}
	if volume.Bind != nil {
		return mount.Mount{}, errors.New("bind options are incompatible with type cluster")
	}
	if volume.Volume != nil {
		return mount.Mount{}, errors.New("volume options are incompatible with type cluster")
	}

	result := createMountFromVolume(volume)
	result.ClusterOptions = &mount.ClusterOptions{}

	if !strings.HasPrefix(volume.Source, "group:") {
		// if the volume is a cluster volume and the source is a volumegroup, we
		// will ignore checking to see if such a volume is defined. the volume
		// group isn't namespaced, and there's no simple way to indicate that
		// external volumes with a given group exist.
		stackVolume, exists := stackVolumes[volume.Source]
		if !exists {
			return mount.Mount{}, errors.Errorf("undefined volume %q", volume.Source)
		}

		// if the volume is not specified with a group source, we may namespace
		// the name, if one is not otherwise specified.
		if stackVolume.Name != "" {
			result.Source = stackVolume.Name
		} else {
			result.Source = namespace.Scope(volume.Source)
		}
	}

	return result, nil
}

func convertVolumeToMount(
	volume composetypes.ServiceVolumeConfig,
	stackVolumes volumes,
	namespace Namespace,
) (mount.Mount, error) {
	switch volume.Type {
	case "volume", "":
		return handleVolumeToMount(volume, stackVolumes, namespace)
	case "bind":
		return handleBindToMount(volume)
	case "tmpfs":
		return handleTmpfsToMount(volume)
	case "npipe":
		return handleNpipeToMount(volume)
	case "cluster":
		return handleClusterToMount(volume, stackVolumes, namespace)
	}
	return mount.Mount{}, errors.New("volume type must be volume, bind, tmpfs, npipe, or cluster")
}
