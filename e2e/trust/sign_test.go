package trust

import (
	"fmt"
	"os"
	"testing"

	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/stretchr/testify/assert"
)

const (
	notaryURL    = "https://notary-server:4443"
	alpineImage  = "registry:5000/alpine:3.6"
	alpineSha    = "641b95ddb2ea9dc2af1a0113b6b348ebc20872ba615204fbe12148e98fd6f23d"
	busyboxImage = "registry:5000/busybox:1.27.2"
	busyboxSha   = "030fcb92e1487b18c974784dcc110a93147c9fc402188370fbfd17efabffc6af"
	localImage   = "registry:5000/signlocal:v1"
	signImage    = "registry:5000/sign:v1"
)

func TestSignLocalImage(t *testing.T) {
	dir := setupConfigFile(t)
	defer dir.Remove()
	icmd.RunCmd(icmd.Command("docker", "pull", alpineImage)).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", alpineImage, signImage).Assert(t, icmd.Success)
	result := icmd.RunCmd(
		icmd.Command("docker", "trust", "sign", signImage),
		withTrustAndPassphrase("root_password", "repo_password", dir))
	result.Assert(t, icmd.Success)
	assert.Contains(t, result.Stdout(), fmt.Sprintf("v1: digest: sha256:%s", alpineSha))

}

func TestSignWithLocalFlag(t *testing.T) {
	dir := setupConfigFile(t)
	defer dir.Remove()
	setupTrustedImageForOverwrite(t, dir)
	result := icmd.RunCmd(
		icmd.Command("docker", "trust", "sign", "--local", localImage),
		withTrustAndPassphrase("root_password", "repo_password", dir))
	result.Assert(t, icmd.Success)
	assert.Contains(t, result.Stdout(), fmt.Sprintf("v1: digest: sha256:%s", busyboxSha))
}

func withTrustAndPassphrase(rootPwd, repositoryPwd string, dir fs.Dir) func(cmd *icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		env := append(os.Environ(),
			"DOCKER_CONTENT_TRUST_SERVER="+notaryURL,
			"DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE="+rootPwd,
			"DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE="+repositoryPwd,
			"DOCKER_CONFIG="+dir.Path(),
		)
		cmd.Env = append(cmd.Env, env...)
	}
}

func setupConfigFile(t *testing.T) fs.Dir {
	dir := fs.NewDir(t, "trust_test", fs.WithMode(0700), fs.WithFile("config.json", `
	{
		"auths": {
			"registry:5000": {
				"auth": "ZWlhaXM6cGFzc3dvcmQK"
			},
			"https://notary-server:4443": {
				"auth": "ZWlhaXM6cGFzc3dvcmQK"
			}
		}
	}
	`))
	return *dir
}

func setupTrustedImageForOverwrite(t *testing.T, dir fs.Dir) {
	icmd.RunCmd(icmd.Command("docker", "pull", alpineImage)).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", alpineImage, localImage).Assert(t, icmd.Success)
	result := icmd.RunCmd(
		icmd.Command("docker", "-D", "trust", "sign", localImage),
		withTrustAndPassphrase("root_password", "repo_password", dir))
	result.Assert(t, icmd.Success)
	assert.Contains(t, result.Stdout(), fmt.Sprintf("v1: digest: sha256:%s", alpineSha))
	icmd.RunCommand("docker", "tag", busyboxImage, localImage).Assert(t, icmd.Success)
}
