package container

import (
	"bytes"
	"io"
	"math/rand"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/environment"
	"github.com/moby/moby/client/pkg/versions"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/poll"
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

func TestRunAttach(t *testing.T) {
	skip.If(t, environment.RemoteDaemon())
	t.Parallel()

	streams := []string{"stdin", "stdout", "stderr"}
	for _, stream := range streams {
		t.Run(stream, func(t *testing.T) {
			t.Parallel()
			c := exec.Command("docker", "run", "-a", stream, "--rm", "alpine",
				"sh", "-c", "sleep 1 && exit 7")
			d := bytes.Buffer{}
			c.Stdout = &d
			c.Stderr = &d
			_, err := pty.Start(c)
			assert.NilError(t, err)

			done := make(chan error)
			go func() {
				done <- c.Wait()
			}()

			select {
			case <-time.After(20 * time.Second):
				t.Fatal("docker run took too long, likely hang", d.String())
			case <-done:
			}

			assert.Equal(t, c.ProcessState.ExitCode(), 7)
		})
	}
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
		t.Run(tc.name, func(t *testing.T) {
			runMount(tc.cmd, "volume-subpath="+tc.subpath).Assert(t, icmd.Expected{
				Err:      tc.expectedErr,
				ExitCode: tc.expectedCode,
				Out:      tc.expectedOut,
			})
		})
	}
}

func TestProcessTermination(t *testing.T) {
	var out bytes.Buffer
	cmd := icmd.Command("docker", "run", "--rm", "-i", fixtures.AlpineImage,
		"sh", "-c", "echo 'starting trap'; trap 'echo got signal; exit 0;' TERM; while true; do sleep 10; done")
	cmd.Stdout = &out
	cmd.Stderr = &out

	result := icmd.StartCmd(cmd).Assert(t, icmd.Success)

	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		if strings.Contains(result.Stdout(), "starting trap") {
			return poll.Success()
		}
		return poll.Continue("waiting for process to trap signal")
	}, poll.WithDelay(1*time.Second), poll.WithTimeout(5*time.Second))

	assert.NilError(t, result.Cmd.Process.Signal(syscall.SIGTERM))

	icmd.WaitOnCmd(time.Second*10, result).Assert(t, icmd.Expected{
		ExitCode: 0,
	})
}

// Adapted from https://github.com/docker/for-mac/issues/7632#issue-2932169772
// Thanks [@almet](https://github.com/almet)!
func TestRunReadAfterContainerExit(t *testing.T) {
	skip.If(t, environment.RemoteDaemon())

	r := rand.New(rand.NewSource(0x123456))

	const size = 18933764
	cmd := exec.Command("docker", "run",
		"--rm", "-i",
		"alpine",
		"sh", "-c", "cat -",
	)

	cmd.Stdin = io.LimitReader(r, size)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	assert.NilError(t, err)

	err = cmd.Start()
	assert.NilError(t, err)

	buffer := make([]byte, 1000)
	counter := 0
	totalRead := 0

	for {
		n, err := stdout.Read(buffer)
		if n > 0 {
			totalRead += n
		}

		// Wait 0.1s every megabyte (approx.)
		if counter%1000 == 0 {
			time.Sleep(100 * time.Millisecond)
		}

		if err != nil || n == 0 {
			break
		}

		counter++
	}

	err = cmd.Wait()
	t.Logf("Error: %v", err)
	t.Logf("Stderr: %s", stderr.String())
	assert.Check(t, err == nil)
	assert.Check(t, is.Equal(totalRead, size))
}
