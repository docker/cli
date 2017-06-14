package registry

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	infoFunc func() (types.Info, error)
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
		c := &fakeClient{infoFunc: tc.infoFunc}
		server, warns, err := ElectAuthServer(context.Background(), c)
		assert.Equal(t, tc.expectedAuthServer, server)
		assert.Nil(t, err)
		if tc.expectedWarning == "" {
			assert.Nil(t, warns)
		} else {
			joinedWarns := ""
			for _, w := range warns {
				joinedWarns += w.Error() + "\n"
			}
			assert.Contains(t, joinedWarns, tc.expectedWarning)
		}
	}
}
