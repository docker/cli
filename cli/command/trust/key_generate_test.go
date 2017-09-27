package trust

import (
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/notary"
	"github.com/docker/notary/passphrase"
	tufutils "github.com/docker/notary/tuf/utils"
	"github.com/stretchr/testify/assert"
)

func TestTrustKeyGenerateErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "not-enough-args",
			expectedError: "requires at least 1 argument",
		},
	}
	tmpDir, err := ioutil.TempDir("", "docker-key-generate-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		cmd := newKeyGenerateCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestGenerateKeySuccess(t *testing.T) {
	pubKeyCWD, err := ioutil.TempDir("", "pub-keys-")
	assert.NoError(t, err)
	defer os.RemoveAll(pubKeyCWD)

	privKeyStorageDir, err := ioutil.TempDir("", "priv-keys-")
	assert.NoError(t, err)
	defer os.RemoveAll(privKeyStorageDir)

	passwd := "password"
	cannedPasswordRetriever := passphrase.ConstantRetriever(passwd)
	// generate a single key
	keyName := "alice"
	assert.NoError(t, generateKey(keyName, pubKeyCWD, privKeyStorageDir, cannedPasswordRetriever))

	// check that the public key exists:
	expectedPubKeyPath := filepath.Join(pubKeyCWD, keyName+".pub")
	_, err = os.Stat(expectedPubKeyPath)
	assert.NoError(t, err)
	// check that the public key is the only file output in CWD
	cwdKeyFiles, err := ioutil.ReadDir(pubKeyCWD)
	assert.NoError(t, err)
	assert.Len(t, cwdKeyFiles, 1)

	// verify the key header is set with the specified name
	from, _ := os.OpenFile(expectedPubKeyPath, os.O_RDONLY, notary.PrivExecPerms)
	defer from.Close()
	fromBytes, _ := ioutil.ReadAll(from)
	keyPEM, _ := pem.Decode(fromBytes)
	assert.Equal(t, keyName, keyPEM.Headers["role"])
	// the default GUN is empty
	assert.Equal(t, "", keyPEM.Headers["gun"])
	// assert public key header
	assert.Equal(t, "PUBLIC KEY", keyPEM.Type)

	// check that an appropriate ~/<trust_dir>/private/<key_id>.key file exists
	expectedPrivKeyDir := filepath.Join(privKeyStorageDir, notary.PrivDir)
	_, err = os.Stat(expectedPrivKeyDir)
	assert.NoError(t, err)

	keyFiles, err := ioutil.ReadDir(expectedPrivKeyDir)
	assert.NoError(t, err)
	assert.Len(t, keyFiles, 1)
	privKeyFilePath := filepath.Join(expectedPrivKeyDir, keyFiles[0].Name())

	// verify the key content
	privFrom, _ := os.OpenFile(privKeyFilePath, os.O_RDONLY, notary.PrivExecPerms)
	defer privFrom.Close()
	fromBytes, _ = ioutil.ReadAll(privFrom)
	keyPEM, _ = pem.Decode(fromBytes)
	assert.Equal(t, keyName, keyPEM.Headers["role"])
	// the default GUN is empty
	assert.Equal(t, "", keyPEM.Headers["gun"])
	// assert encrypted header
	assert.Equal(t, "ENCRYPTED PRIVATE KEY", keyPEM.Type)
	// check that the passphrase matches
	_, err = tufutils.ParsePKCS8ToTufKey(keyPEM.Bytes, []byte(passwd))
	assert.NoError(t, err)
}

func TestValidateKeyArgs(t *testing.T) {
	pubKeyCWD, err := ioutil.TempDir("", "pub-keys-")
	assert.NoError(t, err)
	defer os.RemoveAll(pubKeyCWD)

	err = validateKeyArgs([]string{"a", "b", "C_123", "key-name"}, pubKeyCWD)
	assert.NoError(t, err)

	err = validateKeyArgs([]string{"a", "a"}, pubKeyCWD)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "key names must be unique, found duplicate key name: \"a\"")

	err = validateKeyArgs([]string{"a/b"}, pubKeyCWD)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "key name \"a/b\" must not contain special characters")

	err = validateKeyArgs([]string{"-"}, pubKeyCWD)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "key name \"-\" must not contain special characters")

	assert.NoError(t, ioutil.WriteFile(filepath.Join(pubKeyCWD, "a.pub"), []byte("abc"), notary.PrivExecPerms))
	err = validateKeyArgs([]string{"a"}, pubKeyCWD)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "public key file already exists: \"a.pub\"")
}

func TestGenerateMultipleKeysOutput(t *testing.T) {
	pubKeyCWD, err := ioutil.TempDir("", "pub-keys-")
	assert.NoError(t, err)
	defer os.RemoveAll(pubKeyCWD)

	passwd := "password"
	cannedPasswordRetriever := func() notary.PassRetriever { return passphrase.ConstantRetriever(passwd) }

	cli := test.NewFakeCli(&fakeClient{})
	assert.NoError(t, generateKeys(cli, []string{"alice", "bob", "charlie"}, pubKeyCWD, cannedPasswordRetriever))

	// Check the stdout prints:
	assert.Contains(t, cli.OutBuffer().String(), "\nGenerating key for alice...\n")
	assert.Contains(t, cli.OutBuffer().String(), "Successfully generated and loaded private key. Corresponding public key available: alice.pub\n")
	assert.Contains(t, cli.OutBuffer().String(), "\nGenerating key for bob...\n")
	assert.Contains(t, cli.OutBuffer().String(), "Successfully generated and loaded private key. Corresponding public key available: bob.pub\n")
	assert.Contains(t, cli.OutBuffer().String(), "\nGenerating key for charlie...\n")
	assert.Contains(t, cli.OutBuffer().String(), "Successfully generated and loaded private key. Corresponding public key available: charlie.pub\n")

	// Check that we have three key files:
	cwdKeyFiles, err := ioutil.ReadDir(pubKeyCWD)
	assert.NoError(t, err)
	assert.Len(t, cwdKeyFiles, 3)
}
