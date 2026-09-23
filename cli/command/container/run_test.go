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
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
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
		{
			name:        "with invalid --detach-keys",
			args:        []string{"--detach-keys", "shift-a", "myimage"},
			expectedErr: "invalid detach keys (shift-a):",
		},
		{
			name:        "with --start-healthy-timeout without --detach",
			args:        []string{"--start-healthy-timeout", "30s", "myimage"},
			expectedErr: "--start-healthy-timeout can only be used with --detach",
		},
		{
			name:        "with negative --start-healthy-timeout",
			args:        []string{"--detach", "--start-healthy-timeout", "-30s", "myimage"},
			expectedErr: "--start-healthy-timeout cannot be negative",
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

func TestRunStartHealthyTimeout(t *testing.T) {
	for _, tc := range []struct {
		name        string
		state       container.State
		expectedErr string
	}{
		{
			name:  "healthy",
			state: container.State{Running: true, Health: &container.Health{Status: container.Healthy}},
		},
		{
			name:        "not healthy in time",
			state:       container.State{Running: true, Health: &container.Health{Status: container.Starting}},
			expectedErr: "container id did not become healthy within 100ms",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fakeCLI := test.NewFakeCli(&fakeClient{
				createContainerFunc: func(client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
					return client.ContainerCreateResult{ID: "id"}, nil
				},
				inspectFunc: func(string) (client.ContainerInspectResult, error) {
					return client.ContainerInspectResult{
						Container: container.InspectResponse{ID: "id", State: &tc.state},
					}, nil
				},
				containerStopFunc: func(context.Context, string, client.ContainerStopOptions) (client.ContainerStopResult, error) {
					t.Error("container should not be stopped")
					return client.ContainerStopResult{}, nil
				},
				Version: client.MaxAPIVersion,
			})
			cmd := newRunCommand(fakeCLI)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"--detach", "--start-healthy-timeout", "100ms", "busybox"})

			err := cmd.Execute()
			if tc.expectedErr == "" {
				assert.NilError(t, err)
			} else {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			}
			// The container ID is printed as soon as the container is started,
			// regardless of whether it becomes healthy.
			assert.Check(t, is.Equal(fakeCLI.OutBuffer().String(), "id\n"))
		})
	}
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
		waitFunc: func(_ string) client.ContainerWaitResult {
			responseChan := make(chan container.WaitResponse, 1)
			errChan := make(chan error)

			responseChan <- container.WaitResponse{
				StatusCode: 33,
			}
			return client.ContainerWaitResult{
				Result: responseChan,
				Error:  errChan,
			}
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
	assert.NilError(t, conn.Close())

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
		waitFunc: func(_ string) client.ContainerWaitResult {
			responseChan := make(chan container.WaitResponse, 1)
			errChan := make(chan error)
			<-killCh
			responseChan <- container.WaitResponse{
				StatusCode: 130,
			}
			return client.ContainerWaitResult{
				Result: responseChan,
				Error:  errChan,
			}
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
	assert.NilError(t, conn.Close())

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
		imagePullFunc: func(ctx context.Context, parentReference string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
			server, respReader := net.Pipe()
			t.Cleanup(func() {
				_ = server.Close()
			})
			go func() {
				for range 100 {
					select {
					case <-ctx.Done():
						assert.NilError(t, server.Close(), "failed to close imageCreateFunc server")
						return
					default:
						time.Sleep(100 * time.Millisecond)
					}
				}
			}()
			attachCh <- struct{}{}
			return fakeStreamResult{ReadCloser: respReader}, nil
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
