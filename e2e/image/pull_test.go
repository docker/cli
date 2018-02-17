package image

import (
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
)

const registryPrefix = "registry:5000"

func TestPullWithContentTrust(t *testing.T) {
	image := createMaskedTrustedRemoteImage(t, "trust", "latest")

	result := icmd.RunCmd(icmd.Command("docker", "pull", image), fixtures.WithTrust, fixtures.WithNotary)
	result.Assert(t, icmd.Success)
	golden.Assert(t, result.Stderr(), "pull-with-content-trust-err.golden")
	golden.Assert(t, result.Stdout(), "pull-with-content-trust.golden")
}

// createMaskedTrustedRemoteImage creates a remote image that is signed with
// content trust, then pushes a different untrusted image at the same tag.
func createMaskedTrustedRemoteImage(t *testing.T, repo, tag string) string {
	image := createTrustedRemoteImage(t, repo, tag)
	createNamedUnsignedImageFromBusyBox(t, image)
	return image
}

func createTrustedRemoteImage(t *testing.T, repo, tag string) string {
	image := fmt.Sprintf("%s/%s:%s", registryPrefix, repo, tag)
	icmd.RunCommand("docker", "pull", fixtures.AlpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, image).Assert(t, icmd.Success)
	result := icmd.RunCmd(
		icmd.Command("docker", "push", image),
		fixtures.WithPassphrase("root_password", "repo_password"), fixtures.WithTrust, fixtures.WithNotary)
	result.Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
	return image
}

func createNamedUnsignedImageFromBusyBox(t *testing.T, image string) {
	icmd.RunCommand("docker", "pull", fixtures.BusyboxImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.BusyboxImage, image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "push", image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
}
