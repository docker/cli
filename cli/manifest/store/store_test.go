package store

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/manifest/types"
	"github.com/docker/distribution/reference"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	return &types.SerializableNamed{Named: named}
}

func newTestStore(t *testing.T) (Store, func()) {
	tmpdir, err := ioutil.TempDir("", "manifest-store-test")
	require.NoError(t, err)

	return NewStore(tmpdir), func() { os.RemoveAll(tmpdir) }
}

func getFiles(t *testing.T, store Store) []os.FileInfo {
	infos, err := ioutil.ReadDir(store.(*fsStore).root)
	require.NoError(t, err)
	return infos
}

func TestStoreRemove(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	listRef := ref("list")
	data := types.ImageManifest{Ref: sref(t, "abcdef")}
	require.NoError(t, store.Save(listRef, ref("manifest"), data))
	require.Len(t, getFiles(t, store), 1)

	assert.NoError(t, store.Remove(listRef))
	assert.Len(t, getFiles(t, store), 0)
}

func TestStoreSaveAndGet(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	listRef := ref("list")
	data := types.ImageManifest{Ref: sref(t, "abcdef")}
	err := store.Save(listRef, ref("exists"), data)
	require.NoError(t, err)

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
		actual, err := store.Get(testcase.listRef, testcase.manifestRef)
		if testcase.expectedErr != "" {
			assert.EqualError(t, err, testcase.expectedErr)
			assert.True(t, IsNotFound(err))
			continue
		}
		if !assert.NoError(t, err, testcase.manifestRef.String()) {
			continue
		}
		assert.Equal(t, testcase.expected, actual, testcase.manifestRef.String())
	}
}

func TestStoreGetList(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	listRef := ref("list")
	first := types.ImageManifest{Ref: sref(t, "first")}
	require.NoError(t, store.Save(listRef, ref("first"), first))
	second := types.ImageManifest{Ref: sref(t, "second")}
	require.NoError(t, store.Save(listRef, ref("exists"), second))

	list, err := store.GetList(listRef)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestStoreGetListDoesNotExist(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	listRef := ref("list")
	_, err := store.GetList(listRef)
	assert.EqualError(t, err, "No such manifest: list")
	assert.True(t, IsNotFound(err))
}
