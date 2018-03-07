package store

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/manifest/types"
	"github.com/docker/distribution/reference"
	"github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

type fakeRef struct {
	name string
}

func (f fakeRef) String() string {
	return f.name
}

func (f fakeRef) Name() string {
	return f.name
}

func ref(name string) fakeRef {
	return fakeRef{name: name}
}

func sref(t *testing.T, name string) *types.SerializableNamed {
	named, err := reference.ParseNamed("example.com/" + name)
	assert.NilError(t, err)
	return &types.SerializableNamed{Named: named}
}

func newTestStore(t *testing.T) (Store, func()) {
	tmpdir, err := ioutil.TempDir("", "manifest-store-test")
	assert.NilError(t, err)

	return NewStore(tmpdir), func() { os.RemoveAll(tmpdir) }
}

func getFiles(t *testing.T, store Store) []os.FileInfo {
	infos, err := ioutil.ReadDir(store.(*fsStore).root)
	assert.NilError(t, err)
	return infos
}

func TestStoreRemove(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	listRef := ref("list")
	data := types.ImageManifest{Ref: sref(t, "abcdef")}
	assert.NilError(t, store.Save(listRef, ref("manifest"), data))
	assert.Assert(t, is.Len(getFiles(t, store), 1))

	assert.Check(t, store.Remove(listRef))
	assert.Check(t, is.Len(getFiles(t, store), 0))
}

func TestStoreSaveAndGet(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	listRef := ref("list")
	data := types.ImageManifest{Ref: sref(t, "abcdef")}
	err := store.Save(listRef, ref("exists"), data)
	assert.NilError(t, err)

	var testcases = []struct {
		listRef     reference.Reference
		manifestRef reference.Reference
		expected    types.ImageManifest
		expectedErr string
	}{
		{
			listRef:     listRef,
			manifestRef: ref("exists"),
			expected:    data,
		},
		{
			listRef:     listRef,
			manifestRef: ref("exist:does-not"),
			expectedErr: "No such manifest: exist:does-not",
		},
		{
			listRef:     ref("list:does-not-exist"),
			manifestRef: ref("manifest:does-not-exist"),
			expectedErr: "No such manifest: manifest:does-not-exist",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.manifestRef.String(), func(t *testing.T) {
			actual, err := store.Get(testcase.listRef, testcase.manifestRef)
			if testcase.expectedErr != "" {
				assert.Error(t, err, testcase.expectedErr)
				assert.Check(t, IsNotFound(err))
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, testcase.expected, actual, cmpReferenceNamed)
		})
	}
}

var cmpReferenceNamed = cmp.Transformer("namedref", func(r reference.Named) string {
	return r.String()
})

func TestStoreGetList(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	listRef := ref("list")
	first := types.ImageManifest{Ref: sref(t, "first")}
	assert.NilError(t, store.Save(listRef, ref("first"), first))
	second := types.ImageManifest{Ref: sref(t, "second")}
	assert.NilError(t, store.Save(listRef, ref("exists"), second))

	list, err := store.GetList(listRef)
	assert.NilError(t, err)
	assert.Check(t, is.Len(list, 2))
}

func TestStoreGetListDoesNotExist(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	listRef := ref("list")
	_, err := store.GetList(listRef)
	assert.Error(t, err, "No such manifest: list")
	assert.Check(t, IsNotFound(err))
}
