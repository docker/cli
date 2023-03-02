package trust

import (
	"testing"

	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
	"gotest.tools/v3/assert"
)

func TestGetOrGenerateNotaryKeyAndInitRepo(t *testing.T) {
	notaryRepo, err := client.NewFileCachedRepository(t.TempDir(), "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NilError(t, err)

	err = getOrGenerateRootKeyAndInitRepo(notaryRepo)
	assert.Error(t, err, "client is offline")
}
