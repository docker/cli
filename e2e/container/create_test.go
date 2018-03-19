package container

import (
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/gotestyourself/gotestyourself/icmd"
)

func TestCreateWithContentTrust(t *testing.T) {
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

// FIXME(vdemeester) TestTrustedCreateFromBadTrustServer needs to be backport too (see https://github.com/moby/moby/pull/36515/files#diff-4b1e56bb77ac16f2ccf956fc24cf0a82L331)
