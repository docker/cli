package image

import (
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
)

const registryPrefix = "registry:5000"

func TestPullWithContentTrust(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := fixtures.CreateMaskedTrustedRemoteImage(t, registryPrefix, "trust-pull", "latest")
	defer func() {
		icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
	}()

	result := icmd.RunCmd(icmd.Command("docker", "pull", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Success)
	golden.Assert(t, result.Stderr(), "pull-with-content-trust-err.golden")
	golden.Assert(t, result.Stdout(), "pull-with-content-trust.golden")
}

func TestPullWithContentTrustUsesCacheWhenNotaryUnavailable(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := fixtures.CreateMaskedTrustedRemoteImage(t, registryPrefix, "trust-pull-unreachable", "latest")
	defer func() {
		icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
	}()
	result := icmd.RunCmd(icmd.Command("docker", "pull", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotaryServer("https://invalidnotaryserver"),
	)
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "error contacting notary server",
	})

	// Do valid trusted pull to warm cache
	result = icmd.RunCmd(icmd.Command("docker", "pull", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Success)
	result = icmd.RunCommand("docker", "rmi", image)
	result.Assert(t, icmd.Success)

	// Try pull again with invalid notary server, should use cache
	result = icmd.RunCmd(icmd.Command("docker", "pull", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotaryServer("https://invalidnotaryserver"),
	)
	result.Assert(t, icmd.Success)
}
