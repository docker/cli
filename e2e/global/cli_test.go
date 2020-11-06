package global

import (
	"testing"

	"github.com/docker/cli/internal/test/environment"
	"gotest.tools/v3/icmd"
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
