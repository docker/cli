package trust

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/internal/test"
	notaryfake "github.com/docker/cli/internal/test/notary"
	"github.com/theupdateframework/notary"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
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
			expectedError: "releases is a reserved keyword, use a different signer name",
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
	config.SetDir(t.TempDir())

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		cli.SetNotaryClient(notaryfake.GetOfflineNotaryRepository)
		cmd := newSignerAddCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSignerAddCommandNoTargetsKey(t *testing.T) {
	config.SetDir(t.TempDir())

	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "pemfile")
	assert.NilError(t, err)
	assert.Check(t, tmpFile.Close())

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetEmptyTargetsNotaryRepository)
	cmd := newSignerAddCommand(cli)
	cmd.SetArgs([]string{"--key", tmpFile.Name(), "alice", "alpine", "linuxkit/alpine"})

	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.Error(t, cmd.Execute(), fmt.Sprintf("could not parse public key from file: %s: no valid public key found", tmpFile.Name()))
}

func TestSignerAddCommandBadKeyPath(t *testing.T) {
	config.SetDir(t.TempDir())

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetEmptyTargetsNotaryRepository)
	cmd := newSignerAddCommand(cli)
	cmd.SetArgs([]string{"--key", "/path/to/key.pem", "alice", "alpine"})

	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	expectedError := "unable to read public key from file: open /path/to/key.pem: no such file or directory"
	if runtime.GOOS == "windows" {
		expectedError = "unable to read public key from file: open /path/to/key.pem: The system cannot find the path specified."
	}
	assert.Error(t, cmd.Execute(), expectedError)
}

func TestSignerAddCommandInvalidRepoName(t *testing.T) {
	config.SetDir(t.TempDir())

	pubKeyDir := t.TempDir()
	pubKeyFilepath := filepath.Join(pubKeyDir, "pubkey.pem")
	assert.NilError(t, os.WriteFile(pubKeyFilepath, pubKeyFixture, notary.PrivNoExecPerms))

	cli := test.NewFakeCli(&fakeClient{})
	cli.SetNotaryClient(notaryfake.GetUninitializedNotaryRepository)
	cmd := newSignerAddCommand(cli)
	imageName := "870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd"
	cmd.SetArgs([]string{"--key", pubKeyFilepath, "alice", imageName})

	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.Error(t, cmd.Execute(), "failed to add signer to: 870d292919d01a0af7e7f056271dc78792c05f55f49b9b9012b6d89725bd9abd")
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
	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "pemfile")
	assert.NilError(t, err)
	assert.Check(t, tmpFile.Close())
	_, err = ingestPublicKeys([]string{tmpFile.Name()})
	assert.Error(t, err, fmt.Sprintf("could not parse public key from file: %s: no valid public key found", tmpFile.Name()))
}
