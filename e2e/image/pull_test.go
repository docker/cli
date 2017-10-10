package image

import (
	"fmt"
	"os"
	"testing"

	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
)

const notaryURL = "https://notary-server:4443"
const registryPrefix = "registry:5000"

const alpineImage = "registry:5000/alpine:3.6"
const busyboxImage = "registry:5000/busybox:1.27.2"

func TestPullWithContentTrust(t *testing.T) {
	image := createMaskedTrustedRemoteImage(t, "trust", "latest")

	result := icmd.RunCmd(icmd.Command("docker", "pull", image), withTrustNoPassphrase)
	result.Assert(t, icmd.Expected{Err: icmd.None})
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
	icmd.RunCommand("docker", "pull", alpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", alpineImage, image).Assert(t, icmd.Success)
	result := icmd.RunCmd(
		icmd.Command("docker", "push", image),
		withTrustAndPassphrase("root_password", "repo_password"))
	result.Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
	return image
}

func createNamedUnsignedImageFromBusyBox(t *testing.T, image string) {
	icmd.RunCommand("docker", "pull", busyboxImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", busyboxImage, image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "push", image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
}

func withTrustAndPassphrase(rootPwd, repositoryPwd string) func(cmd *icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		env := append(os.Environ(),
			"DOCKER_CONTENT_TRUST=1",
			"DOCKER_CONTENT_TRUST_SERVER="+notaryURL,
			"DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE="+rootPwd,
			"DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE="+repositoryPwd,
		)
		cmd.Env = append(cmd.Env, env...)
	}
}

func withTrustNoPassphrase(cmd *icmd.Cmd) {
	env := append(os.Environ(),
		"DOCKER_CONTENT_TRUST=1",
		"DOCKER_CONTENT_TRUST_SERVER="+notaryURL,
	)
	cmd.Env = append(cmd.Env, env...)
}
