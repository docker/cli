package trust

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestTrustInspectCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "not-enough-args",
			expectedError: "requires at least 1 argument",
		},
		{
			name:          "sha-reference",
			args:          []string{"870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd"},
			expectedError: "invalid repository name",
		},
		{
			name:          "invalid-img-reference",
			args:          []string{"ALPINE"},
			expectedError: "invalid reference format",
		},
	}
	for _, tc := range testCases {
		cmd := newInspectCommand(
			test.NewFakeCli(&fakeClient{}))
		cmd.Flags().Set("pretty", "true")
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestTrustInspectCommandOfflineErrors(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetOfflineNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"nonexistent-reg-name.io/image"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "No signatures or cannot access nonexistent-reg-name.io/image")

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetOfflineNotaryRepository)
	cmd = newInspectCommand(cli)
	cmd.SetArgs([]string{"nonexistent-reg-name.io/image:tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "No signatures or cannot access nonexistent-reg-name.io/image")
}

func TestTrustInspectCommandUninitializedErrors(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetUninitializedNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"reg/unsigned-img"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "No signatures or cannot access reg/unsigned-img")
	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-uninitialized.golden")

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetUninitializedNotaryRepository)
	cmd = newInspectCommand(cli)
	cmd.SetArgs([]string{"reg/unsigned-img:tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "No signatures or cannot access reg/unsigned-img:tag")
	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-uninitialized.golden")
}

func TestTrustInspectCommandEmptyNotaryRepo(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetEmptyTargetsNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"reg/img:unsigned-tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-empty-repo.golden")
}

func TestTrustInspectCommandFullRepoWithoutSigners(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetLoadedWithNoSignersNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"signed-repo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-full-repo-no-signers.golden")
}

func TestTrustInspectCommandOneTagWithoutSigners(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetLoadedWithNoSignersNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"signed-repo:green"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-one-tag-no-signers.golden")
}

func TestTrustInspectCommandFullRepoWithSigners(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetLoadedNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"signed-repo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-full-repo-with-signers.golden")
}

func TestTrustInspectCommandMultipleFullReposWithSigners(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetLoadedNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"signed-repo", "signed-repo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-multiple-repos-with-signers.golden")
}

func TestTrustInspectCommandUnsignedTagInSignedRepo(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetLoadedNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"signed-repo:unsigned"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-unsigned-tag-with-signers.golden")
}
