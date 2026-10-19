package manifest

import (
	"io"
	"testing"

	"github.com/docker/cli/cli/manifest/store"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestManifestAnnotateError(t *testing.T) {
	testCases := []struct {
		args          []string
		expectedError string
	}{
		{
			args:          []string{"too-few-arguments"},
			expectedError: "requires 2 arguments",
		},
		{
			args:          []string{"th!si'sa/fa!ke/li$t/name", "example.com/alpine:3.0"},
			expectedError: "error parsing name for manifest list",
		},
		{
			args:          []string{"example.com/list:v1", "th!si'sa/fa!ke/im@ge/nam32"},
			expectedError: "error parsing name for manifest",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(nil)
		cmd := newAnnotateCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestManifestAnnotate(t *testing.T) {
	manifestStore := store.NewStore(t.TempDir())

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(manifestStore)
	namedRef := ref(t, "alpine:3.0")
	imageManifest := fullImageManifest(t, namedRef)
	err := manifestStore.Save(ref(t, "list:v1"), namedRef, imageManifest)
	assert.NilError(t, err)

	cmd := newAnnotateCommand(cli)
	cmd.SetArgs([]string{"example.com/list:v1", "example.com/fake:0.0"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	expectedError := "manifest for image example.com/fake:0.0 does not exist"
	assert.ErrorContains(t, cmd.Execute(), expectedError)

	cmd.SetArgs([]string{"example.com/list:v1", "example.com/alpine:3.0"})
	cmd.Flags().Set("os", "freebsd")
	cmd.Flags().Set("arch", "fake")
	cmd.Flags().Set("os-version", "1")
	cmd.Flags().Set("os-features", "feature1")
	cmd.Flags().Set("variant", "v7")
	expectedError = "manifest entry for image has unsupported os/arch combination"
	assert.ErrorContains(t, cmd.Execute(), expectedError)

	cmd.Flags().Set("arch", "arm")
	assert.NilError(t, cmd.Execute())

	cmd = newInspectCommand(cli)
	err = cmd.Flags().Set("verbose", "true")
	assert.NilError(t, err)
	cmd.SetArgs([]string{"example.com/list:v1", "example.com/alpine:3.0"})
	cmd.SetErr(io.Discard)
	assert.NilError(t, cmd.Execute())
	actual := cli.OutBuffer()
	expected := golden.Get(t, "inspect-annotate.golden")
	assert.Check(t, is.Equal(string(expected), actual.String()))
}

func TestManifestAnnotateUnsetFields(t *testing.T) {
	manifestStore := store.NewStore(t.TempDir())

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(manifestStore)
	listRef := ref(t, "list:v1")
	namedRef := ref(t, "alpine:3.0")
	err := manifestStore.Save(listRef, namedRef, fullImageManifest(t, namedRef))
	assert.NilError(t, err)

	annotate := func(args ...string) {
		t.Helper()
		cmd := newAnnotateCommand(cli)
		cmd.SetArgs(append([]string{"example.com/list:v1", "example.com/alpine:3.0"}, args...))
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.NilError(t, cmd.Execute())
	}

	annotate("--os", "linux", "--arch", "arm", "--variant", "v7", "--os-version", "1")
	imageManifest, err := manifestStore.Get(listRef, namedRef)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(imageManifest.Descriptor.Platform.Variant, "v7"))
	assert.Check(t, is.Equal(imageManifest.Descriptor.Platform.OSVersion, "1"))

	// Not passing the flags should preserve the existing values.
	annotate()
	imageManifest, err = manifestStore.Get(listRef, namedRef)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(imageManifest.Descriptor.Platform.Variant, "v7"))
	assert.Check(t, is.Equal(imageManifest.Descriptor.Platform.OSVersion, "1"))

	// Passing an empty value should unset them.
	annotate("--variant", "", "--os-version", "")
	imageManifest, err = manifestStore.Get(listRef, namedRef)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(imageManifest.Descriptor.Platform.Variant, ""))
	assert.Check(t, is.Equal(imageManifest.Descriptor.Platform.OSVersion, ""))
}
