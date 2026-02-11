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

// TestMountOptSourceTargetAliases tests several aliases that should have
// the same result.
func TestMountOptSourceTargetAliases(t *testing.T) {
	for _, tc := range []string{
		"type=bind,src=/source,dst=/target",
		"type=bind,source=/source,target=/target",
		"type=bind,source=/source,destination=/target",
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

// TestMountOptDefaultType ensures that a mount without the type defaults to a
// volume mount.
func TestMountOptDefaultType(t *testing.T) {
	var m MountOpt
	assert.NilError(t, m.Set("target=/target,source=/foo"))
	assert.Check(t, is.Equal(mount.TypeVolume, m.values[0].Type))
}

func TestMountOptErrors(t *testing.T) {
	tests := []struct {
		doc, value, expErr string
	}{
		{
			doc:    "empty value",
			expErr: "value is empty",
		},
		{
			doc:    "invalid key=value",
			value:  "type=volume,target=/foo,bogus=foo",
			expErr: "unknown option 'bogus' in 'bogus=foo'",
		},
		{
			doc:    "invalid key with leading whitespace",
			value:  "type=volume, src=/foo,target=/foo",
			expErr: "invalid option 'src' in ' src=/foo': option should not have whitespace",
		},
		{
			doc:    "invalid key with trailing whitespace",
			value:  "type=volume,src =/foo,target=/foo",
			expErr: "invalid option 'src' in 'src =/foo': option should not have whitespace",
		},
		{
			doc:    "invalid value is empty",
			value:  "type=volume,src=,target=/foo",
			expErr: "invalid value for 'src': value is empty",
		},
		{
			doc:    "invalid value with leading whitespace",
			value:  "type=volume,src= /foo,target=/foo",
			expErr: "invalid value for 'src' in 'src= /foo': value should not have whitespace",
		},
		{
			doc:    "invalid value with trailing whitespace",
			value:  "type=volume,src=/foo ,target=/foo",
			expErr: "invalid value for 'src' in 'src=/foo ': value should not have whitespace",
		},
		{
			doc:    "missing value",
			value:  "type=volume,target=/foo,bogus",
			expErr: "invalid field 'bogus' must be a key=value pair",
		},
		{
			doc:    "invalid tmpfs-size",
			value:  "type=tmpfs,target=/foo,tmpfs-size=foo",
			expErr: "invalid value for tmpfs-size: foo",
		},
		{
			doc:    "invalid tmpfs-mode",
			value:  "type=tmpfs,target=/foo,tmpfs-mode=foo",
			expErr: "invalid value for tmpfs-mode: foo",
		},
		{
			doc:    "mixed bind and volume",
			value:  "type=volume,target=/foo,source=/foo,bind-propagation=rprivate",
			expErr: "cannot mix 'bind-*' options with mount type 'volume'",
		},
		{
			doc:    "mixed volume and bind",
			value:  "type=bind,target=/foo,source=/foo,volume-nocopy=true",
			expErr: "cannot mix 'volume-*' options with mount type 'bind'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			err := (&MountOpt{}).Set(tc.value)
			assert.Error(t, err, tc.expErr)
		})
	}
}

func TestMountOptReadOnly(t *testing.T) {
	tests := []struct {
		value  string
		exp    bool
		expErr string
	}{
		{value: "", exp: false},
		{value: "readonly", exp: true},
		{value: "readonly=", expErr: `invalid value for 'readonly': value is empty`},
		{value: "readonly= true", expErr: `invalid value for 'readonly' in 'readonly= true': value should not have whitespace`},
		{value: "readonly=no", expErr: `invalid value for 'readonly': invalid boolean value ("no"): must be one of "true", "1", "false", or "0" (default "true")`},
		{value: "readonly=1", exp: true},
		{value: "readonly=true", exp: true},
		{value: "readonly=0", exp: false},
		{value: "readonly=false", exp: false},
		{value: "ro", exp: true},
		{value: "ro=1", exp: true},
		{value: "ro=true", exp: true},
		{value: "ro=0", exp: false},
		{value: "ro=false", exp: false},
	}

	for _, tc := range tests {
		name := tc.value
		if name == "" {
			name = "not set"
		}
		t.Run(name, func(t *testing.T) {
			val := "type=bind,target=/foo,source=/foo"
			if tc.value != "" {
				val += "," + tc.value
			}
			var m MountOpt
			err := m.Set(val)
			if tc.expErr != "" {
				assert.Error(t, err, tc.expErr)
				return
			}
			assert.NilError(t, err)
			assert.Check(t, is.Equal(m.values[0].ReadOnly, tc.exp))
		})
	}
}

func TestMountOptVolumeNoCopy(t *testing.T) {
	tests := []struct {
		value  string
		exp    bool
		expErr string
	}{
		{value: "", exp: false},
		{value: "volume-nocopy", exp: true},
		{value: "volume-nocopy=", expErr: `invalid value for 'volume-nocopy': value is empty`},
		{value: "volume-nocopy= true", expErr: `invalid value for 'volume-nocopy' in 'volume-nocopy= true': value should not have whitespace`},
		{value: "volume-nocopy=no", expErr: `invalid value for 'volume-nocopy': invalid boolean value ("no"): must be one of "true", "1", "false", or "0" (default "true")`},
		{value: "volume-nocopy=1", exp: true},
		{value: "volume-nocopy=true", exp: true},
		{value: "volume-nocopy=0", exp: false},
		{value: "volume-nocopy=false", exp: false},
	}

	for _, tc := range tests {
		name := tc.value
		if name == "" {
			name = "not set"
		}
		t.Run(name, func(t *testing.T) {
			val := "type=volume,target=/foo,source=foo"
			if tc.value != "" {
				val += "," + tc.value
			}
			var m MountOpt
			err := m.Set(val)
			if tc.expErr != "" {
				assert.Error(t, err, tc.expErr)
				return
			}
			assert.NilError(t, err)
			if tc.value == "" {
				assert.Check(t, is.Nil(m.values[0].VolumeOptions))
			} else {
				assert.Check(t, m.values[0].VolumeOptions != nil)
				assert.Check(t, is.Equal(m.values[0].VolumeOptions.NoCopy, tc.exp))
			}
		})
	}
}

func TestMountOptVolumeOptions(t *testing.T) {
	tests := []struct {
		doc   string
		value string
		exp   mount.Mount
	}{
		{
			doc:   "volume-label single",
			value: `type=volume,target=/foo,volume-label=foo=foo-value`,
			exp: mount.Mount{
				Type:   mount.TypeVolume,
				Target: "/foo",
				VolumeOptions: &mount.VolumeOptions{
					Labels: map[string]string{
						"foo": "foo-value",
					},
				},
			},
		},
		{
			doc:   "volume-label multiple",
			value: `type=volume,target=/foo,volume-label=foo=foo-value,volume-label=bar=bar-value`,
			exp: mount.Mount{
				Type:   mount.TypeVolume,
				Target: "/foo",
				VolumeOptions: &mount.VolumeOptions{
					Labels: map[string]string{
						"foo": "foo-value",
						"bar": "bar-value",
					},
				},
			},
		},
		{
			doc:   "volume-label empty values",
			value: `type=volume,target=/foo,volume-label=foo=,volume-label=bar`,
			exp: mount.Mount{
				Type:   mount.TypeVolume,
				Target: "/foo",
				VolumeOptions: &mount.VolumeOptions{
					Labels: map[string]string{
						"foo": "",
						"bar": "",
					},
				},
			},
		},
		{
			// TODO(thaJeztah): this should probably be an error instead
			doc:   "volume-label empty key",
			value: `type=volume,target=/foo,volume-label==foo-value`,
			exp: mount.Mount{
				Type:          mount.TypeVolume,
				Target:        "/foo",
				VolumeOptions: &mount.VolumeOptions{},
			},
		},
		{
			doc:   "volume-driver",
			value: `type=volume,target=/foo,volume-driver=my-driver`,
			exp: mount.Mount{
				Type:   mount.TypeVolume,
				Target: "/foo",
				VolumeOptions: &mount.VolumeOptions{
					DriverConfig: &mount.Driver{
						Name: "my-driver",
					},
				},
			},
		},
		{
			doc:   "volume-opt single",
			value: `type=volume,target=/foo,volume-opt=foo=foo-value`,
			exp: mount.Mount{
				Type:   mount.TypeVolume,
				Target: "/foo",
				VolumeOptions: &mount.VolumeOptions{
					DriverConfig: &mount.Driver{
						Options: map[string]string{
							"foo": "foo-value",
						},
					},
				},
			},
		},
		{
			doc:   "volume-opt multiple",
			value: `type=volume,target=/foo,volume-opt=foo=foo-value,volume-opt=bar=bar-value`,
			exp: mount.Mount{
				Type:   mount.TypeVolume,
				Target: "/foo",
				VolumeOptions: &mount.VolumeOptions{
					DriverConfig: &mount.Driver{
						Options: map[string]string{
							"foo": "foo-value",
							"bar": "bar-value",
						},
					},
				},
			},
		},
		{
			doc:   "volume-opt empty values",
			value: `type=volume,target=/foo,volume-opt=foo=,volume-opt=bar`,
			exp: mount.Mount{
				Type:   mount.TypeVolume,
				Target: "/foo",
				VolumeOptions: &mount.VolumeOptions{
					DriverConfig: &mount.Driver{
						Options: map[string]string{
							"foo": "",
							"bar": "",
						},
					},
				},
			},
		},
		{
			// TODO(thaJeztah): this should probably be an error instead
			doc:   "volume-opt empty key",
			value: `type=volume,target=/foo,volume-opt==foo-value`,
			exp: mount.Mount{
				Type:   mount.TypeVolume,
				Target: "/foo",
				VolumeOptions: &mount.VolumeOptions{
					DriverConfig: &mount.Driver{},
				},
			},
		},
		{
			doc:   "volume-label and volume-opt",
			value: `type=volume,volume-driver=my-driver,target=/foo,volume-label=foo=foo-value,volume-label=empty=,volume-opt=foo=foo-value,volume-opt=empty=`,
			exp: mount.Mount{
				Type:   mount.TypeVolume,
				Target: "/foo",
				VolumeOptions: &mount.VolumeOptions{
					Labels: map[string]string{
						"foo":   "foo-value",
						"empty": "",
					},
					DriverConfig: &mount.Driver{
						Name: "my-driver",
						Options: map[string]string{
							"foo":   "foo-value",
							"empty": "",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			var m MountOpt

			assert.NilError(t, m.Set(tc.value))
			assert.Check(t, is.DeepEqual(m.values[0], tc.exp))
		})
	}
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

func TestMountOptSetBindCreateMountpoint(t *testing.T) {
	tests := []struct {
		value  string
		exp    bool
		expErr string
	}{
		{value: "", exp: false},
		{value: "bind-create-mountpoint", exp: true},
		{value: "bind-create-mountpoint=", expErr: `invalid value for 'bind-create-mountpoint': value is empty`},
		{value: "bind-create-mountpoint= true", expErr: `invalid value for 'bind-create-mountpoint' in 'bind-create-mountpoint= true': value should not have whitespace`},
		{value: "bind-create-mountpoint=no", expErr: `invalid value for 'bind-create-mountpoint': invalid boolean value ("no"): must be one of "true", "1", "false", or "0" (default "true")`},
		{value: "bind-create-mountpoint=1", exp: true},
		{value: "bind-create-mountpoint=true", exp: true},
		{value: "bind-create-mountpoint=0", exp: false},
		{value: "bind-create-mountpoint=false", exp: false},
	}

	for _, tc := range tests {
		name := tc.value
		if name == "" {
			name = "not set"
		}
		t.Run(name, func(t *testing.T) {
			val := "type=bind,target=/foo,source=/foo"
			if tc.value != "" {
				val += "," + tc.value
			}
			var m MountOpt
			err := m.Set(val)
			if tc.expErr != "" {
				assert.Error(t, err, tc.expErr)
				return
			}
			assert.NilError(t, err)
			if tc.value == "" {
				assert.Check(t, is.Nil(m.values[0].BindOptions))
			} else {
				assert.Check(t, m.values[0].BindOptions != nil)
				assert.Check(t, is.Equal(m.values[0].BindOptions.CreateMountpoint, tc.exp))
			}
		})
	}
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
