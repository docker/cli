package container

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/pkg/errors"
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

func TestCreateContainerPullsImageIfMissing(t *testing.T) {
	imageName := "does-not-exist-locally"
	responseCounter := 0
	containerID := "abcdef"

	client := &fakeClient{
		createContainerFunc: func(
			config *container.Config,
			hostConfig *container.HostConfig,
			networkingConfig *network.NetworkingConfig,
			containerName string,
		) (container.ContainerCreateCreatedBody, error) {
			defer func() { responseCounter++ }()
			switch responseCounter {
			case 0:
				return container.ContainerCreateCreatedBody{}, fakeNotFound{}
			case 1:
				return container.ContainerCreateCreatedBody{ID: containerID}, nil
			default:
				return container.ContainerCreateCreatedBody{}, errors.New("unexpected")
			}
		},
		imageCreateFunc: func(parentReference string, options types.ImageCreateOptions) (io.ReadCloser, error) {
			return ioutil.NopCloser(strings.NewReader("")), nil
		},
		infoFunc: func() (types.Info, error) {
			return types.Info{IndexServerAddress: "http://indexserver"}, nil
		},
	}
	cli := test.NewFakeCli(client)
	config := &containerConfig{
		Config: &container.Config{
			Image: imageName,
		},
		HostConfig: &container.HostConfig{},
	}
	body, err := createContainer(context.Background(), cli, config, &createOptions{
		name:      "name",
		platform:  runtime.GOOS,
		untrusted: true,
	})
	assert.NilError(t, err)
	expected := container.ContainerCreateCreatedBody{ID: containerID}
	assert.Check(t, is.DeepEqual(expected, *body))
	stderr := cli.ErrBuffer().String()
	assert.Check(t, is.Contains(stderr, "Unable to find image 'does-not-exist-locally:latest' locally"))
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
		cli := test.NewFakeCli(&fakeClient{
			createContainerFunc: func(config *container.Config,
				hostConfig *container.HostConfig,
				networkingConfig *network.NetworkingConfig,
				containerName string,
			) (container.ContainerCreateCreatedBody, error) {
				return container.ContainerCreateCreatedBody{}, fmt.Errorf("shouldn't try to pull image")
			},
		}, test.EnableContentTrust)
		cli.SetNotaryClient(tc.notaryFunc)
		cmd := NewCreateCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		assert.ErrorContains(t, err, tc.expectedError)
	}
}

type fakeNotFound struct{}

func (f fakeNotFound) NotFound() bool { return true }
func (f fakeNotFound) Error() string  { return "error fake not found" }
