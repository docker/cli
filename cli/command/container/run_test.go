package container

import (
	"context"
	"errors"
	"io"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunLabel(t *testing.T) {
	fakeCLI := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(_ *container.Config, _ *container.HostConfig, _ *network.NetworkingConfig, _ *specs.Platform, _ string) (container.CreateResponse, error) {
			return container.CreateResponse{
				ID: "id",
			}, nil
		},
		Version: "1.36",
	})
	cmd := NewRunCommand(fakeCLI)
	cmd.SetArgs([]string{"--detach=true", "--label", "foo", "busybox"})
	assert.NilError(t, cmd.Execute())
}

func TestRunAttach(t *testing.T) {
	p, tty, err := pty.Open()
	assert.NilError(t, err)
	defer func() {
		_ = tty.Close()
		_ = p.Close()
	}()

	var conn net.Conn
	attachCh := make(chan struct{})
	fakeCLI := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(_ *container.Config, _ *container.HostConfig, _ *network.NetworkingConfig, _ *specs.Platform, _ string) (container.CreateResponse, error) {
			return container.CreateResponse{
				ID: "id",
			}, nil
		},
		containerAttachFunc: func(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error) {
			server, client := net.Pipe()
			conn = server
			t.Cleanup(func() {
				_ = server.Close()
			})
			attachCh <- struct{}{}
			return types.NewHijackedResponse(client, types.MediaTypeRawStream), nil
		},
		waitFunc: func(_ string) (<-chan container.WaitResponse, <-chan error) {
			responseChan := make(chan container.WaitResponse, 1)
			errChan := make(chan error)

			responseChan <- container.WaitResponse{
				StatusCode: 33,
			}
			return responseChan, errChan
		},
		// use new (non-legacy) wait API
		// see: 38591f20d07795aaef45d400df89ca12f29c603b
		Version: "1.30",
	}, func(fc *test.FakeCli) {
		fc.SetOut(streams.NewOut(tty))
		fc.SetIn(streams.NewIn(tty))
	})

	cmd := NewRunCommand(fakeCLI)
	cmd.SetArgs([]string{"-it", "busybox"})
	cmd.SilenceUsage = true
	cmdErrC := make(chan error, 1)
	go func() {
		cmdErrC <- cmd.Execute()
	}()

	// run command should attempt to attach to the container
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("containerAttachFunc was not called before the 5 second timeout")
	case <-attachCh:
	}

	// end stream from "container" so that we'll detach
	conn.Close()

	select {
	case cmdErr := <-cmdErrC:
		assert.Equal(t, cmdErr, cli.StatusError{
			StatusCode: 33,
		})
	case <-time.After(2 * time.Second):
		t.Fatal("cmd did not return within timeout")
	}
}

func TestRunAttachTermination(t *testing.T) {
	p, tty, err := pty.Open()
	assert.NilError(t, err)
	defer func() {
		_ = tty.Close()
		_ = p.Close()
	}()

	var conn net.Conn
	killCh := make(chan struct{})
	attachCh := make(chan struct{})
	fakeCLI := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(_ *container.Config, _ *container.HostConfig, _ *network.NetworkingConfig, _ *specs.Platform, _ string) (container.CreateResponse, error) {
			return container.CreateResponse{
				ID: "id",
			}, nil
		},
		containerKillFunc: func(ctx context.Context, containerID, signal string) error {
			killCh <- struct{}{}
			return nil
		},
		containerAttachFunc: func(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error) {
			server, client := net.Pipe()
			conn = server
			t.Cleanup(func() {
				_ = server.Close()
			})
			attachCh <- struct{}{}
			return types.NewHijackedResponse(client, types.MediaTypeRawStream), nil
		},
		waitFunc: func(_ string) (<-chan container.WaitResponse, <-chan error) {
			responseChan := make(chan container.WaitResponse, 1)
			errChan := make(chan error)

			responseChan <- container.WaitResponse{
				StatusCode: 130,
			}
			return responseChan, errChan
		},
		// use new (non-legacy) wait API
		// see: 38591f20d07795aaef45d400df89ca12f29c603b
		Version: "1.30",
	}, func(fc *test.FakeCli) {
		fc.SetOut(streams.NewOut(tty))
		fc.SetIn(streams.NewIn(tty))
	})

	cmd := NewRunCommand(fakeCLI)
	cmd.SetArgs([]string{"-it", "busybox"})
	cmd.SilenceUsage = true
	cmdErrC := make(chan error, 1)
	go func() {
		cmdErrC <- cmd.Execute()
	}()

	// run command should attempt to attach to the container
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("containerAttachFunc was not called before the timeout")
	case <-attachCh:
	}

	assert.NilError(t, syscall.Kill(syscall.Getpid(), syscall.SIGINT))
	// end stream from "container" so that we'll detach
	conn.Close()

	select {
	case <-killCh:
	case <-time.After(5 * time.Second):
		t.Fatal("containerKillFunc was not called before the timeout")
	}

	select {
	case cmdErr := <-cmdErrC:
		assert.Equal(t, cmdErr, cli.StatusError{
			StatusCode: 130,
		})
	case <-time.After(2 * time.Second):
		t.Fatal("cmd did not return before the timeout")
	}
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fakeCLI := test.NewFakeCli(&fakeClient{
				createContainerFunc: func(config *container.Config,
					hostConfig *container.HostConfig,
					networkingConfig *network.NetworkingConfig,
					platform *specs.Platform,
					containerName string,
				) (container.CreateResponse, error) {
					return container.CreateResponse{}, errors.New("shouldn't try to pull image")
				},
			}, test.EnableContentTrust)
			fakeCLI.SetNotaryClient(tc.notaryFunc)
			cmd := NewRunCommand(fakeCLI)
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err := cmd.Execute()
			statusErr := cli.StatusError{}
			assert.Check(t, errors.As(err, &statusErr))
			assert.Check(t, is.Equal(statusErr.StatusCode, 125))
			assert.Check(t, is.ErrorContains(err, tc.expectedError))
		})
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
				context.TODO(),
				dockerCli,
				&pflag.FlagSet{},
				&runOptions{createOptions: createOptions{pull: tc.PullPolicy}},
				&containerOptions{},
			)

			statusErr := cli.StatusError{}
			assert.Check(t, errors.As(err, &statusErr))
			assert.Check(t, is.Equal(statusErr.StatusCode, 125))
			assert.Check(t, is.ErrorContains(err, tc.ExpectedErrMsg))
		})
	}
}
