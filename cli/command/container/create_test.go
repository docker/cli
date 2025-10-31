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
	"github.com/google/go-cmp/cmp"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
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
		t.Run(tc.PullPolicy, func(t *testing.T) {
			pullCounter := 0

			apiClient := &fakeClient{
				createContainerFunc: func(options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
					defer func() { tc.ResponseCounter++ }()
					switch tc.ResponseCounter {
					case 0:
						return client.ContainerCreateResult{}, fakeNotFound{}
					default:
						return client.ContainerCreateResult{ID: containerID}, nil
					}
				},
				imagePullFunc: func(ctx context.Context, parentReference string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
					defer func() { pullCounter++ }()
					return fakeStreamResult{ReadCloser: io.NopCloser(strings.NewReader(""))}, nil
				},
				infoFunc: func() (client.SystemInfoResult, error) {
					return client.SystemInfoResult{
						Info: system.Info{IndexServerAddress: "https://indexserver.example.com"},
					}, nil
				},
			}
			fakeCLI := test.NewFakeCli(apiClient)
			id, err := createContainer(context.Background(), fakeCLI, config, &createOptions{
				name:     "name",
				platform: runtime.GOOS,
				pull:     tc.PullPolicy,
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
			assert.Check(t, is.Equal(statusErr.StatusCode, 125))
			assert.Check(t, is.ErrorContains(err, tc.ExpectedErrMsg))
		})
	}
}

func TestCreateContainerValidateFlags(t *testing.T) {
	for _, tc := range []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "with invalid --attach value",
			args:        []string{"--attach", "STDINFO", "myimage"},
			expectedErr: `invalid argument "STDINFO" for "-a, --attach" flag: valid streams are STDIN, STDOUT and STDERR`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newCreateCommand(test.NewFakeCli(&fakeClient{}))
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

func TestNewCreateCommandWithWarnings(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		warnings []string
		warning  bool
	}{
		{
			name: "container-create-no-warnings",
			args: []string{"image:tag"},
		},
		{
			name:     "container-create-daemon-single-warning",
			args:     []string{"image:tag"},
			warnings: []string{"warning from daemon"},
		},
		{
			name:     "container-create-daemon-multiple-warnings",
			args:     []string{"image:tag"},
			warnings: []string{"warning from daemon", "another warning from daemon"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeCLI := test.NewFakeCli(&fakeClient{
				createContainerFunc: func(options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
					return client.ContainerCreateResult{Warnings: tc.warnings}, nil
				},
			})
			cmd := newCreateCommand(fakeCLI)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			if tc.warning || len(tc.warnings) > 0 {
				golden.Assert(t, fakeCLI.ErrBuffer().String(), tc.name+".golden")
			} else {
				assert.Equal(t, fakeCLI.ErrBuffer().String(), "")
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
		createContainerFunc: func(options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
			sort.Strings(options.Config.Env)
			assert.DeepEqual(t, options.Config.Env, expected)
			return client.ContainerCreateResult{}, nil
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
	cmd := newCreateCommand(fakeCLI)
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"image:tag"})
	err := cmd.Execute()
	assert.NilError(t, err)
}

type fakeNotFound struct{}

func (fakeNotFound) NotFound()     {}
func (fakeNotFound) Error() string { return "error fake not found" }
