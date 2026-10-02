package registry

import (
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

const envAuthConfig = `{"auths":{"env.example.test":{"auth":"ZW52X3VzZXI6ZW52X3Bhc3M="}}}`

func TestMaybePrintEnvAuthWarning(t *testing.T) {
	t.Run("warns when environment credentials take precedence", func(t *testing.T) {
		cli := test.NewFakeCli(&fakeClient{})
		t.Setenv(configfile.DockerEnvConfigKey, envAuthConfig)

		maybePrintEnvAuthWarning(cli)

		assert.Check(t, is.Contains(cli.ErrBuffer().String(), "DOCKER_AUTH_CONFIG is set and takes precedence"))
	})

	t.Run("does not warn in GitLab CI", func(t *testing.T) {
		cli := test.NewFakeCli(&fakeClient{})
		t.Setenv(configfile.DockerEnvConfigKey, envAuthConfig)
		t.Setenv("GITLAB_CI", "true")

		maybePrintEnvAuthWarning(cli)

		assert.Check(t, is.Equal(cli.ErrBuffer().String(), ""))
	})
}
