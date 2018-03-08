package trust

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/internal/test"
	notaryfake "github.com/docker/cli/internal/test/notary"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
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
	assert.NilError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		cli.SetNotaryClient(notaryfake.GetOfflineNotaryRepository)
		cmd := newSignerAddCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSignerAddCommandNoTargetsKey(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	tmpfile, err := ioutil.TempFile("", "pemfile")
	assert.NilError(t, err)
	defer os.Remove(tmpfile.Name())

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetEmptyTargetsNotaryRepository)
	cmd := newSignerAddCommand(cli)
	cmd.SetArgs([]string{"--key", tmpfile.Name(), "alice", "alpine", "linuxkit/alpine"})

	cmd.SetOutput(ioutil.Discard)
	assert.Error(t, cmd.Execute(), fmt.Sprintf("could not parse public key from file: %s: no valid public key found", tmpfile.Name()))
}

func TestSignerAddCommandBadKeyPath(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetEmptyTargetsNotaryRepository)
	cmd := newSignerAddCommand(cli)
	cmd.SetArgs([]string{"--key", "/path/to/key.pem", "alice", "alpine"})

	cmd.SetOutput(ioutil.Discard)
	expectedError := "unable to read public key from file: open /path/to/key.pem: no such file or directory"
	if runtime.GOOS == "windows" {
		expectedError = "unable to read public key from file: open /path/to/key.pem: The system cannot find the path specified."
	}
	assert.Error(t, cmd.Execute(), expectedError)
}

func TestSignerAddCommandInvalidRepoName(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "docker-sign-test-")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	pubKeyDir, err := ioutil.TempDir("", "key-load-test-pubkey-")
	assert.NilError(t, err)
	defer os.RemoveAll(pubKeyDir)
	pubKeyFilepath := filepath.Join(pubKeyDir, "pubkey.pem")
	assert.NilError(t, ioutil.WriteFile(pubKeyFilepath, pubKeyFixture, notary.PrivNoExecPerms))

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetUninitializedNotaryRepository)
	cmd := newSignerAddCommand(cli)
	imageName := "870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd"
	cmd.SetArgs([]string{"--key", pubKeyFilepath, "alice", imageName})

	cmd.SetOutput(ioutil.Discard)
	assert.Error(t, cmd.Execute(), "Failed to add signer to: 870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd")
	expectedErr := fmt.Sprintf("invalid repository name (%s), cannot specify 64-byte hexadecimal strings\n\n", imageName)

	assert.Check(t, is.Equal(expectedErr, cli.ErrBuffer().String()))
}

func TestIngestPublicKeys(t *testing.T) {
	// Call with a bad path
	_, err := ingestPublicKeys([]string{"foo", "bar"})
	expectedError := "unable to read public key from file: open foo: no such file or directory"
	if runtime.GOOS == "windows" {
		expectedError = "unable to read public key from file: open foo: The system cannot find the file specified."
	}
	assert.Error(t, err, expectedError)
	// Call with real file path
	tmpfile, err := ioutil.TempFile("", "pemfile")
	assert.NilError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = ingestPublicKeys([]string{tmpfile.Name()})
	assert.Error(t, err, fmt.Sprintf("could not parse public key from file: %s: no valid public key found", tmpfile.Name()))
}
