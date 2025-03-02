package trust

import (
	"context"
	"io"
	"testing"

	"github.com/docker/cli/cli/trust"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestTrustRevokeCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "not-enough-args",
			expectedError: "requires 1 argument",
		},
		{
			name:          "too-many-args",
			args:          []string{"remote1", "remote2"},
			expectedError: "requires 1 argument",
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
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestTrustRevokeCommand(t *testing.T) {
	revokeCancelledError := "trust revoke has been cancelled"

	testCases := []struct {
		doc              string
		notaryRepository func(trust.ImageRefAndAuth, []string) (client.Repository, error)
		args             []string
		expectedErr      string
		expectedMessage  string
	}{
		{
			doc:              "OfflineErrors_Confirm",
			notaryRepository: notary.GetOfflineNotaryRepository,
			args:             []string{"reg-name.io/image"},
			expectedMessage:  "Confirm you would like to delete all signature data for reg-name.io/image? [y/N] ",
			expectedErr:      revokeCancelledError,
		},
		{
			doc:              "OfflineErrors_Offline",
			notaryRepository: notary.GetOfflineNotaryRepository,
			args:             []string{"reg-name.io/image", "-y"},
			expectedErr:      "could not remove signature for reg-name.io/image: client is offline",
		},
		{
			doc:              "OfflineErrors_WithTag_Offline",
			notaryRepository: notary.GetOfflineNotaryRepository,
			args:             []string{"reg-name.io/image:tag"},
			expectedErr:      "could not remove signature for reg-name.io/image:tag: client is offline",
		},
		{
			doc:              "UninitializedErrors_Confirm",
			notaryRepository: notary.GetUninitializedNotaryRepository,
			args:             []string{"reg-name.io/image"},
			expectedMessage:  "Confirm you would like to delete all signature data for reg-name.io/image? [y/N] ",
			expectedErr:      revokeCancelledError,
		},
		{
			doc:              "UninitializedErrors_NoTrustData",
			notaryRepository: notary.GetUninitializedNotaryRepository,
			args:             []string{"reg-name.io/image", "-y"},
			expectedErr:      "could not remove signature for reg-name.io/image:  does not have trust data for",
		},
		{
			doc:              "UninitializedErrors_WithTag_NoTrustData",
			notaryRepository: notary.GetUninitializedNotaryRepository,
			args:             []string{"reg-name.io/image:tag"},
			expectedErr:      "could not remove signature for reg-name.io/image:tag:  does not have trust data for",
		},
		{
			doc:              "EmptyNotaryRepo_Confirm",
			notaryRepository: notary.GetEmptyTargetsNotaryRepository,
			args:             []string{"reg-name.io/image"},
			expectedMessage:  "Confirm you would like to delete all signature data for reg-name.io/image? [y/N] ",
			expectedErr:      revokeCancelledError,
		},
		{
			doc:              "EmptyNotaryRepo_NoSignedTags",
			notaryRepository: notary.GetEmptyTargetsNotaryRepository,
			args:             []string{"reg-name.io/image", "-y"},
			expectedErr:      "could not remove signature for reg-name.io/image: no signed tags to remove",
		},
		{
			doc:              "EmptyNotaryRepo_NoValidTrustData",
			notaryRepository: notary.GetEmptyTargetsNotaryRepository,
			args:             []string{"reg-name.io/image:tag"},
			expectedErr:      "could not remove signature for reg-name.io/image:tag: No valid trust data for tag",
		},
		{
			doc:              "AllSigConfirmation",
			notaryRepository: notary.GetEmptyTargetsNotaryRepository,
			args:             []string{"alpine"},
			expectedMessage:  "Confirm you would like to delete all signature data for alpine? [y/N] ",
			expectedErr:      revokeCancelledError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			cli.SetNotaryClient(tc.notaryRepository)
			cmd := newRevokeCommand(cli)
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, cmd.Execute(), tc.expectedErr)
			} else {
				assert.NilError(t, cmd.Execute())
			}
			assert.Check(t, is.Contains(cli.OutBuffer().String(), tc.expectedMessage))
		})
	}
}

func TestGetSignableRolesForTargetAndRemoveError(t *testing.T) {
	notaryRepo, err := client.NewFileCachedRepository(t.TempDir(), "gun", "https://localhost", nil, passphrase.ConstantRetriever("password"), trustpinning.TrustPinConfig{})
	assert.NilError(t, err)
	target := client.Target{}
	err = getSignableRolesForTargetAndRemove(target, notaryRepo)
	assert.Error(t, err, "client is offline")
}

func TestRevokeTrustPromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{})
	cmd := newRevokeCommand(cli)
	cmd.SetArgs([]string{"example/trust-demo"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
	golden.Assert(t, cli.OutBuffer().String(), "trust-revoke-prompt-termination.golden")
}
