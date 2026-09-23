//go:build linux

package standalone

import (
	"path/filepath"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func testEngine(t *testing.T) *engine {
	t.Helper()
	root := t.TempDir()
	return &engine{cfg: &config{root: root, runDir: filepath.Join(root, "run"), snapshotter: snapshotterNative}}
}

func TestValidVolumeName(t *testing.T) {
	assert.Check(t, validVolumeName("data"))
	assert.Check(t, validVolumeName("my-vol_1.2"))
	assert.Check(t, !validVolumeName(""))
	assert.Check(t, !validVolumeName("bad/name"))
	assert.Check(t, !validVolumeName("bad name"))
}

func TestCreateAndGetVolume(t *testing.T) {
	e := testEngine(t)

	vm, err := e.createVolume("data", map[string]string{"k": "v"}, false)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(vm.Name, "data"))
	assert.Check(t, is.Equal(vm.Anonymous, false))

	got, err := e.getVolume("data")
	assert.NilError(t, err)
	assert.Check(t, is.Equal(got.Name, "data"))
	assert.Check(t, is.Equal(got.Labels["k"], "v"))

	// Creating it again is a no-op.
	again, err := e.createVolume("data", nil, false)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(again.Name, "data"))

	anon, err := e.createVolume("", nil, true)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(len(anon.Name), 64))
	assert.Check(t, anon.Anonymous)

	vols, err := e.listVolumes()
	assert.NilError(t, err)
	assert.Check(t, is.Len(vols, 2))

	api := e.volumeToAPI(vm)
	assert.Check(t, is.Equal(api.Driver, "local"))
	assert.Check(t, is.Equal(api.Mountpoint, e.volumeDataPath("data")))

	assert.NilError(t, e.removeVolume("data"))
	_, err = e.getVolume("data")
	assert.Check(t, err != nil)
	assert.Check(t, e.removeVolume("nosuch") != nil)
}

func TestResolveBind(t *testing.T) {
	e := testEngine(t)

	t.Run("named volume", func(t *testing.T) {
		om, mp, anon, err := e.resolveBind("vol:/data")
		assert.NilError(t, err)
		assert.Check(t, is.Equal(anon, ""))
		assert.Check(t, is.Equal(om.Destination, "/data"))
		assert.Check(t, is.Equal(om.Source, e.volumeDataPath("vol")))
		assert.Check(t, is.Equal(mp.Type, mount.TypeVolume))
		assert.Check(t, is.Equal(mp.Name, "vol"))
		assert.Check(t, mp.RW)
	})

	t.Run("read-only bind", func(t *testing.T) {
		dir := t.TempDir()
		om, mp, _, err := e.resolveBind(dir + ":/host:ro")
		assert.NilError(t, err)
		assert.Check(t, is.Equal(mp.Type, mount.TypeBind))
		assert.Check(t, !mp.RW)
		assert.Check(t, is.Contains(om.Options, "ro"))
	})

	t.Run("anonymous volume", func(t *testing.T) {
		_, mp, anon, err := e.resolveBind("/data")
		assert.NilError(t, err)
		assert.Check(t, anon != "")
		assert.Check(t, is.Equal(mp.Name, anon))
	})

	for _, spec := range []string{"vol:data", "vol:/data:bogus", "a:b:c:d"} {
		t.Run("invalid "+spec, func(t *testing.T) {
			_, _, _, err := e.resolveBind(spec)
			assert.Check(t, err != nil)
		})
	}
}

func TestResolveMounts(t *testing.T) {
	e := testEngine(t)
	cfg := &container.Config{Volumes: map[string]struct{}{"/cfgvol": {}}}
	hc := &container.HostConfig{
		Binds: []string{"named:/named"},
		Mounts: []mount.Mount{
			{Type: mount.TypeTmpfs, Target: "/tmpfs", TmpfsOptions: &mount.TmpfsOptions{SizeBytes: 1024}},
		},
		Tmpfs: map[string]string{"/legacytmp": "size=2m"},
	}
	imageVolumes := map[string]struct{}{"/imgvol": {}}

	mounts, points, anon, err := e.resolveMounts(cfg, hc, imageVolumes)
	assert.NilError(t, err)
	assert.Check(t, is.Len(mounts, 5))
	assert.Check(t, is.Len(points, 5))
	// One anonymous volume for the image volume and one for the config volume.
	assert.Check(t, is.Len(anon, 2))

	dests := map[string]bool{}
	for _, m := range mounts {
		dests[m.Destination] = true
	}
	for _, want := range []string{"/named", "/tmpfs", "/legacytmp", "/imgvol", "/cfgvol"} {
		assert.Check(t, dests[want], "missing mount %s", want)
	}
}

func TestResolveMountsDuplicateDestination(t *testing.T) {
	e := testEngine(t)
	hc := &container.HostConfig{Binds: []string{"first:/data", "second:/data"}}
	mounts, _, _, err := e.resolveMounts(&container.Config{}, hc, nil)
	assert.NilError(t, err)
	// The first mount for a destination wins, as in the daemon.
	assert.Check(t, is.Len(mounts, 1))
	assert.Check(t, is.Equal(mounts[0].Source, e.volumeDataPath("first")))
}

func TestResolveMountUnsupportedType(t *testing.T) {
	e := testEngine(t)
	_, _, _, err := e.resolveMount(mount.Mount{Type: "image", Target: "/x"}) //nolint:dogsled // only the error matters
	assert.Check(t, err != nil)
}
