package network

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func TestNetworkDisconnectErrors(t *testing.T) {
	testCases := []struct {
		args                  []string
		networkDisconnectFunc func(ctx context.Context, networkID, container string, force bool) error
		expectedError         string
	}{
		{
			expectedError: "requires 2 arguments",
		},
		{
			args: []string{"toto", "titi"},
			networkDisconnectFunc: func(ctx context.Context, networkID, container string, force bool) error {
				return errors.New("error disconnecting network")
			},
			expectedError: "error disconnecting network",
		},
	}

	for _, tc := range testCases {
		cmd := newDisconnectCommand(
			test.NewFakeCli(&fakeClient{
				networkDisconnectFunc: tc.networkDisconnectFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}
