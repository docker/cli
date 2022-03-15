package command_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	// Prevents a circular import with "github.com/docker/cli/internal/test"

	. "github.com/docker/cli/cli/command"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	infoFunc func() (types.Info, error)
}

var testAuthConfigs = []types.AuthConfig{
	{
		ServerAddress: "https://index.docker.io/v1/",
		Username:      "u0",
		Password:      "p0",
	},
	{
		ServerAddress: "server1.io",
		Username:      "u1",
		Password:      "p1",
	},
}

func (cli *fakeClient) Info(_ context.Context) (types.Info, error) {
	if cli.infoFunc != nil {
		return cli.infoFunc()
	}
	return types.Info{}, nil
}

func TestGetDefaultAuthConfig(t *testing.T) {
	testCases := []struct {
		checkCredStore     bool
		inputServerAddress string
		expectedErr        string
		expectedAuthConfig types.AuthConfig
	}{
		{
			checkCredStore:     false,
			inputServerAddress: "",
			expectedErr:        "",
			expectedAuthConfig: types.AuthConfig{
				ServerAddress: "",
				Username:      "",
				Password:      "",
			},
		},
		{
			checkCredStore:     true,
			inputServerAddress: testAuthConfigs[0].ServerAddress,
			expectedErr:        "",
			expectedAuthConfig: testAuthConfigs[0],
		},
		{
			checkCredStore:     true,
			inputServerAddress: testAuthConfigs[1].ServerAddress,
			expectedErr:        "",
			expectedAuthConfig: testAuthConfigs[1],
		},
		{
			checkCredStore:     true,
			inputServerAddress: fmt.Sprintf("https://%s", testAuthConfigs[1].ServerAddress),
			expectedErr:        "",
			expectedAuthConfig: testAuthConfigs[1],
		},
	}
	cli := test.NewFakeCli(&fakeClient{})
	errBuf := new(bytes.Buffer)
	cli.SetErr(errBuf)
	for _, authconfig := range testAuthConfigs {
		cli.ConfigFile().GetCredentialsStore(authconfig.ServerAddress).Store(configtypes.AuthConfig(authconfig))
	}
	for _, tc := range testCases {
		serverAddress := tc.inputServerAddress
		authconfig, err := GetDefaultAuthConfig(cli, tc.checkCredStore, serverAddress, serverAddress == "https://index.docker.io/v1/")
		if tc.expectedErr != "" {
			assert.Check(t, err != nil)
			assert.Check(t, is.Equal(tc.expectedErr, err.Error()))
		} else {
			assert.NilError(t, err)
			assert.Check(t, is.DeepEqual(tc.expectedAuthConfig, authconfig))
		}
	}
}

func TestGetDefaultAuthConfig_HelperError(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	errBuf := new(bytes.Buffer)
	cli.SetErr(errBuf)
	cli.ConfigFile().CredentialsStore = "fake-does-not-exist"
	serverAddress := "test-server-address"
	expectedAuthConfig := types.AuthConfig{
		ServerAddress: serverAddress,
	}
	authconfig, err := GetDefaultAuthConfig(cli, true, serverAddress, serverAddress == "https://index.docker.io/v1/")
	assert.Check(t, is.DeepEqual(expectedAuthConfig, authconfig))
	assert.Check(t, is.ErrorContains(err, "docker-credential-fake-does-not-exist"))
}
