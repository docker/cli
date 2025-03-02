package builders

import "github.com/docker/docker/api/types/volume"

// Volume creates a volume with default values.
// Any number of volume function builder can be passed to augment it.
func Volume(builders ...func(vol *volume.Volume)) *volume.Volume {
	vol := &volume.Volume{
		Name:       "volume",
		Driver:     "local",
		Mountpoint: "/data/volume",
		Scope:      "local",
	}

	for _, builder := range builders {
		builder(vol)
	}

	return vol
}

// VolumeLabels sets the volume labels
func VolumeLabels(labels map[string]string) func(vol *volume.Volume) {
	return func(vol *volume.Volume) {
		vol.Labels = labels
	}
}

// VolumeName sets the volume labels
func VolumeName(name string) func(vol *volume.Volume) {
	return func(vol *volume.Volume) {
		vol.Name = name
	}
}

// VolumeDriver sets the volume driver
func VolumeDriver(name string) func(vol *volume.Volume) {
	return func(vol *volume.Volume) {
		vol.Driver = name
	}
}
