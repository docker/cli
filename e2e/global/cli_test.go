package global

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test/environment"
	"gotest.tools/v3/assert"
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
