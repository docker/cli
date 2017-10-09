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
	"github.com/docker/notary"
	"github.com/docker/notary/passphrase"
	"github.com/docker/notary/storage"
	tufutils "github.com/docker/notary/tuf/utils"
	"github.com/docker/notary/utils"
	"github.com/stretchr/testify/assert"
)

func TestTrustKeyLoadErrors(t *testing.T) {
	testCases := []struct {
		name           string
		args           []string
		expectedError  string
		expectedOutput string
	}{
		{
			name:           "not-enough-args",
			expectedError:  "exactly 1 argument",
			expectedOutput: "",
		},
		{
			name:           "too-many-args",
			args:           []string{"iamnotakey", "alsonotakey"},
			expectedError:  "exactly 1 argument",
			expectedOutput: "",
		},
		{
			name:           "not-a-key",
			args:           []string{"iamnotakey"},
			expectedError:  "error importing key from iamnotakey: stat iamnotakey: no such file or directory",
			expectedOutput: "\nLoading key from \"iamnotakey\"...\n",
		},
	}
	tmpDir, err := ioutil.TempDir("", "docker-key-load-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	config.SetDir(tmpDir)

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		cmd := newKeyLoadCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
		assert.Contains(t, cli.OutBuffer().String(), tc.expectedOutput)
	}
}

var privKeyFixture = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAs7yVMzCw8CBZPoN+QLdx3ZzbVaHnouHIKu+ynX60IZ3stpbb
6rowu78OWON252JcYJqe++2GmdIgbBhg+mZDwhX0ZibMVztJaZFsYL+Ch/2J9KqD
A5NtE1s/XdhYoX5hsv7W4ok9jLFXRYIMj+T4exJRlR4f4GP9p0fcqPWd9/enPnlJ
JFTmu0DXJTZUMVS1UrXUy5t/DPXdrwyl8pM7VCqO3bqK7jqE6mWawdTkEeiku1fJ
ydP0285uiYTbj1Q38VVhPwXzMuLbkaUgRJhCI4BcjfQIjtJLbWpS+VdhUEvtgMVx
XJMKxCVGG69qjXyj9TjI7pxanb/bWglhovJN9wIDAQABAoIBAQCSnMsLxbUfOxPx
RWuwOLN+NZxIvtfnastQEtSdWiRvo5Xa3zYmw5hLHa8DXRC57+cwug/jqr54LQpb
gotg1hiBck05In7ezTK2FXTVeoJskal91bUnLpP0DSOkVnz9xszFKNF6Wr7FTEfH
IC1FF16Fbcz0mW0hKg9X6+uYOzqPcKpQRwli5LAwhT18Alf9h4/3NCeKotiJyr2J
xvcEH1eY2m2c/jQZurBkys7qBC3+i8LJEOW8MBQt7mxajwfbU91wtP2YoqMcoYiS
zsPbYp7Ui2t4G9Yn+OJw+uj4RGP1Bo4nSyRxWDtg+8Zug/JYU6/s+8kVRpiGffd3
T1GvoxUhAoGBAOnPDWG/g1xlJf65Rh71CxMs638zhYbIloU2K4Rqr05DHe7GryTS
9hLVrwhHddK+KwfVbR8HFMPo1DC/NVbuKt8StTAadAu3HsC088gWd28nOiGAWuvH
Bo3x/DYQGYwGFfoo4rzCOgMj6DJjXmcWEXNv3NDMoXoYpkxa0g6zZDyHAoGBAMTL
t7EUneJT+Mm7wyL1I5bmaT/HFwqoUQB2ccBPVD8p1el62NgLdfhOa8iNlBVhMrlh
2aTjrMlSPcjr9sCgKrLcenSWw+2qFsf4+SmV01ntB9kWes2phXpnB0ynXIcbeG05
+BLxbqDTVV0Iqh4r/dGeplyV2WyL3mTpkT3hRq8RAoGAZ93degEUICWnHWO9LN97
Dge0joua0+ekRoVsC6VBP6k9UOfewqMdQfy/hxQH2Zk1kINVuKTyqp1yNj2bOoUP
co3jA/2cc9/jv4QjkE26vRxWDK/ytC90T/aiLno0fyns9XbYUzaNgvuemVPfijgZ
hIi7Nd7SFWWB6wWlr3YuH10CgYEAwh7JVa2mh8iZEjVaKTNyJbmmfDjgq6yYKkKr
ti0KRzv3O9Xn7ERx27tPaobtWaGFLYQt8g57NCMhuv23aw8Sz1fYmwTUw60Rx7P5
42FdF8lOAn/AJvpfJfxXIO+9v7ADPIr//3+TxqRwAdM4K4btWkaKh61wyTe26gfT
MxzyYmECgYAnlU5zsGyiZqwoXVktkhtZrE7Qu0SoztzFb8KpvFNmMTPF1kAAYmJY
GIhbizeGJ3h4cUdozKmt8ZWIt6uFDEYCqEA7XF4RH75dW25x86mpIPO7iRl9eisY
IsLeMYqTIwXAwGx6Ka9v5LOL1kzcHQ2iVj6+QX+yoptSft1dYa9jOA==
-----END RSA PRIVATE KEY-----`)

const privKeyID = "ee69e8e07a14756ad5ff0aca2336b37f86b0ac1710d1f3e94440081e080aecd7"

func TestLoadKeyFromPath(t *testing.T) {
	privKeyDir, err := ioutil.TempDir("", "key-load-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(privKeyDir)
	privKeyFilepath := filepath.Join(privKeyDir, "privkey.pem")
	assert.NoError(t, ioutil.WriteFile(privKeyFilepath, privKeyFixture, notary.PrivNoExecPerms))

	keyStorageDir, err := ioutil.TempDir("", "loaded-keys-")
	assert.NoError(t, err)
	defer os.RemoveAll(keyStorageDir)

	passwd := "password"
	cannedPasswordRetriever := passphrase.ConstantRetriever(passwd)
	keyFileStore, err := storage.NewPrivateKeyFileStorage(keyStorageDir, notary.KeyExtension)
	assert.NoError(t, err)
	privKeyImporters := []utils.Importer{keyFileStore}

	// import the key to our keyStorageDir
	assert.NoError(t, loadPrivKeyFromPath(privKeyImporters, privKeyFilepath, "signer-name", cannedPasswordRetriever))

	// check that the appropriate ~/<trust_dir>/private/<key_id>.key file exists
	expectedImportKeyPath := filepath.Join(keyStorageDir, notary.PrivDir, privKeyID+"."+notary.KeyExtension)
	_, err = os.Stat(expectedImportKeyPath)
	assert.NoError(t, err)

	// verify the key content
	from, _ := os.OpenFile(expectedImportKeyPath, os.O_RDONLY, notary.PrivExecPerms)
	defer from.Close()
	fromBytes, _ := ioutil.ReadAll(from)
	keyPEM, _ := pem.Decode(fromBytes)
	assert.Equal(t, "signer-name", keyPEM.Headers["role"])
	// the default GUN is empty
	assert.Equal(t, "", keyPEM.Headers["gun"])
	// assert encrypted header
	assert.Equal(t, "ENCRYPTED PRIVATE KEY", keyPEM.Type)

	decryptedKey, err := tufutils.ParsePKCS8ToTufKey(keyPEM.Bytes, []byte(passwd))
	assert.NoError(t, err)
	fixturePEM, _ := pem.Decode(privKeyFixture)
	assert.Equal(t, fixturePEM.Bytes, decryptedKey.Private())
}

func TestLoadKeyTooPermissive(t *testing.T) {
	privKeyDir, err := ioutil.TempDir("", "key-load-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(privKeyDir)
	privKeyFilepath := filepath.Join(privKeyDir, "privkey477.pem")
	assert.NoError(t, ioutil.WriteFile(privKeyFilepath, privKeyFixture, 0477))

	keyStorageDir, err := ioutil.TempDir("", "loaded-keys-")
	assert.NoError(t, err)
	defer os.RemoveAll(keyStorageDir)

	passwd := "password"
	cannedPasswordRetriever := passphrase.ConstantRetriever(passwd)
	keyFileStore, err := storage.NewPrivateKeyFileStorage(keyStorageDir, notary.KeyExtension)
	assert.NoError(t, err)
	privKeyImporters := []utils.Importer{keyFileStore}

	// import the key to our keyStorageDir
	err = loadPrivKeyFromPath(privKeyImporters, privKeyFilepath, "signer", cannedPasswordRetriever)
	assert.Error(t, err)
	assert.Contains(t, fmt.Sprintf("private key permission from %s should be set to 400 or 600", privKeyFilepath), err.Error())

	privKeyFilepath = filepath.Join(privKeyDir, "privkey667.pem")
	assert.NoError(t, ioutil.WriteFile(privKeyFilepath, privKeyFixture, 0677))

	err = loadPrivKeyFromPath(privKeyImporters, privKeyFilepath, "signer", cannedPasswordRetriever)
	assert.Error(t, err)
	assert.Contains(t, fmt.Sprintf("private key permission from %s should be set to 400 or 600", privKeyFilepath), err.Error())

	privKeyFilepath = filepath.Join(privKeyDir, "privkey777.pem")
	assert.NoError(t, ioutil.WriteFile(privKeyFilepath, privKeyFixture, 0777))

	err = loadPrivKeyFromPath(privKeyImporters, privKeyFilepath, "signer", cannedPasswordRetriever)
	assert.Error(t, err)
	assert.Contains(t, fmt.Sprintf("private key permission from %s should be set to 400 or 600", privKeyFilepath), err.Error())

	privKeyFilepath = filepath.Join(privKeyDir, "privkey400.pem")
	assert.NoError(t, ioutil.WriteFile(privKeyFilepath, privKeyFixture, 0400))

	err = loadPrivKeyFromPath(privKeyImporters, privKeyFilepath, "signer", cannedPasswordRetriever)
	assert.NoError(t, err)

	privKeyFilepath = filepath.Join(privKeyDir, "privkey600.pem")
	assert.NoError(t, ioutil.WriteFile(privKeyFilepath, privKeyFixture, 0600))

	err = loadPrivKeyFromPath(privKeyImporters, privKeyFilepath, "signer", cannedPasswordRetriever)
	assert.NoError(t, err)
}

var pubKeyFixture = []byte(`-----BEGIN PUBLIC KEY-----
	MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEUIH9AYtrcDFzZrFJBdJZkn21d+4c
	H3nzy2O6Q/ct4BjOBKa+WCdRtPo78bA+C/7t81ADQO8Jqaj59W50rwoqDQ==
	-----END PUBLIC KEY-----`)

func TestLoadPubKeyFailure(t *testing.T) {
	pubKeyDir, err := ioutil.TempDir("", "key-load-test-pubkey-")
	assert.NoError(t, err)
	defer os.RemoveAll(pubKeyDir)
	pubKeyFilepath := filepath.Join(pubKeyDir, "pubkey.pem")
	assert.NoError(t, ioutil.WriteFile(pubKeyFilepath, pubKeyFixture, notary.PrivNoExecPerms))
	keyStorageDir, err := ioutil.TempDir("", "loaded-keys-")
	assert.NoError(t, err)
	defer os.RemoveAll(keyStorageDir)

	passwd := "password"
	cannedPasswordRetriever := passphrase.ConstantRetriever(passwd)
	keyFileStore, err := storage.NewPrivateKeyFileStorage(keyStorageDir, notary.KeyExtension)
	assert.NoError(t, err)
	privKeyImporters := []utils.Importer{keyFileStore}

	// import the key to our keyStorageDir - it should fail
	err = loadPrivKeyFromPath(privKeyImporters, pubKeyFilepath, "signer", cannedPasswordRetriever)
	assert.Error(t, err)
	assert.Contains(t, fmt.Sprintf("provided file %s is not a supported private key - to add a signer's public key use docker trust signer add", pubKeyFilepath), err.Error())
}
