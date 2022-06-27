package container

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunLabel(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(_ *container.Config, _ *container.HostConfig, _ *network.NetworkingConfig, _ *specs.Platform, _ string) (container.CreateResponse, error) {
			return container.CreateResponse{
				ID: "id",
			}, nil
		},
		Version: "1.36",
	})
	cmd := NewRunCommand(cli)
	cmd.SetArgs([]string{"--detach=true", "--label", "foo", "busybox"})
	assert.NilError(t, cmd.Execute())
}

func TestRunCommandWithContentTrustErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
		notaryFunc    test.NotaryClientFuncType
	}{
		{
			name:          "offline-notary-server",
			notaryFunc:    notary.GetOfflineNotaryRepository,
			expectedError: "client is offline",
			args:          []string{"image:tag"},
		},
		{
			name:          "uninitialized-notary-server",
			notaryFunc:    notary.GetUninitializedNotaryRepository,
			expectedError: "remote trust data does not exist",
			args:          []string{"image:tag"},
		},
		{
			name:          "empty-notary-server",
			notaryFunc:    notary.GetEmptyTargetsNotaryRepository,
			expectedError: "No valid trust data for tag",
			args:          []string{"image:tag"},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			createContainerFunc: func(config *container.Config,
				hostConfig *container.HostConfig,
				networkingConfig *network.NetworkingConfig,
				platform *specs.Platform,
				containerName string,
			) (container.CreateResponse, error) {
				return container.CreateResponse{}, fmt.Errorf("shouldn't try to pull image")
			},
		}, test.EnableContentTrust)
		cli.SetNotaryClient(tc.notaryFunc)
		cmd := NewRunCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		err := cmd.Execute()
		assert.Assert(t, err != nil)
		assert.Assert(t, is.Contains(cli.ErrBuffer().String(), tc.expectedError))
	}
}

func TestRunContainerImagePullPolicyInvalid(t *testing.T) {
	cases := []struct {
		PullPolicy     string
		ExpectedErrMsg string
	}{
		{
			PullPolicy:     "busybox:latest",
			ExpectedErrMsg: `invalid pull option: 'busybox:latest': must be one of "always", "missing" or "never"`,
		},
		{
			PullPolicy:     "--network=foo",
			ExpectedErrMsg: `invalid pull option: '--network=foo': must be one of "always", "missing" or "never"`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.PullPolicy, func(t *testing.T) {
			dockerCli := test.NewFakeCli(&fakeClient{})
			err := runRun(
				dockerCli,
				&pflag.FlagSet{},
				&runOptions{createOptions: createOptions{pull: tc.PullPolicy}},
				&containerOptions{},
			)

			statusErr := cli.StatusError{}
			assert.Check(t, errors.As(err, &statusErr))
			assert.Equal(t, statusErr.StatusCode, 125)
			assert.Check(t, is.Contains(dockerCli.ErrBuffer().String(), tc.ExpectedErrMsg))
		})
	}
}
