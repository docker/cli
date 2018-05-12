package command_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"

	// Prevents a circular import with "github.com/docker/cli/internal/test"

	. "github.com/docker/cli/cli/command"
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

func TestElectAuthServer(t *testing.T) {
	testCases := []struct {
		expectedAuthServer string
		expectedWarning    string
		infoFunc           func() (types.Info, error)
	}{
		{
			expectedAuthServer: "https://index.docker.io/v1/",
			expectedWarning:    "",
			infoFunc: func() (types.Info, error) {
				return types.Info{IndexServerAddress: "https://index.docker.io/v1/"}, nil
			},
		},
		{
			expectedAuthServer: "https://index.docker.io/v1/",
			expectedWarning:    "Empty registry endpoint from daemon",
			infoFunc: func() (types.Info, error) {
				return types.Info{IndexServerAddress: ""}, nil
			},
		},
		{
			expectedAuthServer: "https://foo.bar",
			expectedWarning:    "",
			infoFunc: func() (types.Info, error) {
				return types.Info{IndexServerAddress: "https://foo.bar"}, nil
			},
		},
		{
			expectedAuthServer: "https://index.docker.io/v1/",
			expectedWarning:    "failed to get default registry endpoint from daemon",
			infoFunc: func() (types.Info, error) {
				return types.Info{}, errors.Errorf("error getting info")
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{infoFunc: tc.infoFunc})
		server := ElectAuthServer(context.Background(), cli)
		assert.Check(t, is.Equal(tc.expectedAuthServer, server))
		actual := cli.ErrBuffer().String()
		if tc.expectedWarning == "" {
			assert.Check(t, is.Len(actual, 0))
		} else {
			assert.Check(t, is.Contains(actual, tc.expectedWarning))
		}
	}
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
		cli.ConfigFile().GetCredentialsStore(authconfig.ServerAddress).Store(authconfig)
	}
	for _, tc := range testCases {
		serverAddress := tc.inputServerAddress
		authconfig, err := GetDefaultAuthConfig(cli, tc.checkCredStore, serverAddress, serverAddress == "https://index.docker.io/v1/")
		if tc.expectedErr != "" {
			assert.Check(t, err != nil)
			assert.Check(t, is.Equal(tc.expectedErr, err.Error()))
		} else {
			assert.NilError(t, err)
			assert.Check(t, is.DeepEqual(tc.expectedAuthConfig, *authconfig))
		}
	}
}
