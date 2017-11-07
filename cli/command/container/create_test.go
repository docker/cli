package container

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIDFileNoOPWithNoFilename(t *testing.T) {
	file, err := newCIDFile("")
	require.NoError(t, err)
	assert.Equal(t, &cidFile{}, file)

	assert.NoError(t, file.Write("id"))
	assert.NoError(t, file.Close())
}

func TestNewCIDFileWhenFileAlreadyExists(t *testing.T) {
	tempfile := fs.NewFile(t, "test-cid-file")
	defer tempfile.Remove()

	_, err := newCIDFile(tempfile.Path())
	testutil.ErrorContains(t, err, "Container ID file found")
}

func TestCIDFileCloseWithNoWrite(t *testing.T) {
	tempdir := fs.NewDir(t, "test-cid-file")
	defer tempdir.Remove()

	path := tempdir.Join("cidfile")
	file, err := newCIDFile(path)
	require.NoError(t, err)
	assert.Equal(t, file.path, path)

	assert.NoError(t, file.Close())
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestCIDFileCloseWithWrite(t *testing.T) {
	tempdir := fs.NewDir(t, "test-cid-file")
	defer tempdir.Remove()

	path := tempdir.Join("cidfile")
	file, err := newCIDFile(path)
	require.NoError(t, err)

	content := "id"
	assert.NoError(t, file.Write(content))

	actual, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(actual))

	assert.NoError(t, file.Close())
	_, err = os.Stat(path)
	require.NoError(t, err)
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
	body, err := createContainer(context.Background(), cli, config, "name", runtime.GOOS)
	require.NoError(t, err)
	expected := container.ContainerCreateCreatedBody{ID: containerID}
	assert.Equal(t, expected, *body)
	stderr := cli.ErrBuffer().String()
	assert.Contains(t, stderr, "Unable to find image 'does-not-exist-locally:latest' locally")
}

type fakeNotFound struct{}

func (f fakeNotFound) NotFound() bool { return true }
func (f fakeNotFound) Error() string  { return "error fake not found" }
