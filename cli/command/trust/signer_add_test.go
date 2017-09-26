package trust

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/stretchr/testify/assert"
)

func TestTrustSignerAddErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "not-enough-args",
			expectedError: "requires at least 2 argument",
		},
		{
			name:          "no-key",
			args:          []string{"foo", "bar"},
			expectedError: "path to a valid public key must be provided using the `--key` flag",
		},
		{
			name:          "reserved-releases-signer-add",
			args:          []string{"releases", "my-image", "-k", "/path/to/key"},
			expectedError: "releases is a reserved keyword, please use a different signer name",
		},
		{
			name:          "disallowed-chars",
			args:          []string{"ali/ce", "my-image", "-k", "/path/to/key"},
			expectedError: "signer name \"ali/ce\" must not contain uppercase or special characters",
		},
		{
			name:          "no-upper-case",
			args:          []string{"Alice", "my-image", "-k", "/path/to/key"},
			expectedError: "signer name \"Alice\" must not contain uppercase or special characters",
		},
		{
			name:          "start-with-letter",
			args:          []string{"_alice", "my-image", "-k", "/path/to/key"},
			expectedError: "signer name \"_alice\" must not contain uppercase or special characters",
		},
	}
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		cli.SetNotaryClient(getOfflineNotaryRepository)
		cmd := newSignerAddCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSignerAddCommandNoTargetsKey(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	tmpfile, err := ioutil.TempFile("", "pemfile")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getEmptyTargetsNotaryRepository)
	cmd := newSignerAddCommand(cli)
	cmd.SetArgs([]string{"--key", tmpfile.Name(), "alice", "alpine", "linuxkit/alpine"})

	cmd.SetOutput(ioutil.Discard)
	assert.EqualError(t, cmd.Execute(), "Failed to add signer to: alpine, linuxkit/alpine")

	assert.Contains(t, cli.OutBuffer().String(), "Adding signer \"alice\" to alpine...")
	assert.Contains(t, cli.OutBuffer().String(), "no valid public key found")

	assert.Contains(t, cli.OutBuffer().String(), "Adding signer \"alice\" to linuxkit/alpine...")
	assert.Contains(t, cli.OutBuffer().String(), "no valid public key found")
}

func TestSignerAddCommandBadKeyPath(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getEmptyTargetsNotaryRepository)
	cmd := newSignerAddCommand(cli)
	cmd.SetArgs([]string{"--key", "/path/to/key.pem", "alice", "alpine"})

	cmd.SetOutput(ioutil.Discard)
	assert.EqualError(t, cmd.Execute(), "Failed to add signer to: alpine")

	expectedError := "\nAdding signer \"alice\" to alpine...\nfile for public key does not exist: /path/to/key.pem"
	assert.Contains(t, cli.OutBuffer().String(), expectedError)
}

func TestSignerAddCommandInvalidRepoName(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getUninitializedNotaryRepository)
	cmd := newSignerAddCommand(cli)
	imageName := "870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd"
	cmd.SetArgs([]string{"--key", "/path/to/key.pem", "alice", imageName})

	cmd.SetOutput(ioutil.Discard)
	assert.EqualError(t, cmd.Execute(), "Failed to add signer to: 870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd")
	expectedOutput := fmt.Sprintf("\nAdding signer \"alice\" to %s...\n"+
		"invalid repository name (%s), cannot specify 64-byte hexadecimal strings\n",
		imageName, imageName)

	assert.Equal(t, expectedOutput, cli.OutBuffer().String())
}

func TestIngestPublicKeys(t *testing.T) {
	// Call with a bad path
	_, err := ingestPublicKeys([]string{"foo", "bar"})
	assert.EqualError(t, err, "file for public key does not exist: foo")
	// Call with real file path
	tmpfile, err := ioutil.TempFile("", "pemfile")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = ingestPublicKeys([]string{tmpfile.Name()})
	assert.EqualError(t, err, "no valid public key found")
}
