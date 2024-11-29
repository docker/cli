package container

import (
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/google/go-cmp/cmp"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/golden"
)

func TestCIDFileNoOPWithNoFilename(t *testing.T) {
	file, err := newCIDFile("")
	assert.NilError(t, err)
	assert.DeepEqual(t, &cidFile{}, file, cmp.AllowUnexported(cidFile{}))

	assert.NilError(t, file.Write("id"))
	assert.NilError(t, file.Close())
}

func TestNewCIDFileWhenFileAlreadyExists(t *testing.T) {
	tempfile := fs.NewFile(t, "test-cid-file")
	defer tempfile.Remove()

	_, err := newCIDFile(tempfile.Path())
	assert.ErrorContains(t, err, "container ID file found")
}

func TestCIDFileCloseWithNoWrite(t *testing.T) {
	tempdir := fs.NewDir(t, "test-cid-file")
	defer tempdir.Remove()

	path := tempdir.Join("cidfile")
	file, err := newCIDFile(path)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(file.path, path))

	assert.NilError(t, file.Close())
	_, err = os.Stat(path)
	assert.Check(t, os.IsNotExist(err))
}

func TestCIDFileCloseWithWrite(t *testing.T) {
	tempdir := fs.NewDir(t, "test-cid-file")
	defer tempdir.Remove()

	path := tempdir.Join("cidfile")
	file, err := newCIDFile(path)
	assert.NilError(t, err)

	content := "id"
	assert.NilError(t, file.Write(content))

	actual, err := os.ReadFile(path)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(content, string(actual)))

	assert.NilError(t, file.Close())
	_, err = os.Stat(path)
	assert.NilError(t, err)
}

func TestCreateContainerImagePullPolicy(t *testing.T) {
	const (
		imageName   = "does-not-exist-locally"
		containerID = "abcdef"
	)
	config := &containerConfig{
		Config: &container.Config{
			Image: imageName,
		},
		HostConfig: &container.HostConfig{},
	}

	cases := []struct {
		PullPolicy      string
		ExpectedPulls   int
		ExpectedID      string
		ExpectedErrMsg  string
		ResponseCounter int
	}{
		{
			PullPolicy:    PullImageMissing,
			ExpectedPulls: 1,
			ExpectedID:    containerID,
		}, {
			PullPolicy:      PullImageAlways,
			ExpectedPulls:   1,
			ExpectedID:      containerID,
			ResponseCounter: 1, // This lets us return a container on the first pull
		}, {
			PullPolicy:     PullImageNever,
			ExpectedPulls:  0,
			ExpectedErrMsg: "error fake not found",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.PullPolicy, func(t *testing.T) {
			pullCounter := 0

			client := &fakeClient{
				createContainerFunc: func(
					config *container.Config,
					hostConfig *container.HostConfig,
					networkingConfig *network.NetworkingConfig,
					platform *specs.Platform,
					containerName string,
				) (container.CreateResponse, error) {
					defer func() { tc.ResponseCounter++ }()
					switch tc.ResponseCounter {
					case 0:
						return container.CreateResponse{}, fakeNotFound{}
					default:
						return container.CreateResponse{ID: containerID}, nil
					}
				},
				imageCreateFunc: func(ctx context.Context, parentReference string, options image.CreateOptions) (io.ReadCloser, error) {
					defer func() { pullCounter++ }()
					return io.NopCloser(strings.NewReader("")), nil
				},
				infoFunc: func() (system.Info, error) {
					return system.Info{IndexServerAddress: "https://indexserver.example.com"}, nil
				},
			}
			fakeCLI := test.NewFakeCli(client)
			id, err := createContainer(context.Background(), fakeCLI, config, &createOptions{
				name:      "name",
				platform:  runtime.GOOS,
				untrusted: true,
				pull:      tc.PullPolicy,
			})

			if tc.ExpectedErrMsg != "" {
				assert.Check(t, is.ErrorContains(err, tc.ExpectedErrMsg))
			} else {
				assert.Check(t, err)
				assert.Check(t, is.Equal(tc.ExpectedID, id))
			}

			assert.Check(t, is.Equal(tc.ExpectedPulls, pullCounter))
		})
	}
}

func TestCreateContainerImagePullPolicyInvalid(t *testing.T) {
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
			err := runCreate(
				context.TODO(),
				dockerCli,
				&pflag.FlagSet{},
				&createOptions{pull: tc.PullPolicy},
				&containerOptions{},
			)

			statusErr := cli.StatusError{}
			assert.Check(t, errors.As(err, &statusErr))
			assert.Equal(t, statusErr.StatusCode, 125)
			assert.Check(t, is.Contains(dockerCli.ErrBuffer().String(), tc.ExpectedErrMsg))
		})
	}
}

func TestNewCreateCommandWithContentTrustErrors(t *testing.T) {
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
		cmd := NewCreateCommand(fakeCLI)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		assert.ErrorContains(t, err, tc.expectedError)
	}
}

func TestNewCreateCommandWithWarnings(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		warning bool
	}{
		{
			name: "container-create-without-oom-kill-disable",
			args: []string{"image:tag"},
		},
		{
			name: "container-create-oom-kill-disable-false",
			args: []string{"--oom-kill-disable=false", "image:tag"},
		},
		{
			name:    "container-create-oom-kill-without-memory-limit",
			args:    []string{"--oom-kill-disable", "image:tag"},
			warning: true,
		},
		{
			name:    "container-create-oom-kill-true-without-memory-limit",
			args:    []string{"--oom-kill-disable=true", "image:tag"},
			warning: true,
		},
		{
			name: "container-create-oom-kill-true-with-memory-limit",
			args: []string{"--oom-kill-disable=true", "--memory=100M", "image:tag"},
		},
		{
			name:    "container-create-localhost-dns",
			args:    []string{"--dns=127.0.0.11", "image:tag"},
			warning: true,
		},
		{
			name:    "container-create-localhost-dns-ipv6",
			args:    []string{"--dns=::1", "image:tag"},
			warning: true,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				createContainerFunc: func(config *container.Config,
					hostConfig *container.HostConfig,
					networkingConfig *network.NetworkingConfig,
					platform *specs.Platform,
					containerName string,
				) (container.CreateResponse, error) {
					return container.CreateResponse{}, nil
				},
			})
			cmd := NewCreateCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			if tc.warning {
				golden.Assert(t, cli.ErrBuffer().String(), tc.name+".golden")
			} else {
				assert.Equal(t, cli.ErrBuffer().String(), "")
			}
		})
	}
}

func TestCreateContainerWithProxyConfig(t *testing.T) {
	expected := []string{
		"HTTP_PROXY=httpProxy",
		"http_proxy=httpProxy",
		"HTTPS_PROXY=httpsProxy",
		"https_proxy=httpsProxy",
		"NO_PROXY=noProxy",
		"no_proxy=noProxy",
		"FTP_PROXY=ftpProxy",
		"ftp_proxy=ftpProxy",
		"ALL_PROXY=allProxy",
		"all_proxy=allProxy",
	}
	sort.Strings(expected)

	fakeCLI := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(config *container.Config,
			hostConfig *container.HostConfig,
			networkingConfig *network.NetworkingConfig,
			platform *specs.Platform,
			containerName string,
		) (container.CreateResponse, error) {
			sort.Strings(config.Env)
			assert.DeepEqual(t, config.Env, expected)
			return container.CreateResponse{}, nil
		},
	})
	fakeCLI.SetConfigFile(&configfile.ConfigFile{
		Proxies: map[string]configfile.ProxyConfig{
			"default": {
				HTTPProxy:  "httpProxy",
				HTTPSProxy: "httpsProxy",
				NoProxy:    "noProxy",
				FTPProxy:   "ftpProxy",
				AllProxy:   "allProxy",
			},
		},
	})
	cmd := NewCreateCommand(fakeCLI)
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"image:tag"})
	err := cmd.Execute()
	assert.NilError(t, err)
}

type fakeNotFound struct{}

func (f fakeNotFound) NotFound()     {}
func (f fakeNotFound) Error() string { return "error fake not found" }
