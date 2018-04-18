package fixtures

import (
	"fmt"
	"os"
	"testing"

	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/icmd"
)

const (
	//NotaryURL is the location of the notary server
	NotaryURL = "https://notary-server:4443"
	//AlpineImage is an image in the test registry
	AlpineImage = "registry:5000/alpine:3.6"
	//AlpineSha is the sha of the alpine image
	AlpineSha = "641b95ddb2ea9dc2af1a0113b6b348ebc20872ba615204fbe12148e98fd6f23d"
	//BusyboxImage is an image in the test registry
	BusyboxImage = "registry:5000/busybox:1.27.2"
	//BusyboxSha is the sha of the busybox image
	BusyboxSha = "030fcb92e1487b18c974784dcc110a93147c9fc402188370fbfd17efabffc6af"
)

//SetupConfigFile creates a config.json file for testing
func SetupConfigFile(t *testing.T) fs.Dir {
	t.Helper()
	dir := fs.NewDir(t, "trust_test", fs.WithMode(0700), fs.WithFile("config.json", `
	{
		"auths": {
			"registry:5000": {
				"auth": "ZWlhaXM6cGFzc3dvcmQK"
			},
			"https://notary-server:4443": {
				"auth": "ZWlhaXM6cGFzc3dvcmQK"
			}
		},
		"experimental": "enabled"
	}
	`), fs.WithDir("trust", fs.WithDir("private")))
	return *dir
}

//WithConfig sets an environment variable for the docker config location
func WithConfig(dir string) func(cmd *icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		env := append(os.Environ(),
			"DOCKER_CONFIG="+dir,
		)
		cmd.Env = append(cmd.Env, env...)
	}
}

//WithPassphrase sets environment variables for passphrases
func WithPassphrase(rootPwd, repositoryPwd string) func(cmd *icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		env := append(os.Environ(),
			"DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE="+rootPwd,
			"DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE="+repositoryPwd,
		)
		cmd.Env = append(cmd.Env, env...)
	}
}

//WithTrust sets DOCKER_CONTENT_TRUST to 1
func WithTrust(cmd *icmd.Cmd) {
	env := append(os.Environ(),
		"DOCKER_CONTENT_TRUST=1",
	)
	cmd.Env = append(cmd.Env, env...)
}

//WithNotary sets the location of the notary server
func WithNotary(cmd *icmd.Cmd) {
	env := append(os.Environ(),
		"DOCKER_CONTENT_TRUST_SERVER="+NotaryURL,
	)
	cmd.Env = append(cmd.Env, env...)
}

//WithHome sets the HOME environment variable
func WithHome(path string) func(*icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		cmd.Env = append(cmd.Env, "HOME="+path)
	}
}

//WithNotaryServer sets the location of the notary server
func WithNotaryServer(notaryURL string) func(*icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		env := append(os.Environ(),
			"DOCKER_CONTENT_TRUST_SERVER="+notaryURL,
		)
		cmd.Env = append(cmd.Env, env...)
	}
}

// CreateMaskedTrustedRemoteImage creates a remote image that is signed with
// content trust, then pushes a different untrusted image at the same tag.
func CreateMaskedTrustedRemoteImage(t *testing.T, registryPrefix, repo, tag string) string {
	t.Helper()
	image := createTrustedRemoteImage(t, registryPrefix, repo, tag)
	createNamedUnsignedImageFromBusyBox(t, image)
	return image
}

func createTrustedRemoteImage(t *testing.T, registryPrefix, repo, tag string) string {
	t.Helper()
	image := fmt.Sprintf("%s/%s:%s", registryPrefix, repo, tag)
	icmd.RunCommand("docker", "image", "pull", AlpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "image", "tag", AlpineImage, image).Assert(t, icmd.Success)
	result := icmd.RunCmd(
		icmd.Command("docker", "image", "push", image),
		WithPassphrase("root_password", "repo_password"), WithTrust, WithNotary)
	result.Assert(t, icmd.Success)
	icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
	return image
}

func createNamedUnsignedImageFromBusyBox(t *testing.T, image string) {
	t.Helper()
	icmd.RunCommand("docker", "image", "pull", BusyboxImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "image", "tag", BusyboxImage, image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "image", "push", image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
}
