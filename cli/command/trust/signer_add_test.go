package trust

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/theupdateframework/notary"
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
			expectedError: "path to a public key must be provided using the `--key` flag",
		},
		{
			name:          "reserved-releases-signer-add",
			args:          []string{"releases", "my-image", "--key", "/path/to/key"},
			expectedError: "releases is a reserved keyword, please use a different signer name",
		},
		{
			name:          "disallowed-chars",
			args:          []string{"ali/ce", "my-image", "--key", "/path/to/key"},
			expectedError: "signer name \"ali/ce\" must start with lowercase alphanumeric characters and can include \"-\" or \"_\" after the first character",
		},
		{
			name:          "no-upper-case",
			args:          []string{"Alice", "my-image", "--key", "/path/to/key"},
			expectedError: "signer name \"Alice\" must start with lowercase alphanumeric characters and can include \"-\" or \"_\" after the first character",
		},
		{
			name:          "start-with-letter",
			args:          []string{"_alice", "my-image", "--key", "/path/to/key"},
			expectedError: "signer name \"_alice\" must start with lowercase alphanumeric characters and can include \"-\" or \"_\" after the first character",
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
	assert.EqualError(t, cmd.Execute(), fmt.Sprintf("could not parse public key from file: %s: no valid public key found", tmpfile.Name()))
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
	assert.EqualError(t, cmd.Execute(), "unable to read public key from file: open /path/to/key.pem: no such file or directory")
}

func TestSignerAddCommandInvalidRepoName(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	pubKeyDir, err := ioutil.TempDir("", "key-load-test-pubkey-")
	assert.NoError(t, err)
	defer os.RemoveAll(pubKeyDir)
	pubKeyFilepath := filepath.Join(pubKeyDir, "pubkey.pem")
	assert.NoError(t, ioutil.WriteFile(pubKeyFilepath, pubKeyFixture, notary.PrivNoExecPerms))

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(getUninitializedNotaryRepository)
	cmd := newSignerAddCommand(cli)
	imageName := "870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd"
	cmd.SetArgs([]string{"--key", pubKeyFilepath, "alice", imageName})

	cmd.SetOutput(ioutil.Discard)
	assert.EqualError(t, cmd.Execute(), "Failed to add signer to: 870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd")
	expectedErr := fmt.Sprintf("invalid repository name (%s), cannot specify 64-byte hexadecimal strings\n\n", imageName)

	assert.Equal(t, expectedErr, cli.ErrBuffer().String())
}

func TestIngestPublicKeys(t *testing.T) {
	// Call with a bad path
	_, err := ingestPublicKeys([]string{"foo", "bar"})
	assert.EqualError(t, err, "unable to read public key from file: open foo: no such file or directory")
	// Call with real file path
	tmpfile, err := ioutil.TempFile("", "pemfile")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = ingestPublicKeys([]string{tmpfile.Name()})
	assert.EqualError(t, err, fmt.Sprintf("could not parse public key from file: %s: no valid public key found", tmpfile.Name()))
}
