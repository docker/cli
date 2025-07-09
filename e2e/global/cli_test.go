package global

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/e2e/testutils"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/environment"
	"github.com/docker/docker/api/types/versions"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/poll"
	"gotest.tools/v3/skip"
)

func TestTLSVerify(t *testing.T) {
	// Remote daemons use TLS and this test is not applicable when TLS is required.
	skip.If(t, environment.RemoteDaemon())

	icmd.RunCmd(icmd.Command("docker", "ps")).Assert(t, icmd.Success)

	// Regardless of whether we specify true or false we need to
	// test to make sure tls is turned on if --tlsverify is specified at all
	result := icmd.RunCmd(icmd.Command("docker", "--tlsverify=false", "ps"))
	result.Assert(t, icmd.Expected{ExitCode: 1, Err: "unable to resolve docker endpoint:"})

	result = icmd.RunCmd(icmd.Command("docker", "--tlsverify=true", "ps"))
	result.Assert(t, icmd.Expected{ExitCode: 1, Err: "ca.pem"})
}

// TestTCPSchemeUsesHTTPProxyEnv verifies that the cli uses HTTP_PROXY if
// DOCKER_HOST is set to use the 'tcp://' scheme.
//
// Prior to go1.16, https:// schemes would use HTTPS_PROXY, and any other
// scheme would use HTTP_PROXY. However, golang/net@7b1cca2 (per a request in
// golang/go#40909) changed this behavior to only use HTTP_PROXY for http://
// schemes, no longer using a proxy for any other scheme.
//
// Docker uses the tcp:// scheme as a default for API connections, to indicate
// that the API is not "purely" HTTP. Various parts in the code also *require*
// this scheme to be used. While we could change the default and allow http(s)
// schemes to be used, doing so will take time, taking into account that there
// are many installs in existence that have tcp:// configured as DOCKER_HOST.
//
// Note that due to Golang's use of sync.Once for proxy-detection, this test
// cannot be done as a unit-test, hence it being an e2e test.
func TestTCPSchemeUsesHTTPProxyEnv(t *testing.T) {
	const responseJSON = `{"Version": "99.99.9", "ApiVersion": "1.41", "MinAPIVersion": "1.12"}`
	var received string
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Host
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer proxyServer.Close()

	// Configure the CLI to use our proxyServer. DOCKER_HOST can point to any
	// address (as it won't be connected to), but must use tcp:// for this test,
	// to verify it's using HTTP_PROXY.
	result := icmd.RunCmd(
		icmd.Command("docker", "version", "--format", "{{ .Server.Version }}"),
		icmd.WithEnv("HTTP_PROXY="+proxyServer.URL, "DOCKER_HOST=tcp://docker.acme.example.com:2376"),
	)
	// Verify the command ran successfully, and that it connected to the proxyServer
	result.Assert(t, icmd.Success)
	assert.Equal(t, strings.TrimSpace(result.Stdout()), "99.99.9")
	assert.Equal(t, received, "docker.acme.example.com:2376")
}

// Test that the prompt command exits with 0
// when the user sends SIGINT/SIGTERM to the process
func TestPromptExitCode(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dir := fixtures.SetupConfigFile(t)
	t.Cleanup(dir.Remove)

	defaultCmdOpts := []icmd.CmdOp{
		fixtures.WithConfig(dir.Path()),
		fixtures.WithNotary,
	}

	testCases := []struct {
		name string
		run  func(t *testing.T) icmd.Cmd
	}{
		{
			name: "volume prune",
			run: func(t *testing.T) icmd.Cmd {
				t.Helper()
				return icmd.Command("docker", "volume", "prune")
			},
		},
		{
			name: "network prune",
			run: func(t *testing.T) icmd.Cmd {
				t.Helper()
				return icmd.Command("docker", "network", "prune")
			},
		},
		{
			name: "container prune",
			run: func(t *testing.T) icmd.Cmd {
				t.Helper()
				return icmd.Command("docker", "container", "prune")
			},
		},
		{
			name: "image prune",
			run: func(t *testing.T) icmd.Cmd {
				t.Helper()
				return icmd.Command("docker", "image", "prune")
			},
		},
		{
			name: "system prune",
			run: func(t *testing.T) icmd.Cmd {
				t.Helper()
				return icmd.Command("docker", "system", "prune")
			},
		},
		{
			name: "revoke trust",
			run: func(t *testing.T) icmd.Cmd {
				t.Helper()
				return icmd.Command("docker", "trust", "revoke", "example/trust-demo")
			},
		},
		{
			name: "plugin install",
			run: func(t *testing.T) icmd.Cmd {
				t.Helper()
				skip.If(t, versions.LessThan(environment.DaemonAPIVersion(t), "1.44"))

				pluginDir := testutils.SetupPlugin(t, ctx)
				t.Cleanup(pluginDir.Remove)

				plugin := "registry:5000/plugin-content-trust-install:latest"

				icmd.RunCommand("docker", "plugin", "create", plugin, pluginDir.Path()).Assert(t, icmd.Success)
				icmd.RunCmd(icmd.Command("docker", "plugin", "push", plugin), defaultCmdOpts...).Assert(t, icmd.Success)
				icmd.RunCmd(icmd.Command("docker", "plugin", "rm", "-f", plugin), defaultCmdOpts...).Assert(t, icmd.Success)
				return icmd.Command("docker", "plugin", "install", plugin)
			},
		},
		{
			name: "plugin upgrade",
			run: func(t *testing.T) icmd.Cmd {
				t.Helper()
				skip.If(t, versions.LessThan(environment.DaemonAPIVersion(t), "1.44"))

				pluginLatestDir := testutils.SetupPlugin(t, ctx)
				t.Cleanup(pluginLatestDir.Remove)
				pluginNextDir := testutils.SetupPlugin(t, ctx)
				t.Cleanup(pluginNextDir.Remove)

				plugin := "registry:5000/plugin-content-trust-upgrade"

				icmd.RunCommand("docker", "plugin", "create", plugin+":latest", pluginLatestDir.Path()).Assert(t, icmd.Success)
				icmd.RunCommand("docker", "plugin", "create", plugin+":next", pluginNextDir.Path()).Assert(t, icmd.Success)
				icmd.RunCmd(icmd.Command("docker", "plugin", "push", plugin+":latest"), defaultCmdOpts...).Assert(t, icmd.Success)
				icmd.RunCmd(icmd.Command("docker", "plugin", "push", plugin+":next"), defaultCmdOpts...).Assert(t, icmd.Success)
				icmd.RunCmd(icmd.Command("docker", "plugin", "rm", "-f", plugin+":latest"), defaultCmdOpts...).Assert(t, icmd.Success)
				icmd.RunCmd(icmd.Command("docker", "plugin", "rm", "-f", plugin+":next"), defaultCmdOpts...).Assert(t, icmd.Success)
				icmd.RunCmd(icmd.Command("docker", "plugin", "install", "--disable", "--grant-all-permissions", plugin+":latest"), defaultCmdOpts...).Assert(t, icmd.Success)
				return icmd.Command("docker", "plugin", "upgrade", plugin+":latest", plugin+":next")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buf := new(bytes.Buffer)
			bufioWriter := bufio.NewWriter(buf)

			writeDone := make(chan struct{})
			w := test.NewWriterWithHook(bufioWriter, func(p []byte) {
				writeDone <- struct{}{}
			})

			drainChCtx, drainChCtxCancel := context.WithCancel(ctx)
			t.Cleanup(drainChCtxCancel)

			drainChannel(drainChCtx, writeDone)

			r, _ := io.Pipe()
			defer r.Close()
			result := icmd.StartCmd(tc.run(t),
				append(defaultCmdOpts,
					icmd.WithStdout(w),
					icmd.WithStderr(w),
					icmd.WithStdin(r))...)

			poll.WaitOn(t, func(t poll.LogT) poll.Result {
				select {
				case <-ctx.Done():
					return poll.Error(ctx.Err())
				default:

					if err := bufioWriter.Flush(); err != nil {
						return poll.Continue("%v", err)
					}
					if strings.Contains(buf.String(), "[y/N]") {
						return poll.Success()
					}

					return poll.Continue("command did not prompt for confirmation, instead prompted:\n%s\n", buf.String())
				}
			}, poll.WithDelay(100*time.Millisecond), poll.WithTimeout(1*time.Second))

			drainChCtxCancel()

			assert.NilError(t, result.Cmd.Process.Signal(syscall.SIGINT))

			proc, err := result.Cmd.Process.Wait()
			assert.NilError(t, err)
			assert.Equal(t, proc.ExitCode(), 0, "expected exit code to be 0, got %d", proc.ExitCode())

			processCtx, processCtxCancel := context.WithTimeout(ctx, time.Second)
			t.Cleanup(processCtxCancel)

			select {
			case <-processCtx.Done():
				t.Fatal("timed out waiting for new line after process exit")
			case <-writeDone:
				buf.Reset()
				assert.NilError(t, bufioWriter.Flush())
				assert.Assert(t, strings.HasSuffix(buf.String(), "\n"), "expected a new line after the process exits from SIGINT")
			}
		})
	}
}

func drainChannel(ctx context.Context, ch <-chan struct{}) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
			}
		}
	}()
}
