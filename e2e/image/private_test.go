package image

import (
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/e2e/internal/fixtures"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
)

const privateRegistryPrefix = "privateregistry:5001"

// Regression test for https://github.com/docker/cli/issues/5963
func TestPullPushPrivateRepository(t *testing.T) {
	t.Parallel()

	dir := fixtures.SetupConfigFile(t)
	t.Cleanup(dir.Remove)
	emptyConfigDir := t.TempDir()

	sourceImage := fixtures.AlpineImage
	privateImage := privateRegistryPrefix + "/private/alpine:test-private-pull-push"

	runWithPrivateRegistryRetry(t,
		icmd.Command("docker", "pull", sourceImage),
	).Assert(t, icmd.Success)
	t.Cleanup(func() {
		icmd.RunCommand("docker", "image", "rm", "-f", privateImage).Assert(t, icmd.Success)
	})

	icmd.RunCommand("docker", "tag", sourceImage, privateImage).Assert(t, icmd.Success)

	pushNoAuth := runWithPrivateRegistryRetry(t,
		icmd.Command("docker", "push", privateImage),
		fixtures.WithConfig(emptyConfigDir),
	)
	pushNoAuth.Assert(t, icmd.Expected{ExitCode: 1})
	assertAuthDenied(t, pushNoAuth)

	pushWithAuth := runWithPrivateRegistryRetry(t,
		icmd.Command("docker", "push", privateImage),
		fixtures.WithConfig(dir.Path()),
	)
	pushWithAuth.Assert(t, icmd.Success)
	assert.Check(t, strings.Contains(pushWithAuth.Combined(), "The push refers to repository ["+privateImage+"]"), pushWithAuth.Combined())

	icmd.RunCommand("docker", "image", "rm", "-f", privateImage).Assert(t, icmd.Success)

	pullNoAuth := runWithPrivateRegistryRetry(t,
		icmd.Command("docker", "pull", privateImage),
		fixtures.WithConfig(emptyConfigDir),
	)
	pullNoAuth.Assert(t, icmd.Expected{ExitCode: 1})
	assertAuthDenied(t, pullNoAuth)

	pullWithAuth := runWithPrivateRegistryRetry(t,
		icmd.Command("docker", "pull", privateImage),
		fixtures.WithConfig(dir.Path()),
	)
	pullWithAuth.Assert(t, icmd.Success)
	assert.Check(t, strings.Contains(pullWithAuth.Combined(), privateImage), pullWithAuth.Combined())
}

func assertAuthDenied(t *testing.T, result *icmd.Result) {
	t.Helper()
	output := result.Combined()
	if isPrivateRegistryTransient(output) {
		t.Fatalf("private registry unavailable while expecting auth failure: %s", output)
	}

	assert.Check(t,
		strings.Contains(output, "requested access to the resource is denied") ||
			strings.Contains(output, "no basic auth credentials") ||
			strings.Contains(output, "unauthorized") ||
			strings.Contains(output, "authentication required"),
		output,
	)
}

func runWithPrivateRegistryRetry(t *testing.T, cmd icmd.Cmd, opts ...icmd.CmdOp) *icmd.Result {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
	for {
		result := icmd.RunCmd(cmd, opts...)
		output := result.Combined()
		if isPrivateRegistryTransient(output) {
			if time.Now().Before(deadline) {
				t.Logf("waiting for private registry availability: %s", output)
				time.Sleep(500 * time.Millisecond)
				continue
			}
		}
		return result
	}
}

func isPrivateRegistryTransient(output string) bool {
	return strings.Contains(output, "lookup privateregistry") ||
		strings.Contains(output, "lookup registry") ||
		strings.Contains(output, "no such host") ||
		strings.Contains(output, "server misbehaving") ||
		strings.Contains(output, "Temporary failure in name resolution") ||
		strings.Contains(output, "connection refused") ||
		strings.Contains(output, "i/o timeout") ||
		strings.Contains(output, "TLS handshake timeout") ||
		strings.Contains(output, "context deadline exceeded") ||
		strings.Contains(output, "connection reset by peer") ||
		strings.Contains(output, "unexpected EOF")
}
