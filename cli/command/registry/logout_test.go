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
// Credentials for the default registry are stored under
// [registry.IndexServer] regardless of the spelling passed to "docker login",
// so logout has to look for that key as well to find them again.
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
