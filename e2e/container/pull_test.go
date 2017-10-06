package container

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/icmd"
)

const notaryURL = "https://notary-server:4443"
const registryPrefix = "registry:5000"

func TestPullWithContentTrust(t *testing.T) {
	image := createTrustedRemoteImage(t, "trust", "latest")
	icmd.RunCmd(trustedCmdNoPassphrases(icmd.Command("docker", "pull", image))).Assert(t, icmd.Success)

	// test that pulling without the tag defaults to latest
	imageWithoutTag := strings.TrimSuffix(image, ":latest")
	icmd.RunCmd(trustedCmdNoPassphrases(icmd.Command("docker", "pull", imageWithoutTag))).Assert(t, icmd.Success)
}

func createTrustedRemoteImage(t *testing.T, repo, tag string) string {
	image := fmt.Sprintf("%s/%s:%s", registryPrefix, repo, tag)
	icmd.RunCommand("docker", "pull", alpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", alpineImage, image).Assert(t, icmd.Success)
	icmd.RunCmd(trustedCmdWithPassphrases(icmd.Command("docker", "push", image), "root_password", "repo_password")).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
	return image
}

func trustedCmdWithPassphrases(cmd icmd.Cmd, rootPwd, repositoryPwd string) icmd.Cmd {
	env := append(os.Environ(), []string{
		"DOCKER_CONTENT_TRUST=1",
		fmt.Sprintf("DOCKER_CONTENT_TRUST_SERVER=%s", notaryURL),
		fmt.Sprintf("DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE=%s", rootPwd),
		fmt.Sprintf("DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE=%s", repositoryPwd),
	}...)
	cmd.Env = append(cmd.Env, env...)
	return cmd
}

func trustedCmdNoPassphrases(cmd icmd.Cmd) icmd.Cmd {
	env := append(os.Environ(), []string{
		"DOCKER_CONTENT_TRUST=1",
		fmt.Sprintf("DOCKER_CONTENT_TRUST_SERVER=%s", notaryURL),
	}...)
	cmd.Env = append(cmd.Env, env...)
	return cmd
}
