package trust

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/trust"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/theupdateframework/notary/client"
	"gotest.tools/assert"
	"gotest.tools/golden"
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

func TestTrustInspectCommandRepositoryErrors(t *testing.T) {
	testCases := []struct {
		doc              string
		args             []string
		notaryRepository func(trust.ImageRefAndAuth, []string) (client.Repository, error)
		err              string
		golden           string
	}{
		{
			doc:              "OfflineErrors",
			args:             []string{"nonexistent-reg-name.io/image"},
			notaryRepository: notary.GetOfflineNotaryRepository,
			err:              "No signatures or cannot access nonexistent-reg-name.io/image",
		},
		{
			doc:              "OfflineErrorsWithImageTag",
			args:             []string{"nonexistent-reg-name.io/image:tag"},
			notaryRepository: notary.GetOfflineNotaryRepository,
			err:              "No signatures or cannot access nonexistent-reg-name.io/image:tag",
		},
		{
			doc:              "UninitializedErrors",
			args:             []string{"reg/unsigned-img"},
			notaryRepository: notary.GetUninitializedNotaryRepository,
			err:              "No signatures or cannot access reg/unsigned-img",
			golden:           "trust-inspect-uninitialized.golden",
		},
		{
			doc:              "UninitializedErrorsWithImageTag",
			args:             []string{"reg/unsigned-img:tag"},
			notaryRepository: notary.GetUninitializedNotaryRepository,
			err:              "No signatures or cannot access reg/unsigned-img:tag",
			golden:           "trust-inspect-uninitialized.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			cli.SetNotaryClient(tc.notaryRepository)
			cmd := newInspectCommand(cli)
			cmd.SetArgs(tc.args)
			cmd.SetOutput(ioutil.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.err)
			if tc.golden != "" {
				golden.Assert(t, cli.OutBuffer().String(), tc.golden)
			}
		})
	}
}

func TestTrustInspectCommand(t *testing.T) {
	testCases := []struct {
		doc              string
		args             []string
		notaryRepository func(trust.ImageRefAndAuth, []string) (client.Repository, error)
		golden           string
	}{
		{
			doc:              "EmptyNotaryRepo",
			args:             []string{"reg/img:unsigned-tag"},
			notaryRepository: notary.GetEmptyTargetsNotaryRepository,
			golden:           "trust-inspect-empty-repo.golden",
		},
		{
			doc:              "FullRepoWithoutSigners",
			args:             []string{"signed-repo"},
			notaryRepository: notary.GetLoadedWithNoSignersNotaryRepository,
			golden:           "trust-inspect-full-repo-no-signers.golden",
		},
		{
			doc:              "OneTagWithoutSigners",
			args:             []string{"signed-repo:green"},
			notaryRepository: notary.GetLoadedWithNoSignersNotaryRepository,
			golden:           "trust-inspect-one-tag-no-signers.golden",
		},
		{
			doc:              "FullRepoWithSigners",
			args:             []string{"signed-repo"},
			notaryRepository: notary.GetLoadedNotaryRepository,
			golden:           "trust-inspect-full-repo-with-signers.golden",
		},
		{
			doc:              "MultipleFullReposWithSigners",
			args:             []string{"signed-repo", "signed-repo"},
			notaryRepository: notary.GetLoadedNotaryRepository,
			golden:           "trust-inspect-multiple-repos-with-signers.golden",
		},
		{
			doc:              "UnsignedTagInSignedRepo",
			args:             []string{"signed-repo:unsigned"},
			notaryRepository: notary.GetLoadedNotaryRepository,
			golden:           "trust-inspect-unsigned-tag-with-signers.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			cli.SetNotaryClient(tc.notaryRepository)
			cmd := newInspectCommand(cli)
			cmd.SetArgs(tc.args)
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), tc.golden)
		})
	}
}
