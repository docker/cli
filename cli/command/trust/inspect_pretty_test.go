package trust

import (
	"encoding/hex"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/trust"
	"github.com/docker/cli/internal/test"
	notaryfake "github.com/docker/cli/internal/test/notary"
	dockerClient "github.com/docker/docker/client"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
)

// TODO(n4ss): remove common tests with the regular inspect command

type fakeClient struct {
	dockerClient.Client
}

func TestTrustInspectPrettyCommandErrors(t *testing.T) {
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
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		cmd.Flags().Set("pretty", "true")
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestTrustInspectPrettyCommandOfflineErrors(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetOfflineNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"nonexistent-reg-name.io/image"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "No signatures or cannot access nonexistent-reg-name.io/image")

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetOfflineNotaryRepository)
	cmd = newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"nonexistent-reg-name.io/image:tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "No signatures or cannot access nonexistent-reg-name.io/image")
}

func TestTrustInspectPrettyCommandUninitializedErrors(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetUninitializedNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"reg/unsigned-img"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "No signatures or cannot access reg/unsigned-img")

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetUninitializedNotaryRepository)
	cmd = newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"reg/unsigned-img:tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "No signatures or cannot access reg/unsigned-img:tag")
}

func TestTrustInspectPrettyCommandEmptyNotaryRepoErrors(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetEmptyTargetsNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"reg/img:unsigned-tag"})
	cmd.SetOutput(ioutil.Discard)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Contains(cli.OutBuffer().String(), "No signatures for reg/img:unsigned-tag"))
	assert.Check(t, is.Contains(cli.OutBuffer().String(), "Administrative keys for reg/img"))

	cli = test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetEmptyTargetsNotaryRepository)
	cmd = newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"reg/img"})
	cmd.SetOutput(ioutil.Discard)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Contains(cli.OutBuffer().String(), "No signatures for reg/img"))
	assert.Check(t, is.Contains(cli.OutBuffer().String(), "Administrative keys for reg/img"))
}

func TestTrustInspectPrettyCommandFullRepoWithoutSigners(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetLoadedWithNoSignersNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"signed-repo"})
	assert.NilError(t, cmd.Execute())

	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-pretty-full-repo-no-signers.golden")
}

func TestTrustInspectPrettyCommandOneTagWithoutSigners(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetLoadedWithNoSignersNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"signed-repo:green"})
	assert.NilError(t, cmd.Execute())

	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-pretty-one-tag-no-signers.golden")
}

func TestTrustInspectPrettyCommandFullRepoWithSigners(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetLoadedNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"signed-repo"})
	assert.NilError(t, cmd.Execute())

	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-pretty-full-repo-with-signers.golden")
}

func TestTrustInspectPrettyCommandUnsignedTagInSignedRepo(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetLoadedNotaryRepository)
	cmd := newInspectCommand(cli)
	cmd.Flags().Set("pretty", "true")
	cmd.SetArgs([]string{"signed-repo:unsigned"})
	assert.NilError(t, cmd.Execute())

	golden.Assert(t, cli.OutBuffer().String(), "trust-inspect-pretty-unsigned-tag-with-signers.golden")
}

func TestNotaryRoleToSigner(t *testing.T) {
	assert.Check(t, is.Equal(releasedRoleName, notaryRoleToSigner(data.CanonicalTargetsRole)))
	assert.Check(t, is.Equal(releasedRoleName, notaryRoleToSigner(trust.ReleasesRole)))
	assert.Check(t, is.Equal("signer", notaryRoleToSigner("targets/signer")))
	assert.Check(t, is.Equal("docker/signer", notaryRoleToSigner("targets/docker/signer")))

	// It's nonsense for other base roles to have signed off on a target, but this function leaves role names intact
	for _, role := range data.BaseRoles {
		if role == data.CanonicalTargetsRole {
			continue
		}
		assert.Check(t, is.Equal(role.String(), notaryRoleToSigner(role)))
	}
	assert.Check(t, is.Equal("notarole", notaryRoleToSigner(data.RoleName("notarole"))))
}

// check if a role name is "released": either targets/releases or targets TUF roles
func TestIsReleasedTarget(t *testing.T) {
	assert.Check(t, isReleasedTarget(trust.ReleasesRole))
	for _, role := range data.BaseRoles {
		assert.Check(t, is.Equal(role == data.CanonicalTargetsRole, isReleasedTarget(role)))
	}
	assert.Check(t, !isReleasedTarget(data.RoleName("targets/not-releases")))
	assert.Check(t, !isReleasedTarget(data.RoleName("random")))
	assert.Check(t, !isReleasedTarget(data.RoleName("targets/releases/subrole")))
}

// creates a mock delegation with a given name and no keys
func mockDelegationRoleWithName(name string) data.DelegationRole {
	baseRole := data.NewBaseRole(
		data.RoleName(name),
		notary.MinThreshold,
	)
	return data.DelegationRole{BaseRole: baseRole, Paths: []string{}}
}

func TestMatchEmptySignatures(t *testing.T) {
	// first try empty targets
	emptyTgts := []client.TargetSignedStruct{}

	matchedSigRows := matchReleasedSignatures(emptyTgts)
	assert.Check(t, is.Len(matchedSigRows, 0))
}

func TestMatchUnreleasedSignatures(t *testing.T) {
	// try an "unreleased" target with 3 signatures, 0 rows will appear
	unreleasedTgts := []client.TargetSignedStruct{}

	tgt := client.Target{Name: "unreleased", Hashes: data.Hashes{notary.SHA256: []byte("hash")}}
	for _, unreleasedRole := range []string{"targets/a", "targets/b", "targets/c"} {
		unreleasedTgts = append(unreleasedTgts, client.TargetSignedStruct{Role: mockDelegationRoleWithName(unreleasedRole), Target: tgt})
	}

	matchedSigRows := matchReleasedSignatures(unreleasedTgts)
	assert.Check(t, is.Len(matchedSigRows, 0))
}

func TestMatchOneReleasedSingleSignature(t *testing.T) {
	// now try only 1 "released" target with no additional sigs, 1 row will appear with 0 signers
	oneReleasedTgt := []client.TargetSignedStruct{}

	// make and append the "released" target to our mock input
	releasedTgt := client.Target{Name: "released", Hashes: data.Hashes{notary.SHA256: []byte("released-hash")}}
	oneReleasedTgt = append(oneReleasedTgt, client.TargetSignedStruct{Role: mockDelegationRoleWithName("targets/releases"), Target: releasedTgt})

	// make and append 3 non-released signatures on the "unreleased" target
	unreleasedTgt := client.Target{Name: "unreleased", Hashes: data.Hashes{notary.SHA256: []byte("hash")}}
	for _, unreleasedRole := range []string{"targets/a", "targets/b", "targets/c"} {
		oneReleasedTgt = append(oneReleasedTgt, client.TargetSignedStruct{Role: mockDelegationRoleWithName(unreleasedRole), Target: unreleasedTgt})
	}

	matchedSigRows := matchReleasedSignatures(oneReleasedTgt)
	assert.Check(t, is.Len(matchedSigRows, 1))

	outputRow := matchedSigRows[0]
	// Empty signers because "targets/releases" doesn't show up
	assert.Check(t, is.Len(outputRow.Signers, 0))
	assert.Check(t, is.Equal(releasedTgt.Name, outputRow.SignedTag))
	assert.Check(t, is.Equal(hex.EncodeToString(releasedTgt.Hashes[notary.SHA256]), outputRow.Digest))
}

func TestMatchOneReleasedMultiSignature(t *testing.T) {
	// now try only 1 "released" target with 3 additional sigs, 1 row will appear with 3 signers
	oneReleasedTgt := []client.TargetSignedStruct{}

	// make and append the "released" target to our mock input
	releasedTgt := client.Target{Name: "released", Hashes: data.Hashes{notary.SHA256: []byte("released-hash")}}
	oneReleasedTgt = append(oneReleasedTgt, client.TargetSignedStruct{Role: mockDelegationRoleWithName("targets/releases"), Target: releasedTgt})

	// make and append 3 non-released signatures on both the "released" and "unreleased" targets
	unreleasedTgt := client.Target{Name: "unreleased", Hashes: data.Hashes{notary.SHA256: []byte("hash")}}
	for _, unreleasedRole := range []string{"targets/a", "targets/b", "targets/c"} {
		oneReleasedTgt = append(oneReleasedTgt, client.TargetSignedStruct{Role: mockDelegationRoleWithName(unreleasedRole), Target: unreleasedTgt})
		oneReleasedTgt = append(oneReleasedTgt, client.TargetSignedStruct{Role: mockDelegationRoleWithName(unreleasedRole), Target: releasedTgt})
	}

	matchedSigRows := matchReleasedSignatures(oneReleasedTgt)
	assert.Check(t, is.Len(matchedSigRows, 1))

	outputRow := matchedSigRows[0]
	// We should have three signers
	assert.Check(t, is.DeepEqual(outputRow.Signers, []string{"a", "b", "c"}))
	assert.Check(t, is.Equal(releasedTgt.Name, outputRow.SignedTag))
	assert.Check(t, is.Equal(hex.EncodeToString(releasedTgt.Hashes[notary.SHA256]), outputRow.Digest))
}

func TestMatchMultiReleasedMultiSignature(t *testing.T) {
	// now try 3 "released" targets with additional sigs to show 3 rows as follows:
	// target-a is signed by targets/releases and targets/a - a will be the signer
	// target-b is signed by targets/releases, targets/a, targets/b - a and b will be the signers
	// target-c is signed by targets/releases, targets/a, targets/b, targets/c - a, b, and c will be the signers
	multiReleasedTgts := []client.TargetSignedStruct{}
	// make target-a, target-b, and target-c
	targetA := client.Target{Name: "target-a", Hashes: data.Hashes{notary.SHA256: []byte("target-a-hash")}}
	targetB := client.Target{Name: "target-b", Hashes: data.Hashes{notary.SHA256: []byte("target-b-hash")}}
	targetC := client.Target{Name: "target-c", Hashes: data.Hashes{notary.SHA256: []byte("target-c-hash")}}

	// have targets/releases "sign" on all of these targets so they are released
	multiReleasedTgts = append(multiReleasedTgts, client.TargetSignedStruct{Role: mockDelegationRoleWithName("targets/releases"), Target: targetA})
	multiReleasedTgts = append(multiReleasedTgts, client.TargetSignedStruct{Role: mockDelegationRoleWithName("targets/releases"), Target: targetB})
	multiReleasedTgts = append(multiReleasedTgts, client.TargetSignedStruct{Role: mockDelegationRoleWithName("targets/releases"), Target: targetC})

	// targets/a signs off on all three targets (target-a, target-b, target-c):
	for _, tgt := range []client.Target{targetA, targetB, targetC} {
		multiReleasedTgts = append(multiReleasedTgts, client.TargetSignedStruct{Role: mockDelegationRoleWithName("targets/a"), Target: tgt})
	}

	// targets/b signs off on the final two targets (target-b, target-c):
	for _, tgt := range []client.Target{targetB, targetC} {
		multiReleasedTgts = append(multiReleasedTgts, client.TargetSignedStruct{Role: mockDelegationRoleWithName("targets/b"), Target: tgt})
	}

	// targets/c only signs off on the last target (target-c):
	multiReleasedTgts = append(multiReleasedTgts, client.TargetSignedStruct{Role: mockDelegationRoleWithName("targets/c"), Target: targetC})

	matchedSigRows := matchReleasedSignatures(multiReleasedTgts)
	assert.Check(t, is.Len(matchedSigRows, 3))

	// note that the output is sorted by tag name, so we can reliably index to validate data:
	outputTargetA := matchedSigRows[0]
	assert.Check(t, is.DeepEqual(outputTargetA.Signers, []string{"a"}))
	assert.Check(t, is.Equal(targetA.Name, outputTargetA.SignedTag))
	assert.Check(t, is.Equal(hex.EncodeToString(targetA.Hashes[notary.SHA256]), outputTargetA.Digest))

	outputTargetB := matchedSigRows[1]
	assert.Check(t, is.DeepEqual(outputTargetB.Signers, []string{"a", "b"}))
	assert.Check(t, is.Equal(targetB.Name, outputTargetB.SignedTag))
	assert.Check(t, is.Equal(hex.EncodeToString(targetB.Hashes[notary.SHA256]), outputTargetB.Digest))

	outputTargetC := matchedSigRows[2]
	assert.Check(t, is.DeepEqual(outputTargetC.Signers, []string{"a", "b", "c"}))
	assert.Check(t, is.Equal(targetC.Name, outputTargetC.SignedTag))
	assert.Check(t, is.Equal(hex.EncodeToString(targetC.Hashes[notary.SHA256]), outputTargetC.Digest))
}

func TestMatchReleasedSignatureFromTargets(t *testing.T) {
	// now try only 1 "released" target with no additional sigs, one rows will appear
	oneReleasedTgt := []client.TargetSignedStruct{}
	// make and append the "released" target to our mock input
	releasedTgt := client.Target{Name: "released", Hashes: data.Hashes{notary.SHA256: []byte("released-hash")}}
	oneReleasedTgt = append(oneReleasedTgt, client.TargetSignedStruct{Role: mockDelegationRoleWithName(data.CanonicalTargetsRole.String()), Target: releasedTgt})
	matchedSigRows := matchReleasedSignatures(oneReleasedTgt)
	assert.Check(t, is.Len(matchedSigRows, 1))
	outputRow := matchedSigRows[0]
	// Empty signers because "targets" doesn't show up
	assert.Check(t, is.Len(outputRow.Signers, 0))
	assert.Check(t, is.Equal(releasedTgt.Name, outputRow.SignedTag))
	assert.Check(t, is.Equal(hex.EncodeToString(releasedTgt.Hashes[notary.SHA256]), outputRow.Digest))
}

func TestGetSignerRolesWithKeyIDs(t *testing.T) {
	roles := []data.Role{
		{
			RootRole: data.RootRole{
				KeyIDs: []string{"key11"},
			},
			Name: "targets/alice",
		},
		{
			RootRole: data.RootRole{
				KeyIDs: []string{"key21", "key22"},
			},
			Name: "targets/releases",
		},
		{
			RootRole: data.RootRole{
				KeyIDs: []string{"key31"},
			},
			Name: data.CanonicalTargetsRole,
		},
		{
			RootRole: data.RootRole{
				KeyIDs: []string{"key41", "key01"},
			},
			Name: data.CanonicalRootRole,
		},
		{
			RootRole: data.RootRole{
				KeyIDs: []string{"key51"},
			},
			Name: data.CanonicalSnapshotRole,
		},
		{
			RootRole: data.RootRole{
				KeyIDs: []string{"key61"},
			},
			Name: data.CanonicalTimestampRole,
		},
		{
			RootRole: data.RootRole{
				KeyIDs: []string{"key71", "key72"},
			},
			Name: "targets/bob",
		},
	}
	expectedSignerRoleToKeyIDs := map[string][]string{
		"alice": {"key11"},
		"bob":   {"key71", "key72"},
	}

	var roleWithSigs []client.RoleWithSignatures
	for _, role := range roles {
		roleWithSig := client.RoleWithSignatures{Role: role, Signatures: nil}
		roleWithSigs = append(roleWithSigs, roleWithSig)
	}
	signerRoleToKeyIDs := getDelegationRoleToKeyMap(roles)
	assert.Check(t, is.DeepEqual(expectedSignerRoleToKeyIDs, signerRoleToKeyIDs))
}

func TestFormatAdminRole(t *testing.T) {
	aliceRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs: []string{"key11"},
		},
		Name: "targets/alice",
	}
	aliceRoleWithSigs := client.RoleWithSignatures{Role: aliceRole, Signatures: nil}
	assert.Check(t, is.Equal("", formatAdminRole(aliceRoleWithSigs)))

	releasesRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs: []string{"key11"},
		},
		Name: "targets/releases",
	}
	releasesRoleWithSigs := client.RoleWithSignatures{Role: releasesRole, Signatures: nil}
	assert.Check(t, is.Equal("", formatAdminRole(releasesRoleWithSigs)))

	timestampRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs: []string{"key11"},
		},
		Name: data.CanonicalTimestampRole,
	}
	timestampRoleWithSigs := client.RoleWithSignatures{Role: timestampRole, Signatures: nil}
	assert.Check(t, is.Equal("", formatAdminRole(timestampRoleWithSigs)))

	snapshotRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs: []string{"key11"},
		},
		Name: data.CanonicalSnapshotRole,
	}
	snapshotRoleWithSigs := client.RoleWithSignatures{Role: snapshotRole, Signatures: nil}
	assert.Check(t, is.Equal("", formatAdminRole(snapshotRoleWithSigs)))

	rootRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs: []string{"key11"},
		},
		Name: data.CanonicalRootRole,
	}
	rootRoleWithSigs := client.RoleWithSignatures{Role: rootRole, Signatures: nil}
	assert.Check(t, is.Equal("Root Key:\tkey11\n", formatAdminRole(rootRoleWithSigs)))

	targetsRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs: []string{"key99", "abc", "key11"},
		},
		Name: data.CanonicalTargetsRole,
	}
	targetsRoleWithSigs := client.RoleWithSignatures{Role: targetsRole, Signatures: nil}
	assert.Check(t, is.Equal("Repository Key:\tabc, key11, key99\n", formatAdminRole(targetsRoleWithSigs)))
}
