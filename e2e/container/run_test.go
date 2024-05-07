package container

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/environment"
	"github.com/docker/docker/api/types/versions"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/skip"
)

const registryPrefix = "registry:5000"

func TestRunAttachedFromRemoteImageAndRemove(t *testing.T) {
	skip.If(t, environment.RemoteDaemon())

	// Digests in golden file are linux/amd64 specific.
	// TODO: Fix this test and make it work on all platforms.
	environment.SkipIfNotPlatform(t, "linux/amd64")

	image := createRemoteImage(t)

	result := icmd.RunCommand("docker", "run", "--rm", image,
		"echo", "this", "is", "output")

	result.Assert(t, icmd.Success)
	assert.Check(t, is.Equal("this is output\n", result.Stdout()))
	golden.Assert(t, result.Stderr(), "run-attached-from-remote-and-remove.golden")
}

// Regression test for https://github.com/docker/cli/issues/5053
func TestRunInvalidEntrypointWithAutoremove(t *testing.T) {
	environment.SkipIfDaemonNotLinux(t)

	result := make(chan *icmd.Result)
	go func() {
		result <- icmd.RunCommand("docker", "run", "--rm", fixtures.AlpineImage, "invalidcommand")
	}()
	select {
	case r := <-result:
		r.Assert(t, icmd.Expected{ExitCode: 127})
	case <-time.After(4 * time.Second):
		t.Fatal("test took too long, shouldn't hang")
	}
}

func TestRunWithContentTrust(t *testing.T) {
	skip.If(t, environment.RemoteDaemon())

	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := fixtures.CreateMaskedTrustedRemoteImage(t, registryPrefix, "trust-run", "latest")

	defer func() {
		icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
	}()

	result := icmd.RunCmd(
		icmd.Command("docker", "run", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		Err: fmt.Sprintf("Tagging %s@sha", image[:len(image)-7]),
	})
}

func TestUntrustedRun(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := registryPrefix + "/alpine:untrusted"
	// tag the image and upload it to the private registry
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, image).Assert(t, icmd.Success)
	defer func() {
		icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
	}()

	// try trusted run on untrusted tag
	result := icmd.RunCmd(
		icmd.Command("docker", "run", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		ExitCode: 125,
		Err:      "does not have trust data for",
	})
}

func TestTrustedRunFromBadTrustServer(t *testing.T) {
	evilImageName := registryPrefix + "/evil-alpine:latest"
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()

	// tag the image and upload it to the private registry
	icmd.RunCmd(icmd.Command("docker", "tag", fixtures.AlpineImage, evilImageName),
		fixtures.WithConfig(dir.Path()),
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "push", evilImageName),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithPassphrase("root_password", "repo_password"),
		fixtures.WithTrust,
		fixtures.WithNotary,
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "rm", evilImageName)).Assert(t, icmd.Success)

	// try run
	icmd.RunCmd(icmd.Command("docker", "run", evilImageName),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "rm", evilImageName)).Assert(t, icmd.Success)

	// init a client with the evil-server and a new trust dir
	evilNotaryDir := fixtures.SetupConfigWithNotaryURL(t, "evil-test", fixtures.EvilNotaryURL)
	defer evilNotaryDir.Remove()

	// tag the same image and upload it to the private registry but signed with evil notary server
	icmd.RunCmd(icmd.Command("docker", "tag", fixtures.AlpineImage, evilImageName),
		fixtures.WithConfig(evilNotaryDir.Path()),
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "push", evilImageName),
		fixtures.WithConfig(evilNotaryDir.Path()),
		fixtures.WithPassphrase("root_password", "repo_password"),
		fixtures.WithTrust,
		fixtures.WithNotaryServer(fixtures.EvilNotaryURL),
	).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Command("docker", "image", "rm", evilImageName)).Assert(t, icmd.Success)

	// try running with the original client from the evil notary server. This should failed
	// because the new root is invalid
	icmd.RunCmd(icmd.Command("docker", "run", evilImageName),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotaryServer(fixtures.EvilNotaryURL),
	).Assert(t, icmd.Expected{
		ExitCode: 125,
		Err:      "could not rotate trust to a new trusted root",
	})
}

// TODO: create this with registry API instead of engine API
func createRemoteImage(t *testing.T) string {
	t.Helper()
	image := registryPrefix + "/alpine:test-run-pulls"
	icmd.RunCommand("docker", "pull", fixtures.AlpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "push", image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
	return image
}

func TestRunWithCgroupNamespace(t *testing.T) {
	environment.SkipIfDaemonNotLinux(t)
	environment.SkipIfCgroupNamespacesNotSupported(t)

	result := icmd.RunCommand("docker", "run", "--cgroupns=private", "--rm", fixtures.AlpineImage,
		"cat", "/sys/fs/cgroup/cgroup.controllers")
	result.Assert(t, icmd.Success)
}

func TestMountSubvolume(t *testing.T) {
	skip.If(t, versions.LessThan(environment.DaemonAPIVersion(t), "1.45"))

	volName := "test-volume-" + t.Name()
	icmd.RunCommand("docker", "volume", "create", volName).Assert(t, icmd.Success)

	t.Cleanup(func() {
		icmd.RunCommand("docker", "volume", "remove", "-f", volName).Assert(t, icmd.Success)
	})

	defaultMountOpts := []string{
		"type=volume",
		"src=" + volName,
		"dst=/volume",
	}

	// Populate the volume with test data.
	icmd.RunCommand("docker", "run", "--rm", "--mount", strings.Join(defaultMountOpts, ","), fixtures.AlpineImage, "sh", "-c",
		"echo foo > /volume/bar.txt && "+
			"mkdir /volume/etc && echo root > /volume/etc/passwd && "+
			"mkdir /volume/subdir && echo world > /volume/subdir/hello.txt;",
	).Assert(t, icmd.Success)

	runMount := func(cmd string, mountOpts ...string) *icmd.Result {
		mountArg := strings.Join(append(defaultMountOpts, mountOpts...), ",")
		return icmd.RunCommand("docker", "run", "--rm", "--mount", mountArg, fixtures.AlpineImage, cmd, "/volume")
	}

	for _, tc := range []struct {
		name    string
		cmd     string
		subpath string

		expectedOut  string
		expectedErr  string
		expectedCode int
	}{
		{name: "absolute", cmd: "cat", subpath: "/etc/passwd", expectedErr: "subpath must be a relative path within the volume", expectedCode: 125},
		{name: "subpath not exists", cmd: "ls", subpath: "some-path/that/doesnt-exist", expectedErr: "cannot access path ", expectedCode: 127},
		{name: "subdirectory mount", cmd: "ls", subpath: "subdir", expectedOut: "hello.txt"},
		{name: "file mount", cmd: "cat", subpath: "bar.txt", expectedOut: "foo"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			runMount(tc.cmd, "volume-subpath="+tc.subpath).Assert(t, icmd.Expected{
				Err:      tc.expectedErr,
				ExitCode: tc.expectedCode,
				Out:      tc.expectedOut,
			})
		})
	}
}
