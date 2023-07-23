package manifest

import (
	"io"
	"testing"

	"github.com/docker/cli/cli/manifest/store"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

// create two manifest lists and remove them both
func TestRmSeveralManifests(t *testing.T) {
	store := store.NewStore(t.TempDir())

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)

	list1 := ref(t, "first:1")
	namedRef := ref(t, "alpine:3.0")
	err := store.Save(list1, namedRef, fullImageManifest(t, namedRef))
	assert.NilError(t, err)
	namedRef = ref(t, "alpine:3.1")
	err = store.Save(list1, namedRef, fullImageManifest(t, namedRef))
	assert.NilError(t, err)

	list2 := ref(t, "second:2")
	namedRef = ref(t, "alpine:3.2")
	err = store.Save(list2, namedRef, fullImageManifest(t, namedRef))
	assert.NilError(t, err)

	cmd := newRmManifestListCommand(cli)
	cmd.SetArgs([]string{"example.com/first:1", "example.com/second:2"})
	cmd.SetOut(io.Discard)
	err = cmd.Execute()
	assert.NilError(t, err)

	_, search1 := cli.ManifestStore().GetList(list1)
	_, search2 := cli.ManifestStore().GetList(list2)
	assert.Error(t, search1, `"example.com/first:1" does not exist: manifest not found`)
	assert.Error(t, search2, `"example.com/second:2" does not exist: manifest not found`)
}

// attempt to remove a manifest list which was never created
func TestRmManifestNotCreated(t *testing.T) {
	store := store.NewStore(t.TempDir())

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)

	list2 := ref(t, "second:2")
	namedRef := ref(t, "alpine:3.2")
	err := store.Save(list2, namedRef, fullImageManifest(t, namedRef))
	assert.NilError(t, err)

	cmd := newRmManifestListCommand(cli)
	cmd.SetArgs([]string{"example.com/first:1", "example.com/second:2"})
	cmd.SetOut(io.Discard)
	err = cmd.Execute()
	assert.Error(t, err, `"example.com/first:1" does not exist: manifest not found`)

	_, err = cli.ManifestStore().GetList(list2)
	assert.Error(t, err, `"example.com/second:2" does not exist: manifest not found`)
}
