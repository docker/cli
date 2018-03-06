package trust

import (
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/icmd"
)

const (
	localImage = "registry:5000/signlocal:v1"
	signImage  = "registry:5000/sign:v1"
)

func TestSignLocalImage(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	icmd.RunCmd(icmd.Command("docker", "pull", fixtures.AlpineImage)).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, signImage).Assert(t, icmd.Success)
	result := icmd.RunCmd(
		icmd.Command("docker", "trust", "sign", signImage),
		fixtures.WithPassphrase("root_password", "repo_password"),
		fixtures.WithConfig(dir.Path()), fixtures.WithNotary)
	result.Assert(t, icmd.Success)
	assert.Check(t, is.Contains(result.Stdout(), fmt.Sprintf("v1: digest: sha256:%s", fixtures.AlpineSha)))

}

func TestSignWithLocalFlag(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	setupTrustedImageForOverwrite(t, dir)
	result := icmd.RunCmd(
		icmd.Command("docker", "trust", "sign", "--local", localImage),
		fixtures.WithPassphrase("root_password", "repo_password"),
		fixtures.WithConfig(dir.Path()), fixtures.WithNotary)
	result.Assert(t, icmd.Success)
	assert.Check(t, is.Contains(result.Stdout(), fmt.Sprintf("v1: digest: sha256:%s", fixtures.BusyboxSha)))
}

func setupTrustedImageForOverwrite(t *testing.T, dir fs.Dir) {
	icmd.RunCmd(icmd.Command("docker", "pull", fixtures.AlpineImage)).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, localImage).Assert(t, icmd.Success)
	result := icmd.RunCmd(
		icmd.Command("docker", "-D", "trust", "sign", localImage),
		fixtures.WithPassphrase("root_password", "repo_password"),
		fixtures.WithConfig(dir.Path()), fixtures.WithNotary)
	result.Assert(t, icmd.Success)
	assert.Check(t, is.Contains(result.Stdout(), fmt.Sprintf("v1: digest: sha256:%s", fixtures.AlpineSha)))
	icmd.RunCmd(icmd.Command("docker", "pull", fixtures.BusyboxImage)).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.BusyboxImage, localImage).Assert(t, icmd.Success)
}
