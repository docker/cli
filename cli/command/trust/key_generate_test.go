package trust

import (
	"encoding/pem"
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
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustmanager"
	tufutils "github.com/theupdateframework/notary/tuf/utils"
)

func TestTrustKeyGenerateErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "not-enough-args",
			expectedError: "requires exactly 1 argument",
		},
		{
			name:          "too-many-args",
			args:          []string{"key-1", "key-2"},
			expectedError: "requires exactly 1 argument",
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
	privKeyFileStore, err := trustmanager.NewKeyFileStore(privKeyStorageDir, cannedPasswordRetriever)
	assert.NoError(t, err)

	pubKeyPEM, err := generateKeyAndOutputPubPEM(keyName, privKeyFileStore)
	assert.NoError(t, err)

	assert.Equal(t, keyName, pubKeyPEM.Headers["role"])
	// the default GUN is empty
	assert.Equal(t, "", pubKeyPEM.Headers["gun"])
	// assert public key header
	assert.Equal(t, "PUBLIC KEY", pubKeyPEM.Type)

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
	fromBytes, _ := ioutil.ReadAll(privFrom)
	privKeyPEM, _ := pem.Decode(fromBytes)
	assert.Equal(t, keyName, privKeyPEM.Headers["role"])
	// the default GUN is empty
	assert.Equal(t, "", privKeyPEM.Headers["gun"])
	// assert encrypted header
	assert.Equal(t, "ENCRYPTED PRIVATE KEY", privKeyPEM.Type)
	// check that the passphrase matches
	_, err = tufutils.ParsePKCS8ToTufKey(privKeyPEM.Bytes, []byte(passwd))
	assert.NoError(t, err)

	// check that the public key exists at the correct path if we use the helper:
	returnedPath, err := writePubKeyPEMToDir(pubKeyPEM, keyName, pubKeyCWD)
	assert.NoError(t, err)
	expectedPubKeyPath := filepath.Join(pubKeyCWD, keyName+".pub")
	assert.Equal(t, returnedPath, expectedPubKeyPath)
	_, err = os.Stat(expectedPubKeyPath)
	assert.NoError(t, err)
	// check that the public key is the only file output in CWD
	cwdKeyFiles, err := ioutil.ReadDir(pubKeyCWD)
	assert.NoError(t, err)
	assert.Len(t, cwdKeyFiles, 1)
}

func TestValidateKeyArgs(t *testing.T) {
	pubKeyCWD, err := ioutil.TempDir("", "pub-keys-")
	assert.NoError(t, err)
	defer os.RemoveAll(pubKeyCWD)

	err = validateKeyArgs("a", pubKeyCWD)
	assert.NoError(t, err)

	err = validateKeyArgs("a/b", pubKeyCWD)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "key name \"a/b\" must start with lowercase alphanumeric characters and can include \"-\" or \"_\" after the first character")

	err = validateKeyArgs("-", pubKeyCWD)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "key name \"-\" must start with lowercase alphanumeric characters and can include \"-\" or \"_\" after the first character")

	assert.NoError(t, ioutil.WriteFile(filepath.Join(pubKeyCWD, "a.pub"), []byte("abc"), notary.PrivExecPerms))
	err = validateKeyArgs("a", pubKeyCWD)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), fmt.Sprintf("public key file already exists: \"%s/a.pub\"", pubKeyCWD))

	err = validateKeyArgs("a", "/random/dir/")
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "public key path does not exist: \"/random/dir/\"")
}
