package trust

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/notary/client"
	"github.com/docker/notary/passphrase"
	"github.com/docker/notary/trustpinning"

	"github.com/stretchr/testify/assert"
)

func TestGetOrGenerateNotaryKeyAndInitRepo(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever(passwd), trustpinning.TrustPinConfig{})
	assert.NoError(t, err)

	err = getOrGenerateRootKeyAndInitRepo(notaryRepo)
	assert.EqualError(t, err, "client is offline")
}
