package trust

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/client/changelist"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
	"github.com/theupdateframework/notary/tuf/data"
)

const passwd = "password"

func TestTrustSignCommandErrors(t *testing.T) {
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
			args:          []string{"image", "tag"},
			expectedError: "requires exactly 1 argument",
		},
		{
			name:          "sha-reference",
			args:          []string{"870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd"},
			expectedError: "invalid repository name",
		},
		{
			name:          "invalid-img-reference",
			args:          []string{"ALPINE:latest"},
			expectedError: "invalid reference format",
		},
		{
			name:          "no-tag",
			args:          []string{"reg/img"},
			expectedError: "No tag specified for reg/img",
		},
		{
			name:          "digest-reference",
			args:          []string{"ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2"},
			expectedError: "cannot use a digest reference for IMAGE:TAG",
		},
	}
	// change to a tmpdir
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)
	for _, tc := range testCases {
		cmd := newSignCommand(
			test.NewFakeCli(&fakeClient{}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestTrustSignCommandOfflineErrors(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getOfflineNotaryRepository)
	cmd := newSignCommand(cli)
	cmd.SetArgs([]string{"reg-name.io/image:tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.Error(t, cmd.Execute())
	testutil.ErrorContains(t, cmd.Execute(), "client is offline")
}

func TestGetOrGenerateNotaryKey(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)

	// repo is empty, try making a root key
	rootKeyA, err := getOrGenerateNotaryKey(notaryRepo, data.CanonicalRootRole)
	assert.NoError(t, err)
	assert.NotNil(t, rootKeyA)

	// we should only have one newly generated key
	allKeys := notaryRepo.GetCryptoService().ListAllKeys()
	assert.Len(t, allKeys, 1)
	assert.NotNil(t, notaryRepo.GetCryptoService().GetKey(rootKeyA.ID()))

	// this time we should get back the same key if we ask for another root key
	rootKeyB, err := getOrGenerateNotaryKey(notaryRepo, data.CanonicalRootRole)
	assert.NoError(t, err)
	assert.NotNil(t, rootKeyB)

	// we should only have one newly generated key
	allKeys = notaryRepo.GetCryptoService().ListAllKeys()
	assert.Len(t, allKeys, 1)
	assert.NotNil(t, notaryRepo.GetCryptoService().GetKey(rootKeyB.ID()))

	// The key we retrieved should be identical to the one we generated
	assert.Equal(t, rootKeyA, rootKeyB)

	// Now also try with a delegation key
	releasesKey, err := getOrGenerateNotaryKey(notaryRepo, data.RoleName(trust.ReleasesRole))
	assert.NoError(t, err)
	assert.NotNil(t, releasesKey)

	// we should now have two keys
	allKeys = notaryRepo.GetCryptoService().ListAllKeys()
	assert.Len(t, allKeys, 2)
	assert.NotNil(t, notaryRepo.GetCryptoService().GetKey(releasesKey.ID()))
	// The key we retrieved should be identical to the one we generated
	assert.NotEqual(t, releasesKey, rootKeyA)
	assert.NotEqual(t, releasesKey, rootKeyB)
}

func TestAddStageSigners(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)

	// stage targets/user
	userRole := data.RoleName("targets/user")
	userKey := data.NewPublicKey("algoA", []byte("a"))
	err = addStagedSigner(notaryRepo, userRole, []data.PublicKey{userKey})
	assert.NoError(t, err)
	// check the changelist for four total changes: two on targets/releases and two on targets/user
	cl, err := notaryRepo.GetChangelist()
	assert.NoError(t, err)
	changeList := cl.List()
	assert.Len(t, changeList, 4)
	// ordering is determinstic:

	// first change is for targets/user key creation
	newSignerKeyChange := changeList[0]
	expectedJSON, err := json.Marshal(&changelist.TUFDelegation{
		NewThreshold: notary.MinThreshold,
		AddKeys:      data.KeyList([]data.PublicKey{userKey}),
	})
	require.NoError(t, err)
	expectedChange := changelist.NewTUFChange(
		changelist.ActionCreate,
		userRole,
		changelist.TypeTargetsDelegation,
		"", // no path for delegations
		expectedJSON,
	)
	assert.Equal(t, expectedChange, newSignerKeyChange)

	// second change is for targets/user getting all paths
	newSignerPathsChange := changeList[1]
	expectedJSON, err = json.Marshal(&changelist.TUFDelegation{
		AddPaths: []string{""},
	})
	require.NoError(t, err)
	expectedChange = changelist.NewTUFChange(
		changelist.ActionCreate,
		userRole,
		changelist.TypeTargetsDelegation,
		"", // no path for delegations
		expectedJSON,
	)
	assert.Equal(t, expectedChange, newSignerPathsChange)

	releasesRole := data.RoleName("targets/releases")

	// third change is for targets/releases key creation
	releasesKeyChange := changeList[2]
	expectedJSON, err = json.Marshal(&changelist.TUFDelegation{
		NewThreshold: notary.MinThreshold,
		AddKeys:      data.KeyList([]data.PublicKey{userKey}),
	})
	require.NoError(t, err)
	expectedChange = changelist.NewTUFChange(
		changelist.ActionCreate,
		releasesRole,
		changelist.TypeTargetsDelegation,
		"", // no path for delegations
		expectedJSON,
	)
	assert.Equal(t, expectedChange, releasesKeyChange)

	// fourth change is for targets/releases getting all paths
	releasesPathsChange := changeList[3]
	expectedJSON, err = json.Marshal(&changelist.TUFDelegation{
		AddPaths: []string{""},
	})
	require.NoError(t, err)
	expectedChange = changelist.NewTUFChange(
		changelist.ActionCreate,
		releasesRole,
		changelist.TypeTargetsDelegation,
		"", // no path for delegations
		expectedJSON,
	)
	assert.Equal(t, expectedChange, releasesPathsChange)
}

func TestGetSignedManifestHashAndSize(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)
	target := &client.Target{}
	target.Hashes, target.Length, err = getSignedManifestHashAndSize(notaryRepo, "test")
	assert.EqualError(t, err, "client is offline")
}

func TestGetReleasedTargetHashAndSize(t *testing.T) {
	oneReleasedTgt := []client.TargetSignedStruct{}
	// make and append 3 non-released signatures on the "unreleased" target
	unreleasedTgt := client.Target{Name: "unreleased", Hashes: data.Hashes{notary.SHA256: []byte("hash")}}
	for _, unreleasedRole := range []string{"targets/a", "targets/b", "targets/c"} {
		oneReleasedTgt = append(oneReleasedTgt, client.TargetSignedStruct{Role: mockDelegationRoleWithName(unreleasedRole), Target: unreleasedTgt})
	}
	_, _, err := getReleasedTargetHashAndSize(oneReleasedTgt, "unreleased")
	assert.EqualError(t, err, "No valid trust data for unreleased")
	releasedTgt := client.Target{Name: "released", Hashes: data.Hashes{notary.SHA256: []byte("released-hash")}}
	oneReleasedTgt = append(oneReleasedTgt, client.TargetSignedStruct{Role: mockDelegationRoleWithName("targets/releases"), Target: releasedTgt})
	hash, _, _ := getReleasedTargetHashAndSize(oneReleasedTgt, "unreleased")
	assert.Equal(t, data.Hashes{notary.SHA256: []byte("released-hash")}, hash)

}

func TestCreateTarget(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)
	_, err = createTarget(notaryRepo, "")
	assert.EqualError(t, err, "No tag specified")
	_, err = createTarget(notaryRepo, "1")
	assert.EqualError(t, err, "client is offline")
}

func TestGetExistingSignatureInfoForReleasedTag(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)
	_, err = getExistingSignatureInfoForReleasedTag(notaryRepo, "test")
	assert.EqualError(t, err, "client is offline")
}

func TestPrettyPrintExistingSignatureInfo(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	signers := []string{"Bob", "Alice", "Carol"}
	existingSig := trustTagRow{trustTagKey{"tagName", "abc123"}, signers}
	prettyPrintExistingSignatureInfo(buf, existingSig)

	assert.Contains(t, buf.String(), "Existing signatures for tag tagName digest abc123 from:\nAlice, Bob, Carol")
}

func TestSignCommandChangeListIsCleanedOnError(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config.SetDir(tmpDir)
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getLoadedNotaryRepository)
	cmd := newSignCommand(cli)
	cmd.SetArgs([]string{"ubuntu:latest"})
	cmd.SetOutput(ioutil.Discard)

	err = cmd.Execute()
	require.Error(t, err)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "docker.io/library/ubuntu", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)
	cl, err := notaryRepo.GetChangelist()
	require.NoError(t, err)
	assert.Equal(t, len(cl.List()), 0)
}

func TestSignCommandLocalFlag(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getEmptyTargetsNotaryRepository)
	cmd := newSignCommand(cli)
	cmd.SetArgs([]string{"--local", "reg-name.io/image:red"})
	cmd.SetOutput(ioutil.Discard)
	testutil.ErrorContains(t, cmd.Execute(), "error during connect: Get /images/reg-name.io/image:red/json: unsupported protocol scheme")

}
