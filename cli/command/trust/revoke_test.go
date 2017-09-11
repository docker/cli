package trust

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/notary/client"
	"github.com/docker/notary/passphrase"
	"github.com/docker/notary/trustpinning"
	"github.com/stretchr/testify/assert"
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
			name:          "trust-data-for-tag-does-not-exist",
			args:          []string{"alpine:foo"},
			expectedError: "could not remove signature for alpine:foo: No valid trust data for foo",
		},
		{
			name:          "invalid-img-reference",
			args:          []string{"ALPINE"},
			expectedError: "invalid reference format",
		},
		{
			name: "unsigned-img-reference",
			args: []string{"riyaz/unsigned-img:v1"},
			expectedError: strings.Join([]string{
				"could not remove signature for riyaz/unsigned-img:v1:",
				"notary.docker.io does not have trust data for docker.io/riyaz/unsigned-img",
			}, " "),
		},
		{
			name:          "no-signing-keys-for-image",
			args:          []string{"alpine", "-y"},
			expectedError: "could not remove signature for alpine: could not find necessary signing keys",
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
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewRevokeTrustAllSigConfirmation(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cmd := newRevokeCommand(cli)
	cmd.SetArgs([]string{"alpine"})
	assert.NoError(t, cmd.Execute())

	assert.Contains(t, cli.OutBuffer().String(), "Please confirm you would like to delete all signature data for alpine? [y/N] \nAborting action.")
}

func TestGetSignableRolesForTargetAndRemoveError(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever("password"), trustpinning.TrustPinConfig{})
	target := client.Target{}
	err = getSignableRolesForTargetAndRemove(target, notaryRepo)
	assert.EqualError(t, err, "client is offline")
}
