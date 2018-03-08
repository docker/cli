package trust

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
)

func TestTrustRevokeCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "not-enough-args",
			expectedError: "requires exactly 1 argument",
		},
		{
			name:          "too-many-args",
			args:          []string{"remote1", "remote2"},
			expectedError: "requires exactly 1 argument",
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
		{
			name:          "digest-reference",
			args:          []string{"ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2"},
			expectedError: "cannot use a digest reference for IMAGE:TAG",
		},
	}
	for _, tc := range testCases {
		cmd := newRevokeCommand(
			test.NewFakeCli(&fakeClient{}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestTrustRevokeCommandOfflineErrors(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetOfflineNotaryRepository)
	cmd := newRevokeCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image"})
	cmd.SetOutput(ioutil.Discard)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Contains(cli.OutBuffer().String(), "Please confirm you would like to delete all signature data for reg-name.io/image? [y/N] \nAborting action."))

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetOfflineNotaryRepository)
	cmd = newRevokeCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image", "-y"})
	cmd.SetOutput(ioutil.Discard)

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetOfflineNotaryRepository)
	cmd = newRevokeCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image:tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "could not remove signature for reg-name.io/image:tag: client is offline")
}

func TestTrustRevokeCommandUninitializedErrors(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetUninitializedNotaryRepository)
	cmd := newRevokeCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image"})
	cmd.SetOutput(ioutil.Discard)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Contains(cli.OutBuffer().String(), "Please confirm you would like to delete all signature data for reg-name.io/image? [y/N] \nAborting action."))

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetUninitializedNotaryRepository)
	cmd = newRevokeCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image", "-y"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "could not remove signature for reg-name.io/image:  does not have trust data for")

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetUninitializedNotaryRepository)
	cmd = newRevokeCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image:tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "could not remove signature for reg-name.io/image:tag:  does not have trust data for")
}

func TestTrustRevokeCommandEmptyNotaryRepo(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetEmptyTargetsNotaryRepository)
	cmd := newRevokeCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image"})
	cmd.SetOutput(ioutil.Discard)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Contains(cli.OutBuffer().String(), "Please confirm you would like to delete all signature data for reg-name.io/image? [y/N] \nAborting action."))

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetEmptyTargetsNotaryRepository)
	cmd = newRevokeCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image", "-y"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "could not remove signature for reg-name.io/image: no signed tags to remove")

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetEmptyTargetsNotaryRepository)
	cmd = newRevokeCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image:tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "could not remove signature for reg-name.io/image:tag: No valid trust data for tag")
}

func TestNewRevokeTrustAllSigConfirmation(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notary.GetEmptyTargetsNotaryRepository)
	cmd := newRevokeCommand(cli)
	cmd.SetArgs([]string{"alpine"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Contains(cli.OutBuffer().String(), "Please confirm you would like to delete all signature data for alpine? [y/N] \nAborting action."))
}

func TestGetSignableRolesForTargetAndRemoveError(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever("password"), trustpinning.TrustPinConfig{})
	assert.NilError(t, err)
	target := client.Target{}
	err = getSignableRolesForTargetAndRemove(target, notaryRepo)
	assert.Error(t, err, "client is offline")
}
