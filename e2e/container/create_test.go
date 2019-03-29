package container

import (
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/environment"
	"gotest.tools/icmd"
	"gotest.tools/skip"
)

func TestCreateWithContentTrust(t *testing.T) {
	skip.If(t, environment.RemoteDaemon())

	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := fixtures.CreateMaskedTrustedRemoteImage(t, registryPrefix, "trust-create", "latest")

	defer func() {
		icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
	}()

	result := icmd.RunCmd(
		icmd.Command("docker", "create", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		Err: fmt.Sprintf("Tagging %s@sha", image[:len(image)-7]),
	})
}

func TestTrustedCreateFromUnreachableTrustServer(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := fixtures.CreateMaskedTrustedRemoteImage(t, registryPrefix, "trust-create", "latest")

	result := icmd.RunCmd(
		icmd.Command("docker", "create", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotaryServer("https://notary.invalid"),
	)
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "error contacting notary server",
	})
}

func TestTrustedCreateFromBadTrustServer(t *testing.T) {
	evilImageName := "registry:5000/evil-alpine:latest"
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

	// try create
	icmd.RunCmd(icmd.Command("docker", "create", evilImageName),
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

	// try creating with the original client from the evil notary server. This should failed
	// because the new root is invalid
	icmd.RunCmd(icmd.Command("docker", "create", evilImageName),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotaryServer(fixtures.EvilNotaryURL),
	).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "could not rotate trust to a new trusted root",
	})
}
