package convert

import (
	"testing"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	clitypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/docker/api/types/mount"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestConvertVolumeToMountAnonymousVolume(t *testing.T) {
	config := composetypes.ServiceVolumeConfig{
		Type:   "volume",
		Target: "/foo/bar",
	}
	expected := mount.Mount{
		Type:   mount.TypeVolume,
		Target: "/foo/bar",
	}
	mnt, err := convertVolumeToMount(config, volumes{}, NewNamespace("foo"))
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}

func TestConvertVolumeToMountAnonymousBind(t *testing.T) {
	config := composetypes.ServiceVolumeConfig{
		Type:   "bind",
		Target: "/foo/bar",
		Bind: &composetypes.ServiceVolumeBind{
			Propagation: "slave",
		},
	}
	_, err := convertVolumeToMount(config, volumes{}, NewNamespace("foo"))
	assert.Error(t, err, "invalid bind source, source cannot be empty")
}

func TestConvertVolumeToMountUnapprovedType(t *testing.T) {
	config := composetypes.ServiceVolumeConfig{
		Type:   "foo",
		Target: "/foo/bar",
	}
	_, err := convertVolumeToMount(config, volumes{}, NewNamespace("foo"))
	assert.Error(t, err, "volume type must be volume, bind, tmpfs, npipe, or cluster")
}

func TestConvertVolumeToMountConflictingOptionsBindInVolume(t *testing.T) {
	namespace := NewNamespace("foo")

	config := composetypes.ServiceVolumeConfig{
		Type:   "volume",
		Source: "foo",
		Target: "/target",
		Bind: &composetypes.ServiceVolumeBind{
			Propagation: "slave",
		},
	}
	_, err := convertVolumeToMount(config, volumes{}, namespace)
	assert.Error(t, err, "bind options are incompatible with type volume")
}

func TestConvertVolumeToMountConflictingOptionsTmpfsInVolume(t *testing.T) {
	namespace := NewNamespace("foo")

	config := composetypes.ServiceVolumeConfig{
		Type:   "volume",
		Source: "foo",
		Target: "/target",
		Tmpfs: &composetypes.ServiceVolumeTmpfs{
			Size: 1000,
		},
	}
	_, err := convertVolumeToMount(config, volumes{}, namespace)
	assert.Error(t, err, "tmpfs options are incompatible with type volume")
}

func TestConvertVolumeToMountConflictingOptionsVolumeInBind(t *testing.T) {
	namespace := NewNamespace("foo")

	config := composetypes.ServiceVolumeConfig{
		Type:   "bind",
		Source: "/foo",
		Target: "/target",
		Volume: &composetypes.ServiceVolumeVolume{
			NoCopy: true,
		},
	}
	_, err := convertVolumeToMount(config, volumes{}, namespace)
	assert.Error(t, err, "volume options are incompatible with type bind")
}

func TestConvertVolumeToMountConflictingOptionsTmpfsInBind(t *testing.T) {
	namespace := NewNamespace("foo")

	config := composetypes.ServiceVolumeConfig{
		Type:   "bind",
		Source: "/foo",
		Target: "/target",
		Tmpfs: &composetypes.ServiceVolumeTmpfs{
			Size: 1000,
		},
	}
	_, err := convertVolumeToMount(config, volumes{}, namespace)
	assert.Error(t, err, "tmpfs options are incompatible with type bind")
}

func TestConvertVolumeToMountConflictingOptionsBindInTmpfs(t *testing.T) {
	namespace := NewNamespace("foo")

	config := composetypes.ServiceVolumeConfig{
		Type:   "tmpfs",
		Target: "/target",
		Bind: &composetypes.ServiceVolumeBind{
			Propagation: "slave",
		},
	}
	_, err := convertVolumeToMount(config, volumes{}, namespace)
	assert.Error(t, err, "bind options are incompatible with type tmpfs")
}

func TestConvertVolumeToMountConflictingOptionsVolumeInTmpfs(t *testing.T) {
	namespace := NewNamespace("foo")

	config := composetypes.ServiceVolumeConfig{
		Type:   "tmpfs",
		Target: "/target",
		Volume: &composetypes.ServiceVolumeVolume{
			NoCopy: true,
		},
	}
	_, err := convertVolumeToMount(config, volumes{}, namespace)
	assert.Error(t, err, "volume options are incompatible with type tmpfs")
}

func TestConvertVolumeToMountNamedVolume(t *testing.T) {
	stackVolumes := volumes{
		"normal": composetypes.VolumeConfig{
			Driver: "glusterfs",
			DriverOpts: map[string]string{
				"opt": "value",
			},
			Labels: map[string]string{
				"something": "labeled",
			},
		},
	}
	namespace := NewNamespace("foo")
	expected := mount.Mount{
		Type:     mount.TypeVolume,
		Source:   "foo_normal",
		Target:   "/foo",
		ReadOnly: true,
		VolumeOptions: &mount.VolumeOptions{
			Labels: map[string]string{
				LabelNamespace: "foo",
				"something":    "labeled",
			},
			DriverConfig: &mount.Driver{
				Name: "glusterfs",
				Options: map[string]string{
					"opt": "value",
				},
			},
			NoCopy: true,
		},
	}
	config := composetypes.ServiceVolumeConfig{
		Type:     "volume",
		Source:   "normal",
		Target:   "/foo",
		ReadOnly: true,
		Volume: &composetypes.ServiceVolumeVolume{
			NoCopy: true,
		},
	}
	mnt, err := convertVolumeToMount(config, stackVolumes, namespace)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}

func TestConvertVolumeToMountNamedVolumeWithNameCustomizd(t *testing.T) {
	stackVolumes := volumes{
		"normal": composetypes.VolumeConfig{
			Name:   "user_specified_name",
			Driver: "vsphere",
			DriverOpts: map[string]string{
				"opt": "value",
			},
			Labels: map[string]string{
				"something": "labeled",
			},
		},
	}
	namespace := NewNamespace("foo")
	expected := mount.Mount{
		Type:     mount.TypeVolume,
		Source:   "user_specified_name",
		Target:   "/foo",
		ReadOnly: true,
		VolumeOptions: &mount.VolumeOptions{
			Labels: map[string]string{
				LabelNamespace: "foo",
				"something":    "labeled",
			},
			DriverConfig: &mount.Driver{
				Name: "vsphere",
				Options: map[string]string{
					"opt": "value",
				},
			},
			NoCopy: true,
		},
	}
	config := composetypes.ServiceVolumeConfig{
		Type:     "volume",
		Source:   "normal",
		Target:   "/foo",
		ReadOnly: true,
		Volume: &composetypes.ServiceVolumeVolume{
			NoCopy: true,
		},
	}
	mnt, err := convertVolumeToMount(config, stackVolumes, namespace)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}

func TestConvertVolumeToMountNamedVolumeExternal(t *testing.T) {
	stackVolumes := volumes{
		"outside": composetypes.VolumeConfig{
			Name:     "special",
			External: true,
		},
	}
	namespace := NewNamespace("foo")
	expected := mount.Mount{
		Type:          mount.TypeVolume,
		Source:        "special",
		Target:        "/foo",
		VolumeOptions: &mount.VolumeOptions{NoCopy: false},
	}
	config := composetypes.ServiceVolumeConfig{
		Type:   "volume",
		Source: "outside",
		Target: "/foo",
	}
	mnt, err := convertVolumeToMount(config, stackVolumes, namespace)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}

func TestConvertVolumeToMountNamedVolumeExternalNoCopy(t *testing.T) {
	stackVolumes := volumes{
		"outside": composetypes.VolumeConfig{
			Name:     "special",
			External: true,
		},
	}
	namespace := NewNamespace("foo")
	expected := mount.Mount{
		Type:   mount.TypeVolume,
		Source: "special",
		Target: "/foo",
		VolumeOptions: &mount.VolumeOptions{
			NoCopy: true,
		},
	}
	config := composetypes.ServiceVolumeConfig{
		Type:   "volume",
		Source: "outside",
		Target: "/foo",
		Volume: &composetypes.ServiceVolumeVolume{
			NoCopy: true,
		},
	}
	mnt, err := convertVolumeToMount(config, stackVolumes, namespace)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}

func TestConvertVolumeToMountBind(t *testing.T) {
	stackVolumes := volumes{}
	namespace := NewNamespace("foo")
	expected := mount.Mount{
		Type:        mount.TypeBind,
		Source:      "/bar",
		Target:      "/foo",
		ReadOnly:    true,
		BindOptions: &mount.BindOptions{Propagation: mount.PropagationShared},
	}
	config := composetypes.ServiceVolumeConfig{
		Type:     "bind",
		Source:   "/bar",
		Target:   "/foo",
		ReadOnly: true,
		Bind:     &composetypes.ServiceVolumeBind{Propagation: "shared"},
	}
	mnt, err := convertVolumeToMount(config, stackVolumes, namespace)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}

func TestConvertVolumeToMountVolumeDoesNotExist(t *testing.T) {
	namespace := NewNamespace("foo")
	config := composetypes.ServiceVolumeConfig{
		Type:     "volume",
		Source:   "unknown",
		Target:   "/foo",
		ReadOnly: true,
	}
	_, err := convertVolumeToMount(config, volumes{}, namespace)
	assert.Error(t, err, "undefined volume \"unknown\"")
}

func TestConvertTmpfsToMountVolume(t *testing.T) {
	config := composetypes.ServiceVolumeConfig{
		Type:   "tmpfs",
		Target: "/foo/bar",
		Tmpfs: &composetypes.ServiceVolumeTmpfs{
			Size: 1000,
		},
	}
	expected := mount.Mount{
		Type:         mount.TypeTmpfs,
		Target:       "/foo/bar",
		TmpfsOptions: &mount.TmpfsOptions{SizeBytes: 1000},
	}
	mnt, err := convertVolumeToMount(config, volumes{}, NewNamespace("foo"))
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}

func TestConvertTmpfsToMountVolumeWithSource(t *testing.T) {
	config := composetypes.ServiceVolumeConfig{
		Type:   "tmpfs",
		Source: "/bar",
		Target: "/foo/bar",
		Tmpfs: &composetypes.ServiceVolumeTmpfs{
			Size: 1000,
		},
	}

	_, err := convertVolumeToMount(config, volumes{}, NewNamespace("foo"))
	assert.Error(t, err, "invalid tmpfs source, source must be empty")
}

func TestConvertVolumeToMountAnonymousNpipe(t *testing.T) {
	config := composetypes.ServiceVolumeConfig{
		Type:   "npipe",
		Source: `\\.\pipe\foo`,
		Target: `\\.\pipe\foo`,
	}
	expected := mount.Mount{
		Type:   mount.TypeNamedPipe,
		Source: `\\.\pipe\foo`,
		Target: `\\.\pipe\foo`,
	}
	mnt, err := convertVolumeToMount(config, volumes{}, NewNamespace("foo"))
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}

func TestConvertVolumeMountClusterName(t *testing.T) {
	stackVolumes := volumes{
		"my-csi": composetypes.VolumeConfig{
			Driver: "mycsidriver",
			Extensions: map[string]any{
				"x-cluster-spec": &clitypes.ClusterVolumeSpec{
					Group: "mygroup",
					AccessMode: &clitypes.AccessMode{
						Scope:       "single",
						Sharing:     "none",
						BlockVolume: &clitypes.BlockVolume{},
					},
					Availability: "active",
				},
			},
		},
	}

	config := composetypes.ServiceVolumeConfig{
		Type:   "cluster",
		Source: "my-csi",
		Target: "/srv",
	}

	expected := mount.Mount{
		Type:           mount.TypeCluster,
		Source:         "foo_my-csi",
		Target:         "/srv",
		ClusterOptions: &mount.ClusterOptions{},
	}

	mnt, err := convertVolumeToMount(config, stackVolumes, NewNamespace("foo"))
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}

func TestConvertVolumeMountClusterGroup(t *testing.T) {
	stackVolumes := volumes{
		"my-csi": composetypes.VolumeConfig{
			Driver: "mycsidriver",
			Extensions: map[string]any{
				"x-cluster-spec": &clitypes.ClusterVolumeSpec{
					Group: "mygroup",
					AccessMode: &clitypes.AccessMode{
						Scope:       "single",
						Sharing:     "none",
						BlockVolume: &clitypes.BlockVolume{},
					},
					Availability: "active",
				},
			},
		},
	}

	config := composetypes.ServiceVolumeConfig{
		Type:   "cluster",
		Source: "group:mygroup",
		Target: "/srv",
	}

	expected := mount.Mount{
		Type:           mount.TypeCluster,
		Source:         "group:mygroup",
		Target:         "/srv",
		ClusterOptions: &mount.ClusterOptions{},
	}

	mnt, err := convertVolumeToMount(config, stackVolumes, NewNamespace("foo"))
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, mnt))
}
