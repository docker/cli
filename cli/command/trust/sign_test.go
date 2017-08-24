package trust

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/notary"
	"github.com/docker/notary/client"
	"github.com/docker/notary/client/changelist"
	"github.com/docker/notary/passphrase"
	"github.com/docker/notary/trustpinning"
	"github.com/docker/notary/tuf/data"
	"github.com/stretchr/testify/assert"
)

const passwd = "password"

func TestTrustSignErrors(t *testing.T) {
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
			name:          "nonexistent-reg",
			args:          []string{"nonexistent-reg-name.io/image:tag"},
			expectedError: "no such host",
		},
		{
			name:          "invalid-img-reference",
			args:          []string{"ALPINE:latest"},
			expectedError: "invalid reference format",
		},
		{
			name:          "no-shell-for-passwd",
			args:          []string{"riyaz/unsigned-img:latest"},
			expectedError: "error during connect: Get /images/riyaz/unsigned-img:latest/json",
		},
		{
			name:          "no-tag",
			args:          []string{"riyaz/unsigned-img"},
			expectedError: "No tag specified for riyaz/unsigned-img",
		},
		{
			name:          "digest-reference",
			args:          []string{"ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2"},
			expectedError: "cannot use a digest reference for IMAGE:TAG",
		},
		{
			name:          "no-keys",
			args:          []string{"ubuntu:latest"},
			expectedError: "failed to sign \"docker.io/library/ubuntu\":latest - you are not authorized to perform this operation: server returned 401.",
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

func TestGetOrGenerateNotaryKey(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedNotaryRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)

	// repo is empty, try making a root key
	rootKeyA, err := getOrGenerateNotaryKey(notaryRepo, data.CanonicalRootRole)
	assert.NoError(t, err)
	assert.NotNil(t, rootKeyA)

	// we should only have one newly generated key
	allKeys := notaryRepo.CryptoService.ListAllKeys()
	assert.Len(t, allKeys, 1)
	assert.NotNil(t, notaryRepo.CryptoService.GetKey(rootKeyA.ID()))

	// this time we should get back the same key if we ask for another root key
	rootKeyB, err := getOrGenerateNotaryKey(notaryRepo, data.CanonicalRootRole)
	assert.NoError(t, err)
	assert.NotNil(t, rootKeyB)

	// we should only have one newly generated key
	allKeys = notaryRepo.CryptoService.ListAllKeys()
	assert.Len(t, allKeys, 1)
	assert.NotNil(t, notaryRepo.CryptoService.GetKey(rootKeyB.ID()))

	// The key we retrieved should be identical to the one we generated
	assert.Equal(t, rootKeyA, rootKeyB)

	// Now also try with a delegation key
	releasesKey, err := getOrGenerateNotaryKey(notaryRepo, data.RoleName(trust.ReleasesRole))
	assert.NoError(t, err)
	assert.NotNil(t, releasesKey)

	// we should now have two keys
	allKeys = notaryRepo.CryptoService.ListAllKeys()
	assert.Len(t, allKeys, 2)
	assert.NotNil(t, notaryRepo.CryptoService.GetKey(releasesKey.ID()))
	// The key we retrieved should be identical to the one we generated
	assert.NotEqual(t, releasesKey, rootKeyA)
	assert.NotEqual(t, releasesKey, rootKeyB)
}

func TestAddStageSigners(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedNotaryRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)

	// stage targets/user
	userRole := data.RoleName("targets/user")
	userKey := data.NewPublicKey("algoA", []byte("a"))
	addStagedSigner(notaryRepo, userRole, []data.PublicKey{userKey})
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

	notaryRepo, err := client.NewFileCachedNotaryRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
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

	notaryRepo, err := client.NewFileCachedNotaryRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
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

	notaryRepo, err := client.NewFileCachedNotaryRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)
	_, err = getExistingSignatureInfoForReleasedTag(notaryRepo, "test")
	assert.EqualError(t, err, "client is offline")
}

func TestPrettyPrintExistingSignatureInfo(t *testing.T) {
	fakeCli := test.NewFakeCli(&fakeClient{})

	signers := []string{"Bob", "Alice", "Carol"}
	existingSig := trustTagRow{trustTagKey{"tagName", "abc123"}, signers}
	prettyPrintExistingSignatureInfo(fakeCli, existingSig)

	assert.Contains(t, fakeCli.OutBuffer().String(), "Existing signatures for tag tagName digest abc123 from:\nAlice, Bob, Carol")
}

func TestChangeList(t *testing.T) {

	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)
	cmd := newSignCommand(
		test.NewFakeCli(&fakeClient{}))
	cmd.SetArgs([]string{"ubuntu:latest"})
	cmd.SetOutput(ioutil.Discard)
	err = cmd.Execute()
	notaryRepo, err := client.NewFileCachedNotaryRepository(tmpDir, "docker.io/library/ubuntu", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)
	cl, err := notaryRepo.GetChangelist()
	assert.Equal(t, len(cl.List()), 0)
}
