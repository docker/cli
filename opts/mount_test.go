package opts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/moby/moby/api/types/mount"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestMountOptString(t *testing.T) {
	m := MountOpt{
		values: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/home/path",
				Target: "/target",
			},
			{
				Type:   mount.TypeVolume,
				Source: "foo",
				Target: "/target/foo",
			},
		},
	}
	expected := "bind /home/path /target, volume foo /target/foo"
	assert.Check(t, is.Equal(expected, m.String()))
}

func TestMountRelative(t *testing.T) {
	for _, testcase := range []struct {
		name string
		path string
		bind string
	}{
		{
			name: "Current path",
			path: ".",
			bind: "type=bind,source=.,target=/target",
		},
		{
			name: "Current path with slash",
			path: "./",
			bind: "type=bind,source=./,target=/target",
		},
		{
			name: "Parent path with slash",
			path: "../",
			bind: "type=bind,source=../,target=/target",
		},
		{
			name: "Parent path",
			path: "..",
			bind: "type=bind,source=..,target=/target",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var m MountOpt
			assert.NilError(t, m.Set(testcase.bind))

			mounts := m.Value()
			assert.Assert(t, is.Len(mounts, 1))
			abs, err := filepath.Abs(testcase.path)
			assert.NilError(t, err)
			assert.Check(t, is.DeepEqual(mount.Mount{
				Type:   mount.TypeBind,
				Source: abs,
				Target: "/target",
			}, mounts[0]))
		})
	}
}

// TestMountOptSetBindNoErrorBind tests several aliases that should have
// the same result.
func TestMountOptSetBindNoErrorBind(t *testing.T) {
	for _, tc := range []string{
		"type=bind,target=/target,source=/source",
		"type=bind,src=/source,dst=/target",
		"type=bind,source=/source,dst=/target",
		"type=bind,src=/source,target=/target",
	} {
		t.Run(tc, func(t *testing.T) {
			var m MountOpt

			assert.NilError(t, m.Set(tc))

			mounts := m.Value()
			assert.Assert(t, is.Len(mounts, 1))
			assert.Check(t, is.DeepEqual(mount.Mount{
				Type:   mount.TypeBind,
				Source: "/source",
				Target: "/target",
			}, mounts[0]))
		})
	}
}

// TestMountOptSetVolumeNoError tests several aliases that should have
// the same result.
func TestMountOptSetVolumeNoError(t *testing.T) {
	for _, tc := range []string{
		"type=volume,target=/target,source=/source",
		"type=volume,src=/source,dst=/target",
		"type=volume,source=/source,dst=/target",
		"type=volume,src=/source,target=/target",
	} {
		t.Run(tc, func(t *testing.T) {
			var m MountOpt

			assert.NilError(t, m.Set(tc))

			mounts := m.Value()
			assert.Assert(t, is.Len(mounts, 1))
			assert.Check(t, is.DeepEqual(mount.Mount{
				Type:   mount.TypeVolume,
				Source: "/source",
				Target: "/target",
			}, mounts[0]))
		})
	}
}

// TestMountOptDefaultType ensures that a mount without the type defaults to a
// volume mount.
func TestMountOptDefaultType(t *testing.T) {
	var m MountOpt
	assert.NilError(t, m.Set("target=/target,source=/foo"))
	assert.Check(t, is.Equal(mount.TypeVolume, m.values[0].Type))
}

func TestMountOptSetErrorNoTarget(t *testing.T) {
	var m MountOpt
	assert.Error(t, m.Set("type=volume,source=/foo"), "target is required")
}

func TestMountOptSetErrorInvalidKey(t *testing.T) {
	var m MountOpt
	assert.Error(t, m.Set("type=volume,bogus=foo"), "unexpected key 'bogus' in 'bogus=foo'")
}

func TestMountOptSetErrorInvalidField(t *testing.T) {
	var m MountOpt
	assert.Error(t, m.Set("type=volume,bogus"), "invalid field 'bogus' must be a key=value pair")
}

func TestMountOptSetErrorInvalidReadOnly(t *testing.T) {
	var m MountOpt
	assert.Error(t, m.Set("type=volume,readonly=no"), "invalid value for readonly: no")
	assert.Error(t, m.Set("type=volume,readonly=invalid"), "invalid value for readonly: invalid")
}

func TestMountOptDefaultEnableReadOnly(t *testing.T) {
	var m MountOpt
	assert.NilError(t, m.Set("type=bind,target=/foo,source=/foo"))
	assert.Check(t, !m.values[0].ReadOnly)

	m = MountOpt{}
	assert.NilError(t, m.Set("type=bind,target=/foo,source=/foo,readonly"))
	assert.Check(t, m.values[0].ReadOnly)

	m = MountOpt{}
	assert.NilError(t, m.Set("type=bind,target=/foo,source=/foo,readonly=1"))
	assert.Check(t, m.values[0].ReadOnly)

	m = MountOpt{}
	assert.NilError(t, m.Set("type=bind,target=/foo,source=/foo,readonly=true"))
	assert.Check(t, m.values[0].ReadOnly)

	m = MountOpt{}
	assert.NilError(t, m.Set("type=bind,target=/foo,source=/foo,readonly=0"))
	assert.Check(t, !m.values[0].ReadOnly)
}

func TestMountOptVolumeNoCopy(t *testing.T) {
	var m MountOpt
	assert.NilError(t, m.Set("type=volume,target=/foo,volume-nocopy"))
	assert.Check(t, is.Equal("", m.values[0].Source))

	m = MountOpt{}
	assert.NilError(t, m.Set("type=volume,target=/foo,source=foo"))
	assert.Check(t, m.values[0].VolumeOptions == nil)

	m = MountOpt{}
	assert.NilError(t, m.Set("type=volume,target=/foo,source=foo,volume-nocopy=true"))
	assert.Check(t, m.values[0].VolumeOptions != nil)
	assert.Check(t, m.values[0].VolumeOptions.NoCopy)

	m = MountOpt{}
	assert.NilError(t, m.Set("type=volume,target=/foo,source=foo,volume-nocopy"))
	assert.Check(t, m.values[0].VolumeOptions != nil)
	assert.Check(t, m.values[0].VolumeOptions.NoCopy)

	m = MountOpt{}
	assert.NilError(t, m.Set("type=volume,target=/foo,source=foo,volume-nocopy=1"))
	assert.Check(t, m.values[0].VolumeOptions != nil)
	assert.Check(t, m.values[0].VolumeOptions.NoCopy)
}

func TestMountOptTypeConflict(t *testing.T) {
	var m MountOpt
	assert.ErrorContains(t, m.Set("type=bind,target=/foo,source=/foo,volume-nocopy=true"), "cannot mix")
	assert.ErrorContains(t, m.Set("type=volume,target=/foo,source=/foo,bind-propagation=rprivate"), "cannot mix")
}

func TestMountOptSetImageNoError(t *testing.T) {
	for _, tc := range []string{
		"type=image,source=foo,target=/target,image-subpath=/bar",
	} {
		var m MountOpt

		assert.NilError(t, m.Set(tc))

		mounts := m.Value()
		assert.Assert(t, is.Len(mounts, 1))
		assert.Check(t, is.DeepEqual(mount.Mount{
			Type:   mount.TypeImage,
			Source: "foo",
			Target: "/target",
			ImageOptions: &mount.ImageOptions{
				Subpath: "/bar",
			},
		}, mounts[0]))
	}
}

// TestMountOptSetTmpfsNoError tests several aliases that should have
// the same result.
func TestMountOptSetTmpfsNoError(t *testing.T) {
	for _, tc := range []string{
		"type=tmpfs,target=/target,tmpfs-size=1m,tmpfs-mode=0700",
		"type=tmpfs,target=/target,tmpfs-size=1MB,tmpfs-mode=700",
	} {
		t.Run(tc, func(t *testing.T) {
			var m MountOpt

			assert.NilError(t, m.Set(tc))

			mounts := m.Value()
			assert.Assert(t, is.Len(mounts, 1))
			assert.Check(t, is.DeepEqual(mount.Mount{
				Type:   mount.TypeTmpfs,
				Target: "/target",
				TmpfsOptions: &mount.TmpfsOptions{
					SizeBytes: 1024 * 1024, // not 1000 * 1000
					Mode:      os.FileMode(0o700),
				},
			}, mounts[0]))
		})
	}
}

func TestMountOptSetTmpfsError(t *testing.T) {
	var m MountOpt
	assert.ErrorContains(t, m.Set("type=tmpfs,target=/foo,tmpfs-size=foo"), "invalid value for tmpfs-size")
	assert.ErrorContains(t, m.Set("type=tmpfs,target=/foo,tmpfs-mode=foo"), "invalid value for tmpfs-mode")
	assert.ErrorContains(t, m.Set("type=tmpfs"), "target is required")
}

func TestMountOptSetBindRecursive(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		var m MountOpt
		assert.NilError(t, m.Set("type=bind,source=/foo,target=/bar,bind-recursive=enabled"))
		assert.Check(t, is.DeepEqual([]mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/foo",
				Target: "/bar",
			},
		}, m.Value()))
	})

	t.Run("disabled", func(t *testing.T) {
		var m MountOpt
		assert.NilError(t, m.Set("type=bind,source=/foo,target=/bar,bind-recursive=disabled"))
		assert.Check(t, is.DeepEqual([]mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/foo",
				Target: "/bar",
				BindOptions: &mount.BindOptions{
					NonRecursive: true,
				},
			},
		}, m.Value()))
	})

	t.Run("writable", func(t *testing.T) {
		var m MountOpt
		assert.Error(t, m.Set("type=bind,source=/foo,target=/bar,bind-recursive=writable"),
			"option 'bind-recursive=writable' requires 'readonly' to be specified in conjunction")
		assert.NilError(t, m.Set("type=bind,source=/foo,target=/bar,bind-recursive=writable,readonly"))
		assert.Check(t, is.DeepEqual([]mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   "/foo",
				Target:   "/bar",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					ReadOnlyNonRecursive: true,
				},
			},
		}, m.Value()))
	})

	t.Run("readonly", func(t *testing.T) {
		var m MountOpt
		assert.Error(t, m.Set("type=bind,source=/foo,target=/bar,bind-recursive=readonly"),
			"option 'bind-recursive=readonly' requires 'readonly' to be specified in conjunction")
		assert.Error(t, m.Set("type=bind,source=/foo,target=/bar,bind-recursive=readonly,readonly"),
			"option 'bind-recursive=readonly' requires 'bind-propagation=rprivate' to be specified in conjunction")
		assert.NilError(t, m.Set("type=bind,source=/foo,target=/bar,bind-recursive=readonly,readonly,bind-propagation=rprivate"))
		assert.Check(t, is.DeepEqual([]mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   "/foo",
				Target:   "/bar",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					ReadOnlyForceRecursive: true,
					Propagation:            mount.PropagationRPrivate,
				},
			},
		}, m.Value()))
	})
}
