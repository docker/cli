package trust

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
)

func TestGetOrGenerateNotaryKeyAndInitRepo(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NilError(t, err)

	err = getOrGenerateRootKeyAndInitRepo(notaryRepo)
	assert.Error(t, err, "client is offline")
}
