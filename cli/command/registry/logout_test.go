package registry

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	registrytypes "github.com/docker/docker/api/types/registry"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func (c fakeClient) RegistryLogout(ctx context.Context, auth types.AuthConfig) (registrytypes.AuthenticateOKBody, error) {
	if auth.Password == expiredPassword {
		return registrytypes.AuthenticateOKBody{}, fmt.Errorf("Invalid Username or Password")
	}
	if auth.Password == useToken {
		return registrytypes.AuthenticateOKBody{
			IdentityToken: auth.Password,
		}, nil
	}
	err := testAuthErrors[auth.Username]
	return registrytypes.AuthenticateOKBody{}, err
}

func TestLogoutWithCredStoreCreds(t *testing.T) {
	testCases := []struct {
		warningCount int
		serverURL    string
	}{
		{
			serverURL:    "registry",
			warningCount: 1,
		},
		{
			serverURL:    "badregistry",
			warningCount: 0,
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})

		configStr := `
		{
			"auths": {
				"registry": {}
			},
			"HttpHeaders": {
				"User-Agent": "Docker-Client/18.09.7 (linux)"
			}
		}
		`

		errBuf := new(bytes.Buffer)
		cli.SetErr(errBuf)

		configReader := bytes.NewReader([]byte(configStr))
		cli.ConfigFile().LoadFromReader(configReader)

		runLogout(cli, tc.serverURL)
		errorString := errBuf.String()

		//We will fail since the file store will fail to delete
		//the file. We only only want one warning to ensure we
		//only logout once and not twice.
		warningCount := wordCount(errorString, "WARNING:")
		assert.Check(t, is.Equal(tc.warningCount, warningCount), "Unexpected number of warnings")
	}
}

func wordCount(s string, w string) int {
	var count, index, wlen int

	wlen = len(w)
	index = strings.Index(s, w)

	for {
		if index == -1 {
			break
		}
		index += wlen
		s = s[index:]
		index = strings.Index(s, w)
		count++
	}
	return count
}
