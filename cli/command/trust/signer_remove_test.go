package trust

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
)

func TestTrustSignerRemoveErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "not-enough-args-0",
			expectedError: "requires at least 2 arguments",
		},
		{
			name:          "not-enough-args-1",
			args:          []string{"user"},
			expectedError: "requires at least 2 arguments",
		},
	}
	for _, tc := range testCases {
		cmd := newSignerRemoveCommand(
			test.NewFakeCli(&fakeClient{}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
	testCasesWithOutput := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "not-an-image",
			args:          []string{"user", "notanimage"},
			expectedError: "error retrieving signers for notanimage",
		},
		{
			name:          "sha-reference",
			args:          []string{"user", "870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd"},
			expectedError: "invalid repository name",
		},
		{
			name:          "invalid-img-reference",
			args:          []string{"user", "ALPINE"},
			expectedError: "invalid reference format",
		},
	}
	for _, tc := range testCasesWithOutput {
		cli := test.NewFakeCli(&fakeClient{})
		cli.SetNotaryClient(getOfflineNotaryRepository)
		cmd := newSignerRemoveCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		cmd.Execute()
		assert.Contains(t, cli.ErrBuffer().String(), tc.expectedError)
	}

}

func TestRemoveSingleSigner(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getLoadedNotaryRepository)
	err := removeSingleSigner(cli, "signed-repo", "test", true)
	assert.EqualError(t, err, "No signer test for repository signed-repo")
	err = removeSingleSigner(cli, "signed-repo", "releases", true)
	assert.EqualError(t, err, "releases is a reserved keyword and cannot be removed")
}

func TestRemoveMultipleSigners(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getLoadedNotaryRepository)
	err := removeSigner(cli, signerRemoveOptions{signer: "test", repos: []string{"signed-repo", "signed-repo"}, forceYes: true})
	assert.EqualError(t, err, "Error removing signer from: signed-repo, signed-repo")
	assert.Contains(t, cli.ErrBuffer().String(),
		"No signer test for repository signed-repo")
	assert.Contains(t, cli.OutBuffer().String(), "Removing signer \"test\" from signed-repo...\n")
}
func TestRemoveLastSignerWarning(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getLoadedNotaryRepository)

	err := removeSigner(cli, signerRemoveOptions{signer: "alice", repos: []string{"signed-repo"}, forceYes: false})
	assert.NoError(t, err)
	assert.Contains(t, cli.OutBuffer().String(),
		"The signer \"alice\" signed the last released version of signed-repo. "+
			"Removing this signer will make signed-repo unpullable. "+
			"Are you sure you want to continue? [y/N]")
}

func TestIsLastSignerForReleases(t *testing.T) {
	role := data.Role{}
	releaserole := client.RoleWithSignatures{}
	releaserole.Name = releasesRoleTUFName
	releaserole.Threshold = 1
	allrole := []client.RoleWithSignatures{releaserole}
	lastsigner, _ := isLastSignerForReleases(role, allrole)
	assert.Equal(t, false, lastsigner)

	role.KeyIDs = []string{"deadbeef"}
	sig := data.Signature{}
	sig.KeyID = "deadbeef"
	releaserole.Signatures = []data.Signature{sig}
	releaserole.Threshold = 1
	allrole = []client.RoleWithSignatures{releaserole}
	lastsigner, _ = isLastSignerForReleases(role, allrole)
	assert.Equal(t, true, lastsigner)

	sig.KeyID = "8badf00d"
	releaserole.Signatures = []data.Signature{sig}
	releaserole.Threshold = 1
	allrole = []client.RoleWithSignatures{releaserole}
	lastsigner, _ = isLastSignerForReleases(role, allrole)
	assert.Equal(t, false, lastsigner)
}
