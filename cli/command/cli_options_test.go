package command

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func contentTrustEnabled(t *testing.T) bool {
	var cli DockerCli
	assert.NilError(t, WithContentTrustFromEnv()(&cli))
	return cli.contentTrust
}

// NB: Do not t.Parallel() this test -- it messes with the process environment.
func TestWithContentTrustFromEnv(t *testing.T) {
	const envvar = "DOCKER_CONTENT_TRUST"
	t.Setenv(envvar, "true")
	assert.Check(t, contentTrustEnabled(t))
	t.Setenv(envvar, "false")
	assert.Check(t, !contentTrustEnabled(t))
	t.Setenv(envvar, "invalid")
	assert.Check(t, contentTrustEnabled(t))
	os.Unsetenv(envvar)
	assert.Check(t, !contentTrustEnabled(t))
}
