package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/output"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
)

const (
	notary = "/usr/local/bin/notary"

	pubkey1  = "./testdata/notary/delgkey1.crt"
	privkey1 = "./testdata/notary/delgkey1.key"
	pubkey2  = "./testdata/notary/delgkey2.crt"
	privkey2 = "./testdata/notary/delgkey2.key"
	pubkey3  = "./testdata/notary/delgkey3.crt"
	privkey3 = "./testdata/notary/delgkey3.key"
	pubkey4  = "./testdata/notary/delgkey4.crt"
	privkey4 = "./testdata/notary/delgkey4.key"
)

func TestPushWithContentTrust(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := createImage(t, registryPrefix, "trust-push", "latest")

	result := icmd.RunCmd(icmd.Command("docker", "push", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
		fixtures.WithPassphrase("foo", "bar"),
	)
	result.Assert(t, icmd.Success)
	golden.Assert(t, result.Stderr(), "push-with-content-trust-err.golden")
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0: output.Equals("The push refers to repository [registry:5000/trust-push]"),
		1: output.Equals("5bef08742407: Preparing"),
		3: output.Equals("latest: digest: sha256:641b95ddb2ea9dc2af1a0113b6b348ebc20872ba615204fbe12148e98fd6f23d size: 528"),
		4: output.Equals("Signing and pushing trust metadata"),
		5: output.Equals(`Finished initializing "registry:5000/trust-push"`),
		6: output.Equals("Successfully signed registry:5000/trust-push:latest"),
	})
}

func TestPushWithContentTrustUnreachableServer(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := createImage(t, registryPrefix, "trust-push-unreachable", "latest")

	result := icmd.RunCmd(icmd.Command("docker", "push", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotaryServer("https://invalidnotaryserver"),
	)
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "error contacting notary server",
	})
}

func TestPushWithContentTrustExistingTag(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := createImage(t, registryPrefix, "trust-push-existing", "latest")

	result := icmd.RunCmd(icmd.Command("docker", "push", image))
	result.Assert(t, icmd.Success)

	result = icmd.RunCmd(icmd.Command("docker", "push", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
		fixtures.WithPassphrase("foo", "bar"),
	)
	result.Assert(t, icmd.Expected{
		Out: "Signing and pushing trust metadata",
	})

	// Re-push
	result = icmd.RunCmd(icmd.Command("docker", "push", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
		fixtures.WithPassphrase("foo", "bar"),
	)
	result.Assert(t, icmd.Expected{
		Out: "Signing and pushing trust metadata",
	})
}

func TestPushWithContentTrustReleasesDelegationOnly(t *testing.T) {
	role := "targets/releases"

	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	copyPrivateKey(t, dir.Join("trust", "private"), privkey1)
	notaryDir := setupNotaryConfig(t, dir)
	defer notaryDir.Remove()
	homeDir := fs.NewDir(t, "push_test_home")
	defer notaryDir.Remove()

	baseRef := fmt.Sprintf("%s/%s", registryPrefix, "trust-push-releases-delegation")
	targetRef := fmt.Sprintf("%s:%s", baseRef, "latest")

	// Init repository
	notaryInit(t, notaryDir, homeDir, baseRef)
	// Add delegation key (public key)
	notaryAddDelegation(t, notaryDir, homeDir, baseRef, role, pubkey1)
	// Publish it
	notaryPublish(t, notaryDir, homeDir, baseRef)
	// Import private key
	notaryImportPrivateKey(t, notaryDir, homeDir, baseRef, role, privkey1)

	// Tag & push with content trust
	icmd.RunCommand("docker", "pull", fixtures.AlpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, targetRef).Assert(t, icmd.Success)
	result := icmd.RunCmd(icmd.Command("docker", "push", targetRef),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
		fixtures.WithPassphrase("foo", "foo"),
	)
	result.Assert(t, icmd.Expected{
		Out: "Signing and pushing trust metadata",
	})

	targetsInRole := notaryListTargetsInRole(t, notaryDir, homeDir, baseRef, role)
	assert.Assert(t, targetsInRole["latest"] == role, "%v", targetsInRole)
	targetsInRole = notaryListTargetsInRole(t, notaryDir, homeDir, baseRef, "targets")
	assert.Assert(t, targetsInRole["latest"] != "targets", "%v", targetsInRole)

	result = icmd.RunCmd(icmd.Command("docker", "pull", targetRef),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Success)
}

func TestPushWithContentTrustSignsAllFirstLevelRolesWeHaveKeysFor(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	copyPrivateKey(t, dir.Join("trust", "private"), privkey1)
	copyPrivateKey(t, dir.Join("trust", "private"), privkey2)
	copyPrivateKey(t, dir.Join("trust", "private"), privkey3)
	notaryDir := setupNotaryConfig(t, dir)
	defer notaryDir.Remove()
	homeDir := fs.NewDir(t, "push_test_home")
	defer notaryDir.Remove()

	baseRef := fmt.Sprintf("%s/%s", registryPrefix, "trust-push-releases-first-roles")
	targetRef := fmt.Sprintf("%s:%s", baseRef, "latest")

	// Init repository
	notaryInit(t, notaryDir, homeDir, baseRef)
	// Add delegation key (public key)
	notaryAddDelegation(t, notaryDir, homeDir, baseRef, "targets/role1", pubkey1)
	notaryAddDelegation(t, notaryDir, homeDir, baseRef, "targets/role2", pubkey2)
	notaryAddDelegation(t, notaryDir, homeDir, baseRef, "targets/role3", pubkey3)
	notaryAddDelegation(t, notaryDir, homeDir, baseRef, "targets/role1/subrole", pubkey3)
	// Import private key
	notaryImportPrivateKey(t, notaryDir, homeDir, baseRef, "targets/role1", privkey1)
	notaryImportPrivateKey(t, notaryDir, homeDir, baseRef, "targets/role2", privkey2)
	notaryImportPrivateKey(t, notaryDir, homeDir, baseRef, "targets/role1/subrole", privkey3)
	// Publish it
	notaryPublish(t, notaryDir, homeDir, baseRef)

	// Tag & push with content trust
	icmd.RunCommand("docker", "pull", fixtures.AlpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, targetRef).Assert(t, icmd.Success)
	result := icmd.RunCmd(icmd.Command("docker", "push", targetRef),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
		fixtures.WithPassphrase("foo", "foo"),
	)
	result.Assert(t, icmd.Expected{
		Out: "Signing and pushing trust metadata",
	})

	// check to make sure that the target has been added to targets/role1 and targets/role2, and
	// not targets (because there are delegations) or targets/role3 (due to missing key) or
	// targets/role1/subrole (due to it being a second level delegation)
	targetsInRole := notaryListTargetsInRole(t, notaryDir, homeDir, baseRef, "targets/role1")
	assert.Assert(t, targetsInRole["latest"] == "targets/role1", "%v", targetsInRole)
	targetsInRole = notaryListTargetsInRole(t, notaryDir, homeDir, baseRef, "targets/role2")
	assert.Assert(t, targetsInRole["latest"] == "targets/role2", "%v", targetsInRole)
	targetsInRole = notaryListTargetsInRole(t, notaryDir, homeDir, baseRef, "targets")
	assert.Assert(t, targetsInRole["latest"] != "targets", "%v", targetsInRole)

	assert.NilError(t, os.RemoveAll(filepath.Join(dir.Join("trust"))))
	// Try to pull, should fail because non of these are the release role
	// FIXME(vdemeester) should be unit test
	result = icmd.RunCmd(icmd.Command("docker", "pull", targetRef),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
	})
}

func TestPushWithContentTrustSignsForRolesWithKeysAndValidPaths(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	copyPrivateKey(t, dir.Join("trust", "private"), privkey1)
	copyPrivateKey(t, dir.Join("trust", "private"), privkey2)
	copyPrivateKey(t, dir.Join("trust", "private"), privkey3)
	copyPrivateKey(t, dir.Join("trust", "private"), privkey4)
	notaryDir := setupNotaryConfig(t, dir)
	defer notaryDir.Remove()
	homeDir := fs.NewDir(t, "push_test_home")
	defer notaryDir.Remove()

	baseRef := fmt.Sprintf("%s/%s", registryPrefix, "trust-push-releases-keys-valid-paths")
	targetRef := fmt.Sprintf("%s:%s", baseRef, "latest")

	// Init repository
	notaryInit(t, notaryDir, homeDir, baseRef)
	// Add delegation key (public key)
	notaryAddDelegation(t, notaryDir, homeDir, baseRef, "targets/role1", pubkey1, "l", "z")
	notaryAddDelegation(t, notaryDir, homeDir, baseRef, "targets/role2", pubkey2, "x", "y")
	notaryAddDelegation(t, notaryDir, homeDir, baseRef, "targets/role3", pubkey3, "latest")
	notaryAddDelegation(t, notaryDir, homeDir, baseRef, "targets/role4", pubkey4, "latest")
	// Import private keys (except 3rd key)
	notaryImportPrivateKey(t, notaryDir, homeDir, baseRef, "targets/role1", privkey1)
	notaryImportPrivateKey(t, notaryDir, homeDir, baseRef, "targets/role2", privkey2)
	notaryImportPrivateKey(t, notaryDir, homeDir, baseRef, "targets/role4", privkey4)
	// Publish it
	notaryPublish(t, notaryDir, homeDir, baseRef)

	// Tag & push with content trust
	icmd.RunCommand("docker", "pull", fixtures.AlpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, targetRef).Assert(t, icmd.Success)
	result := icmd.RunCmd(icmd.Command("docker", "push", targetRef),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
		fixtures.WithPassphrase("foo", "foo"),
	)
	result.Assert(t, icmd.Expected{
		Out: "Signing and pushing trust metadata",
	})

	// check to make sure that the target has been added to targets/role1 and targets/role4, and
	// not targets (because there are delegations) or targets/role2 (due to path restrictions) or
	// targets/role3 (due to missing key)
	targetsInRole := notaryListTargetsInRole(t, notaryDir, homeDir, baseRef, "targets/role1")
	assert.Assert(t, targetsInRole["latest"] == "targets/role1", "%v", targetsInRole)
	targetsInRole = notaryListTargetsInRole(t, notaryDir, homeDir, baseRef, "targets/role4")
	assert.Assert(t, targetsInRole["latest"] == "targets/role4", "%v", targetsInRole)
	targetsInRole = notaryListTargetsInRole(t, notaryDir, homeDir, baseRef, "targets")
	assert.Assert(t, targetsInRole["latest"] != "targets", "%v", targetsInRole)

	assert.NilError(t, os.RemoveAll(filepath.Join(dir.Join("trust"))))
	// Try to pull, should fail because non of these are the release role
	// FIXME(vdemeester) should be unit test
	result = icmd.RunCmd(icmd.Command("docker", "pull", targetRef),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
	})
}

func createImage(t *testing.T, registryPrefix, repo, tag string) string {
	image := fmt.Sprintf("%s/%s:%s", registryPrefix, repo, tag)
	icmd.RunCommand("docker", "pull", fixtures.AlpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, image).Assert(t, icmd.Success)
	return image
}

func withNotaryPassphrase(pwd string) func(*icmd.Cmd) {
	return func(c *icmd.Cmd) {
		c.Env = append(c.Env, []string{
			fmt.Sprintf("NOTARY_ROOT_PASSPHRASE=%s", pwd),
			fmt.Sprintf("NOTARY_TARGETS_PASSPHRASE=%s", pwd),
			fmt.Sprintf("NOTARY_SNAPSHOT_PASSPHRASE=%s", pwd),
			fmt.Sprintf("NOTARY_DELEGATION_PASSPHRASE=%s", pwd),
		}...)
	}
}

func notaryImportPrivateKey(t *testing.T, notaryDir, homeDir *fs.Dir, baseRef, role, privkey string) {
	icmd.RunCmd(
		icmd.Command(notary, "-c", notaryDir.Join("client-config.json"), "key", "import", privkey, "-g", baseRef, "-r", role),
		withNotaryPassphrase("foo"),
		fixtures.WithHome(homeDir.Path()),
	).Assert(t, icmd.Success)
}

func notaryPublish(t *testing.T, notaryDir, homeDir *fs.Dir, baseRef string) {
	icmd.RunCmd(
		icmd.Command(notary, "-c", notaryDir.Join("client-config.json"), "publish", baseRef),
		withNotaryPassphrase("foo"),
		fixtures.WithHome(homeDir.Path()),
	).Assert(t, icmd.Success)
}

func notaryAddDelegation(t *testing.T, notaryDir, homeDir *fs.Dir, baseRef, role, pubkey string, paths ...string) {
	pathsArg := "--all-paths"
	if len(paths) > 0 {
		pathsArg = "--paths=" + strings.Join(paths, ",")
	}
	icmd.RunCmd(
		icmd.Command(notary, "-c", notaryDir.Join("client-config.json"), "delegation", "add", baseRef, role, pubkey, pathsArg),
		withNotaryPassphrase("foo"),
		fixtures.WithHome(homeDir.Path()),
	).Assert(t, icmd.Success)
}

func notaryInit(t *testing.T, notaryDir, homeDir *fs.Dir, baseRef string) {
	icmd.RunCmd(
		icmd.Command(notary, "-c", notaryDir.Join("client-config.json"), "init", baseRef),
		withNotaryPassphrase("foo"),
		fixtures.WithHome(homeDir.Path()),
	).Assert(t, icmd.Success)
}

func notaryListTargetsInRole(t *testing.T, notaryDir, homeDir *fs.Dir, baseRef, role string) map[string]string {
	result := icmd.RunCmd(
		icmd.Command(notary, "-c", notaryDir.Join("client-config.json"), "list", baseRef, "-r", role),
		fixtures.WithHome(homeDir.Path()),
	)
	out := result.Combined()

	// should look something like:
	//    NAME                                 DIGEST                                SIZE (BYTES)    ROLE
	// ------------------------------------------------------------------------------------------------------
	//   latest   24a36bbc059b1345b7e8be0df20f1b23caa3602e85d42fff7ecd9d0bd255de56   1377           targets

	targets := make(map[string]string)

	// no target
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && strings.Contains(out, "No targets present in this repository.") {
		return targets
	}

	// otherwise, there is at least one target
	assert.Assert(t, len(lines) >= 3, "output is %s", out)

	for _, line := range lines[2:] {
		tokens := strings.Fields(line)
		assert.Assert(t, len(tokens) == 4)
		targets[tokens[0]] = tokens[3]
	}

	return targets
}

func setupNotaryConfig(t *testing.T, dockerConfigDir fs.Dir) *fs.Dir {
	return fs.NewDir(t, "notary_test", fs.WithMode(0700),
		fs.WithFile("client-config.json", fmt.Sprintf(`
{
	"trust_dir": "%s",
	"remote_server": {
		"url": "%s"
	}
}`, dockerConfigDir.Join("trust"), fixtures.NotaryURL)),
	)
}

func copyPrivateKey(t *testing.T, dir, source string) {
	icmd.RunCommand("/bin/cp", source, dir).Assert(t, icmd.Success)
}
