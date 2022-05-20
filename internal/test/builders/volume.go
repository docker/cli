package builders

import "github.com/docker/docker/api/types/volume"

// Volume creates a volume with default values.
// Any number of volume function builder can be passed to augment it.
func Volume(builders ...func(volume *volume.Volume)) *volume.Volume {
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
func VolumeLabels(labels map[string]string) func(volume *volume.Volume) {
	return func(volume *volume.Volume) {
		volume.Labels = labels
	}
}

// VolumeName sets the volume labels
func VolumeName(name string) func(volume *volume.Volume) {
	return func(volume *volume.Volume) {
		volume.Name = name
	}
}

// VolumeDriver sets the volume driver
func VolumeDriver(name string) func(volume *volume.Volume) {
	return func(volume *volume.Volume) {
		volume.Driver = name
	}
}
