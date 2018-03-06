package trust

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
)

func TestGetOrGenerateNotaryKeyAndInitRepo(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.Check(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.Check(t, err)

	err = getOrGenerateRootKeyAndInitRepo(notaryRepo)
	assert.Check(t, is.Error(err, "client is offline"))
}
