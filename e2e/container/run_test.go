package container

import (
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/environment"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"
	"gotest.tools/icmd"
	"gotest.tools/skip"
)

const registryPrefix = "registry:5000"

func TestRunAttachedFromRemoteImageAndRemove(t *testing.T) {
	skip.If(t, environment.RemoteDaemon())

	image := createRemoteImage(t)

	result := icmd.RunCommand("docker", "run", "--rm", image,
		"echo", "this", "is", "output")

	result.Assert(t, icmd.Success)
	assert.Check(t, is.Equal("this is output\n", result.Stdout()))
	golden.Assert(t, result.Stderr(), "run-attached-from-remote-and-remove.golden")
}

func TestRunWithContentTrust(t *testing.T) {
	skip.If(t, environment.RemoteDaemon())

	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := fixtures.CreateMaskedTrustedRemoteImage(t, registryPrefix, "trust-run", "latest")

	defer func() {
		icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
	}()

	result := icmd.RunCmd(
		icmd.Command("docker", "run", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		Err: fmt.Sprintf("Tagging %s@sha", image[:len(image)-7]),
	})
}

func TestUntrustedRun(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := registryPrefix + "/alpine:untrusted"
	// tag the image and upload it to the private registry
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, image).Assert(t, icmd.Success)
	defer func() {
		icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
	}()

	// try trusted run on untrusted tag
	result := icmd.RunCmd(
		icmd.Command("docker", "run", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		ExitCode: 125,
		Err:      "does not have trust data for",
	})
}

func TestTrustedRunFromBadTrustServer(t *testing.T) {
	evilImageName := registryPrefix + "/evil-alpine:latest"
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()

	// tag the image and upload it to the private registry
	icmd.RunCmd(icmd.Command("docker", "tag", fixtures.AlpineImage, evilImageName),
		fixtures.WithConfig(dir.Path()),
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "push", evilImageName),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithPassphrase("root_password", "repo_password"),
		fixtures.WithTrust,
		fixtures.WithNotary,
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "rm", evilImageName)).Assert(t, icmd.Success)

	// try run
	icmd.RunCmd(icmd.Command("docker", "run", evilImageName),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "rm", evilImageName)).Assert(t, icmd.Success)

	// init a client with the evil-server and a new trust dir
	evilNotaryDir := fixtures.SetupConfigWithNotaryURL(t, "evil-test", fixtures.EvilNotaryURL)
	defer evilNotaryDir.Remove()

	// tag the same image and upload it to the private registry but signed with evil notary server
	icmd.RunCmd(icmd.Command("docker", "tag", fixtures.AlpineImage, evilImageName),
		fixtures.WithConfig(evilNotaryDir.Path()),
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "push", evilImageName),
		fixtures.WithConfig(evilNotaryDir.Path()),
		fixtures.WithPassphrase("root_password", "repo_password"),
		fixtures.WithTrust,
		fixtures.WithNotaryServer(fixtures.EvilNotaryURL),
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "rm", evilImageName)).Assert(t, icmd.Success)

	// try running with the original client from the evil notary server. This should failed
	// because the new root is invalid
	icmd.RunCmd(icmd.Command("docker", "run", evilImageName),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotaryServer(fixtures.EvilNotaryURL),
	).Assert(t, icmd.Expected{
		ExitCode: 125,
		Err:      "could not rotate trust to a new trusted root",
	})
}

// TODO: create this with registry API instead of engine API
func createRemoteImage(t *testing.T) string {
	image := registryPrefix + "/alpine:test-run-pulls"
	icmd.RunCommand("docker", "pull", fixtures.AlpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "push", image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
	return image
}
