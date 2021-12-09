package container

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/google/go-cmp/cmp"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
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
	assert.ErrorContains(t, err, "Container ID file found")
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

	actual, err := ioutil.ReadFile(path)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(content, string(actual)))

	assert.NilError(t, file.Close())
	_, err = os.Stat(path)
	assert.NilError(t, err)
}

func TestCreateContainerImagePullPolicy(t *testing.T) {
	imageName := "does-not-exist-locally"
	containerID := "abcdef"
	config := &containerConfig{
		Config: &container.Config{
			Image: imageName,
		},
		HostConfig: &container.HostConfig{},
	}

	cases := []struct {
		PullPolicy      string
		ExpectedPulls   int
		ExpectedBody    container.ContainerCreateCreatedBody
		ExpectedErrMsg  string
		ResponseCounter int
	}{
		{
			PullPolicy:    PullImageMissing,
			ExpectedPulls: 1,
			ExpectedBody:  container.ContainerCreateCreatedBody{ID: containerID},
		}, {
			PullPolicy:      PullImageAlways,
			ExpectedPulls:   1,
			ExpectedBody:    container.ContainerCreateCreatedBody{ID: containerID},
			ResponseCounter: 1, // This lets us return a container on the first pull
		}, {
			PullPolicy:     PullImageNever,
			ExpectedPulls:  0,
			ExpectedErrMsg: "error fake not found",
		},
	}
	for _, c := range cases {
		c := c
		pullCounter := 0

		client := &fakeClient{
			createContainerFunc: func(
				config *container.Config,
				hostConfig *container.HostConfig,
				networkingConfig *network.NetworkingConfig,
				platform *specs.Platform,
				containerName string,
			) (container.ContainerCreateCreatedBody, error) {
				defer func() { c.ResponseCounter++ }()
				switch c.ResponseCounter {
				case 0:
					return container.ContainerCreateCreatedBody{}, fakeNotFound{}
				default:
					return container.ContainerCreateCreatedBody{ID: containerID}, nil
				}
			},
			imageCreateFunc: func(parentReference string, options types.ImageCreateOptions) (io.ReadCloser, error) {
				defer func() { pullCounter++ }()
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
			infoFunc: func() (types.Info, error) {
				return types.Info{IndexServerAddress: "https://indexserver.example.com"}, nil
			},
		}
		cli := test.NewFakeCli(client)
		body, err := createContainer(context.Background(), cli, config, &createOptions{
			name:      "name",
			platform:  runtime.GOOS,
			untrusted: true,
			pull:      c.PullPolicy,
		})

		if c.ExpectedErrMsg != "" {
			assert.ErrorContains(t, err, c.ExpectedErrMsg)
		} else {
			assert.NilError(t, err)
			assert.Check(t, is.DeepEqual(c.ExpectedBody, *body))
		}

		assert.Check(t, is.Equal(c.ExpectedPulls, pullCounter))
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
		cli := test.NewFakeCli(&fakeClient{
			createContainerFunc: func(config *container.Config,
				hostConfig *container.HostConfig,
				networkingConfig *network.NetworkingConfig,
				platform *specs.Platform,
				containerName string,
			) (container.ContainerCreateCreatedBody, error) {
				return container.ContainerCreateCreatedBody{}, fmt.Errorf("shouldn't try to pull image")
			},
		}, test.EnableContentTrust)
		cli.SetNotaryClient(tc.notaryFunc)
		cmd := NewCreateCommand(cli)
		cmd.SetOut(ioutil.Discard)
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
				) (container.ContainerCreateCreatedBody, error) {
					return container.ContainerCreateCreatedBody{}, nil
				},
			})
			cmd := NewCreateCommand(cli)
			cmd.SetOut(ioutil.Discard)
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

	cli := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(config *container.Config,
			hostConfig *container.HostConfig,
			networkingConfig *network.NetworkingConfig,
			platform *specs.Platform,
			containerName string,
		) (container.ContainerCreateCreatedBody, error) {
			sort.Strings(config.Env)
			assert.DeepEqual(t, config.Env, expected)
			return container.ContainerCreateCreatedBody{}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
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
	cmd := NewCreateCommand(cli)
	cmd.SetOut(ioutil.Discard)
	cmd.SetArgs([]string{"image:tag"})
	err := cmd.Execute()
	assert.NilError(t, err)
}

type fakeNotFound struct{}

func (f fakeNotFound) NotFound() bool { return true }
func (f fakeNotFound) Error() string  { return "error fake not found" }
