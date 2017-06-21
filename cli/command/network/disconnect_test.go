package network

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/pkg/testutil"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

func TestNetworkDisconnectErrors(t *testing.T) {
	testCases := []struct {
		args                  []string
		flags                 map[string]string
		networkDisconnectFunc func(ctx context.Context, networkID, container string, force bool) error
		expectedError         string
	}{
		{
			expectedError: "requires exactly 2 argument(s)",
		},
		{
			args: []string{"toto", "titi"},
			networkDisconnectFunc: func(ctx context.Context, networkID, container string, force bool) error {
				return errors.Errorf("error disconnecting network")
			},
			expectedError: "error disconnecting network",
		},
	}

	for _, tc := range testCases {
		buf := new(bytes.Buffer)
		cmd := newConnectCommand(
			test.NewFakeCli(&fakeClient{
				networkDisconnectFunc: tc.networkDisconnectFunc,
			}, buf),
		)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}
