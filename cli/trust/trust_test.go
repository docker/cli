package trust

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/distribution/reference"
	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
)

func TestGetTag(t *testing.T) {
	ref, err := reference.ParseNormalizedNamed("ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2")
	assert.NoError(t, err)
	tag := getTag(ref)
	assert.Equal(t, "", tag)

	ref, err = reference.ParseNormalizedNamed("alpine:latest")
	assert.NoError(t, err)
	tag = getTag(ref)
	assert.Equal(t, tag, "latest")

	ref, err = reference.ParseNormalizedNamed("alpine")
	assert.NoError(t, err)
	tag = getTag(ref)
	assert.Equal(t, tag, "")
}

func TestGetDigest(t *testing.T) {
	ref, err := reference.ParseNormalizedNamed("ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2")
	assert.NoError(t, err)
	d := getDigest(ref)
	assert.Equal(t, digest.Digest("sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2"), d)

	ref, err = reference.ParseNormalizedNamed("alpine:latest")
	assert.NoError(t, err)
	d = getDigest(ref)
	assert.Equal(t, digest.Digest(""), d)

	ref, err = reference.ParseNormalizedNamed("alpine")
	assert.NoError(t, err)
	d = getDigest(ref)
	assert.Equal(t, digest.Digest(""), d)
}

func TestGetSignableRolesError(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever("password"), trustpinning.TrustPinConfig{})
	require.NoError(t, err)
	target := client.Target{}
	_, err = GetSignableRoles(notaryRepo, &target)
	assert.EqualError(t, err, "client is offline")
}
