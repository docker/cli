package fixtures

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/cli/cli/config"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

const (
	// NotaryURL is the location of the notary server
	NotaryURL = "https://notary-server:4443"
	// EvilNotaryURL is the location of the evil notary server
	EvilNotaryURL = "https://evil-notary-server:4444"
	// AlpineImage is an image in the test registry
	AlpineImage = "registry:5000/alpine:frozen"
	// AlpineSha is the sha of the alpine image
	AlpineSha = "e2e16842c9b54d985bf1ef9242a313f36b856181f188de21313820e177002501" // #nosec G101 -- ignoring: Potential hardcoded credentials (gosec)
	// BusyboxImage is an image in the test registry
	BusyboxImage = "registry:5000/busybox:frozen"
	// BusyboxSha is the sha of the busybox image
	BusyboxSha = "030fcb92e1487b18c974784dcc110a93147c9fc402188370fbfd17efabffc6af" // #nosec G101 -- ignoring: Potential hardcoded credentials (gosec)
)

// SetupConfigFile creates a config.json file for testing
func SetupConfigFile(t *testing.T) fs.Dir {
	t.Helper()
	return SetupConfigWithNotaryURL(t, "trust_test", NotaryURL)
}

// SetupConfigWithNotaryURL creates a config.json file for testing in the given path
// with the given notaryURL
func SetupConfigWithNotaryURL(t *testing.T, path, notaryURL string) fs.Dir {
	t.Helper()
	dir := fs.NewDir(t, path, fs.WithMode(0o700), fs.WithFile("config.json", fmt.Sprintf(`
	{
		"auths": {
			"registry:5000": {
				"auth": "ZWlhaXM6cGFzc3dvcmQK"
			},
			"%s": {
				"auth": "ZWlhaXM6cGFzc3dvcmQK"
			}
		},
		"experimental": "enabled"
	}
	`, notaryURL)), fs.WithDir("trust", fs.WithDir("private")))
	return *dir
}

// WithConfig sets an environment variable for the docker config location
func WithConfig(dir string) func(cmd *icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		addEnvs(cmd, config.EnvOverrideConfigDir+"="+dir)
	}
}

// WithPassphrase sets environment variables for passphrases
func WithPassphrase(rootPwd, repositoryPwd string) func(cmd *icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		addEnvs(cmd,
			"DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE="+rootPwd,
			"DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE="+repositoryPwd,
		)
	}
}

// WithTrust sets DOCKER_CONTENT_TRUST to 1
func WithTrust(cmd *icmd.Cmd) {
	addEnvs(cmd, "DOCKER_CONTENT_TRUST=1")
}

// WithNotary sets the location of the notary server
func WithNotary(cmd *icmd.Cmd) {
	addEnvs(cmd, "DOCKER_CONTENT_TRUST_SERVER="+NotaryURL)
}

// WithHome sets the HOME environment variable
func WithHome(path string) func(*icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		addEnvs(cmd, "HOME="+path)
	}
}

// WithNotaryServer sets the location of the notary server
func WithNotaryServer(notaryURL string) func(*icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		addEnvs(cmd, "DOCKER_CONTENT_TRUST_SERVER="+notaryURL)
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

// addEnvs adds environment variables to cmd, making sure to preserve the
// current os.Environ(), which would otherwise be omitted (for non-empty .Env).
func addEnvs(cmd *icmd.Cmd, envs ...string) {
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, envs...)
}
