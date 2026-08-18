package registry

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// TestLogoutRemovesCredentialsStoredByLogin verifies that logging out with the
// same argument that was used to log in removes the stored credentials.
//
// "docker.io" is normalized to [registry.IndexServer] when logging in, so
// logout must apply the same normalization to find the entry again.
func TestLogoutRemovesCredentialsStoredByLogin(t *testing.T) {
	for _, serverAddress := range []string{
		"",
		"docker.io",
		"index.docker.io",
		"https://index.docker.io/v1/",
		"myreg.example.com",
	} {
		name := serverAddress
		if name == "" {
			name = "no server address"
		}
		t.Run(name, func(t *testing.T) {
			configFile := configfile.New(filepath.Join(t.TempDir(), "config.json"))
			cli := test.NewFakeCli(&fakeClient{})
			cli.SetConfigFile(configFile)

			err := runLogin(context.Background(), cli, loginOptions{
				serverAddress: serverAddress,
				user:          "my-username",
				password:      "my-password",
			})
			assert.NilError(t, err)
			assert.Assert(t, is.Len(configFile.AuthConfigs, 1), "login did not store credentials")

			err = runLogout(context.Background(), cli, serverAddress)
			assert.NilError(t, err)
			assert.Check(t, is.Len(configFile.AuthConfigs, 0))
		})
	}
}
