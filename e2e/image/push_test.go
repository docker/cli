package image

import (
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/environment"
	"github.com/docker/cli/internal/test/output"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/skip"
)

func TestPushAllTags(t *testing.T) {
	skip.If(t, environment.RemoteDaemon())

	// Compared digests are linux/amd64 specific.
	// TODO: Fix this test and make it work on all platforms.
	environment.SkipIfNotPlatform(t, "linux/amd64")

	_ = createImage(t, "push-all-tags", "latest", "v1", "v1.0", "v1.0.1")
	result := icmd.RunCmd(icmd.Command("docker", "push", "--all-tags", registryPrefix+"/push-all-tags"))

	result.Assert(t, icmd.Success)
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0:  output.Equals("The push refers to repository [registry:5000/push-all-tags]"),
		1:  output.Equals("7cd52847ad77: Preparing"),
		3:  output.Equals("latest: digest: sha256:e2e16842c9b54d985bf1ef9242a313f36b856181f188de21313820e177002501 size: 528"),
		6:  output.Equals("v1: digest: sha256:e2e16842c9b54d985bf1ef9242a313f36b856181f188de21313820e177002501 size: 528"),
		9:  output.Equals("v1.0: digest: sha256:e2e16842c9b54d985bf1ef9242a313f36b856181f188de21313820e177002501 size: 528"),
		12: output.Equals("v1.0.1: digest: sha256:e2e16842c9b54d985bf1ef9242a313f36b856181f188de21313820e177002501 size: 528"),
	})
}

func TestPushQuietErrors(t *testing.T) {
	result := icmd.RunCmd(icmd.Command("docker", "push", "--quiet", "nosuchimage"))
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "An image does not exist locally with the tag: nosuchimage",
	})
}

func createImage(t *testing.T, repo string, tags ...string) string {
	t.Helper()
	icmd.RunCommand("docker", "pull", fixtures.AlpineImage).Assert(t, icmd.Success)

	for _, tag := range tags {
		image := fmt.Sprintf("%s/%s:%s", registryPrefix, repo, tag)
		icmd.RunCommand("docker", "tag", fixtures.AlpineImage, image).Assert(t, icmd.Success)
	}
	return fmt.Sprintf("%s/%s:%s", registryPrefix, repo, tags[0])
}
