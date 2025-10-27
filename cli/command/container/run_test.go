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
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/progress"
	"github.com/moby/moby/client/pkg/streamformatter"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunValidateFlags(t *testing.T) {
	for _, tc := range []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "with conflicting --attach, --detach",
			args:        []string{"--attach", "stdin", "--detach", "myimage"},
			expectedErr: "conflicting options: cannot specify both --attach and --detach",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRunCommand(test.NewFakeCli(&fakeClient{}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			} else {
				assert.Check(t, is.Nil(err))
			}
		})
	}
}

func TestRunLabel(t *testing.T) {
	fakeCLI := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
			return client.ContainerCreateResult{ID: "id"}, nil
		},
		Version: client.MaxAPIVersion,
	})
	cmd := newRunCommand(fakeCLI)
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
		createContainerFunc: func(options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
			return client.ContainerCreateResult{ID: "id"}, nil
		},
		containerAttachFunc: func(ctx context.Context, containerID string, options client.ContainerAttachOptions) (client.ContainerAttachResult, error) {
			server, clientConn := net.Pipe()
			conn = server
			t.Cleanup(func() {
				_ = server.Close()
			})
			attachCh <- struct{}{}
			return client.ContainerAttachResult{
				HijackedResponse: client.NewHijackedResponse(clientConn, types.MediaTypeRawStream),
			}, nil
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
		// see: https://github.com/docker/cli/commit/38591f20d07795aaef45d400df89ca12f29c603b
		Version: client.MaxAPIVersion,
	}, func(fc *test.FakeCli) {
		fc.SetOut(streams.NewOut(tty))
		fc.SetIn(streams.NewIn(tty))
	})

	cmd := newRunCommand(fakeCLI)
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
		createContainerFunc: func(options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
			return client.ContainerCreateResult{ID: "id"}, nil
		},
		containerKillFunc: func(ctx context.Context, container string, options client.ContainerKillOptions) (client.ContainerKillResult, error) {
			if options.Signal == "TERM" {
				close(killCh)
			}
			return client.ContainerKillResult{}, nil
		},
		containerAttachFunc: func(ctx context.Context, containerID string, options client.ContainerAttachOptions) (client.ContainerAttachResult, error) {
			server, clientConn := net.Pipe()
			conn = server
			t.Cleanup(func() {
				_ = server.Close()
			})
			attachCh <- struct{}{}
			return client.ContainerAttachResult{
				HijackedResponse: client.NewHijackedResponse(clientConn, types.MediaTypeRawStream),
			}, nil
		},
		waitFunc: func(_ string) (<-chan container.WaitResponse, <-chan error) {
			responseChan := make(chan container.WaitResponse, 1)
			errChan := make(chan error)
			<-killCh
			responseChan <- container.WaitResponse{
				StatusCode: 130,
			}
			return responseChan, errChan
		},
		// use new (non-legacy) wait API
		// see: https://github.com/docker/cli/commit/38591f20d07795aaef45d400df89ca12f29c603b
		Version: client.MaxAPIVersion,
	}, func(fc *test.FakeCli) {
		fc.SetOut(streams.NewOut(tty))
		fc.SetIn(streams.NewIn(tty))
	})

	cmd := newRunCommand(fakeCLI)
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

	assert.NilError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
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

func TestRunPullTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	attachCh := make(chan struct{})
	fakeCLI := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
			return client.ContainerCreateResult{}, errors.New("shouldn't try to create a container")
		},
		containerAttachFunc: func(ctx context.Context, containerID string, options client.ContainerAttachOptions) (client.ContainerAttachResult, error) {
			return client.ContainerAttachResult{}, errors.New("shouldn't try to attach to a container")
		},
		imageCreateFunc: func(ctx context.Context, parentReference string, options client.ImageCreateOptions) (client.ImageCreateResult, error) {
			server, respReader := net.Pipe()
			t.Cleanup(func() {
				_ = server.Close()
			})
			go func() {
				id := test.RandomID()[:12] // short-ID
				progressOutput := streamformatter.NewJSONProgressOutput(server, true)
				for i := 0; i < 100; i++ {
					select {
					case <-ctx.Done():
						assert.NilError(t, server.Close(), "failed to close imageCreateFunc server")
						return
					default:
						assert.NilError(t, progressOutput.WriteProgress(progress.Progress{
							ID:      id,
							Message: "Downloading",
							Current: int64(i),
							Total:   100,
						}))
						time.Sleep(100 * time.Millisecond)
					}
				}
			}()
			attachCh <- struct{}{}
			return client.ImageCreateResult{Body: respReader}, nil
		},
		Version: client.MaxAPIVersion,
	})

	cmd := newRunCommand(fakeCLI)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--pull", "always", "foobar:latest"})

	cmdErrC := make(chan error, 1)
	go func() {
		cmdErrC <- cmd.ExecuteContext(ctx)
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatal("imageCreateFunc was not called before the timeout")
	case <-attachCh:
	}

	cancel()

	select {
	case cmdErr := <-cmdErrC:
		assert.Equal(t, cmdErr, cli.StatusError{
			Cause:      context.Canceled,
			StatusCode: 125,
			Status:     "docker: context canceled\n\nRun 'docker run --help' for more information",
		})
	case <-time.After(10 * time.Second):
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
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCKER_CONTENT_TRUST", "true")
			fakeCLI := test.NewFakeCli(&fakeClient{
				createContainerFunc: func(options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
					return client.ContainerCreateResult{}, errors.New("shouldn't try to pull image")
				},
			})
			fakeCLI.SetNotaryClient(tc.notaryFunc)
			cmd := newRunCommand(fakeCLI)
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
