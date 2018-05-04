package manifest

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/manifest/store"
	"github.com/docker/cli/cli/manifest/types"
	manifesttypes "github.com/docker/cli/cli/manifest/types"
	"github.com/docker/cli/internal/test"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

func newTempManifestStore(t *testing.T) (store.Store, func()) {
	tmpdir, err := ioutil.TempDir("", "test-manifest-storage")
	assert.NilError(t, err)

	return store.NewStore(tmpdir), func() { os.RemoveAll(tmpdir) }
}

func ref(t *testing.T, name string) reference.Named {
	named, err := reference.ParseNamed("example.com/" + name)
	assert.NilError(t, err)
	return named
}

func fullImageManifest(t *testing.T, ref reference.Named) types.ImageManifest {
	man, err := schema2.FromStruct(schema2.Manifest{
		Versioned: schema2.SchemaVersion,
		Config: distribution.Descriptor{
			Digest:    "sha256:7328f6f8b41890597575cbaadc884e7386ae0acc53b747401ebce5cf0d624560",
			Size:      1520,
			MediaType: schema2.MediaTypeImageConfig,
		},
		Layers: []distribution.Descriptor{
			{
				MediaType: schema2.MediaTypeLayer,
				Size:      1990402,
				Digest:    "sha256:88286f41530e93dffd4b964e1db22ce4939fffa4a4c665dab8591fbab03d4926",
			},
		},
	})
	assert.NilError(t, err)
	// TODO: include image data for verbose inspect
	return types.NewImageManifest(ref, digest.Digest("sha256:7328f6f8b41890597575cbaadc884e7386ae0acc53b747401ebce5cf0d62abcd"), types.Image{OS: "linux", Architecture: "amd64"}, man)
}

func TestInspectCommandLocalManifestNotFound(t *testing.T) {
	store, cleanup := newTempManifestStore(t)
	defer cleanup()

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)

	cmd := newInspectCommand(cli)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"example.com/list:v1", "example.com/alpine:3.0"})
	err := cmd.Execute()
	assert.Error(t, err, "No such manifest: example.com/alpine:3.0")
}

func TestInspectCommandNotFound(t *testing.T) {
	store, cleanup := newTempManifestStore(t)
	defer cleanup()

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)
	cli.SetRegistryClient(&fakeRegistryClient{
		getManifestFunc: func(_ context.Context, _ reference.Named) (manifesttypes.ImageManifest, error) {
			return manifesttypes.ImageManifest{}, errors.New("missing")
		},
		getManifestListFunc: func(ctx context.Context, ref reference.Named) ([]manifesttypes.ImageManifest, error) {
			return nil, errors.Errorf("No such manifest: %s", ref)
		},
	})

	cmd := newInspectCommand(cli)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"example.com/alpine:3.0"})
	err := cmd.Execute()
	assert.Error(t, err, "No such manifest: example.com/alpine:3.0")
}

func TestInspectCommandLocalManifest(t *testing.T) {
	store, cleanup := newTempManifestStore(t)
	defer cleanup()

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)
	namedRef := ref(t, "alpine:3.0")
	imageManifest := fullImageManifest(t, namedRef)
	err := store.Save(ref(t, "list:v1"), namedRef, imageManifest)
	assert.NilError(t, err)

	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"example.com/list:v1", "example.com/alpine:3.0"})
	assert.NilError(t, cmd.Execute())
	actual := cli.OutBuffer()
	expected := golden.Get(t, "inspect-manifest.golden")
	assert.Check(t, is.Equal(string(expected), actual.String()))
}

func TestInspectcommandRemoteManifest(t *testing.T) {
	store, cleanup := newTempManifestStore(t)
	defer cleanup()

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)
	cli.SetRegistryClient(&fakeRegistryClient{
		getManifestFunc: func(_ context.Context, ref reference.Named) (manifesttypes.ImageManifest, error) {
			return fullImageManifest(t, ref), nil
		},
	})

	cmd := newInspectCommand(cli)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"example.com/alpine:3.0"})
	assert.NilError(t, cmd.Execute())
	actual := cli.OutBuffer()
	expected := golden.Get(t, "inspect-manifest.golden")
	assert.Check(t, is.Equal(string(expected), actual.String()))
}
