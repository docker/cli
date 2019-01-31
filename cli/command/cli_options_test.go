package command

import (
	"os"
	"testing"

	"gotest.tools/assert"
)

func contentTrustEnabled(t *testing.T) bool {
	var cli DockerCli
	assert.NilError(t, WithContentTrustFromEnv()(&cli))
	return cli.contentTrust
}

// NB: Do not t.Parallel() this test -- it messes with the process environment.
func TestWithContentTrustFromEnv(t *testing.T) {
	envvar := "DOCKER_CONTENT_TRUST"
	if orig, ok := os.LookupEnv(envvar); ok {
		defer func() {
			os.Setenv(envvar, orig)
		}()
	} else {
		defer func() {
			os.Unsetenv(envvar)
		}()
	}

	os.Setenv(envvar, "true")
	assert.Assert(t, contentTrustEnabled(t))
	os.Setenv(envvar, "false")
	assert.Assert(t, !contentTrustEnabled(t))
	os.Setenv(envvar, "invalid")
	assert.Assert(t, contentTrustEnabled(t))
	os.Unsetenv(envvar)
	assert.Assert(t, !contentTrustEnabled(t))
}
