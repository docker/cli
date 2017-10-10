package image

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/stretchr/testify/require"
)

const notaryURL = "https://notary-server:4443"
const registryPrefix = "registry:5000"

const alpineImage = "registry:5000/alpine:3.6"
const busyboxImage = "registry:5000/busybox:1.27.2"

func TestPullWithContentTrust(t *testing.T) {
	image := createTrustedRemoteImage(t, "trust", "latest")

	// test that pulling without the tag defaults to latest
	imageWithoutTag := strings.TrimSuffix(image, ":latest")
	icmd.RunCmd(trustedCmdNoPassphrases(icmd.Command("docker", "pull", imageWithoutTag))).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)

	// try pulling with the tag, record output for comparison later
	result := icmd.RunCmd(trustedCmdNoPassphrases(icmd.Command("docker", "pull", image)))
	result.Assert(t, icmd.Success)
	firstPullOutput := result.String()
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)

	// push an unsigned image on the same reference name, but with different content (busybox)
	createNamedUnsignedImageFromBusyBox(t, image)

	// now pull with content trust
	result = icmd.RunCmd(trustedCmdNoPassphrases(icmd.Command("docker", "pull", image)))
	result.Assert(t, icmd.Success)
	secondPullOutput := result.String()

	// assert that the digest and other output is the same since we ignore the unsigned image
	require.Equal(t, firstPullOutput, secondPullOutput)
}

func createTrustedRemoteImage(t *testing.T, repo, tag string) string {
	image := fmt.Sprintf("%s/%s:%s", registryPrefix, repo, tag)
	icmd.RunCommand("docker", "pull", alpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", alpineImage, image).Assert(t, icmd.Success)
	icmd.RunCmd(trustedCmdWithPassphrases(icmd.Command("docker", "push", image), "root_password", "repo_password")).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
	return image
}

func createNamedUnsignedImageFromBusyBox(t *testing.T, image string) {
	icmd.RunCommand("docker", "pull", busyboxImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", busyboxImage, image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "push", image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
}

func trustedCmdWithPassphrases(cmd icmd.Cmd, rootPwd, repositoryPwd string) icmd.Cmd {
	env := append(os.Environ(), []string{
		"DOCKER_CONTENT_TRUST=1",
		"DOCKER_CONTENT_TRUST_SERVER=" + notaryURL,
		"DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE=" + rootPwd,
		"DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE=" + repositoryPwd,
	}...)
	cmd.Env = append(cmd.Env, env...)
	return cmd
}

func trustedCmdNoPassphrases(cmd icmd.Cmd) icmd.Cmd {
	env := append(os.Environ(), []string{
		"DOCKER_CONTENT_TRUST=1",
		"DOCKER_CONTENT_TRUST_SERVER=" + notaryURL,
	}...)
	cmd.Env = append(cmd.Env, env...)
	return cmd
}
