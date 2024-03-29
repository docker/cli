package trust

import (
	"testing"

	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestGetTag(t *testing.T) {
	ref, err := reference.ParseNormalizedNamed("ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2")
	assert.NilError(t, err)
	tag := getTag(ref)
	assert.Check(t, is.Equal("", tag))

	ref, err = reference.ParseNormalizedNamed("alpine:latest")
	assert.NilError(t, err)
	tag = getTag(ref)
	assert.Check(t, is.Equal(tag, "latest"))

	ref, err = reference.ParseNormalizedNamed("alpine")
	assert.NilError(t, err)
	tag = getTag(ref)
	assert.Check(t, is.Equal(tag, ""))
}

func TestGetDigest(t *testing.T) {
	ref, err := reference.ParseNormalizedNamed("ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2")
	assert.NilError(t, err)
	d := getDigest(ref)
	assert.Check(t, is.Equal(digest.Digest("sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2"), d))

	ref, err = reference.ParseNormalizedNamed("alpine:latest")
	assert.NilError(t, err)
	d = getDigest(ref)
	assert.Check(t, is.Equal(digest.Digest(""), d))

	ref, err = reference.ParseNormalizedNamed("alpine")
	assert.NilError(t, err)
	d = getDigest(ref)
	assert.Check(t, is.Equal(digest.Digest(""), d))
}

func TestGetSignableRolesError(t *testing.T) {
	notaryRepo, err := client.NewFileCachedRepository(t.TempDir(), "gun", "https://localhost", nil, passphrase.ConstantRetriever("password"), trustpinning.TrustPinConfig{})
	assert.NilError(t, err)
	target := client.Target{}
	_, err = GetSignableRoles(notaryRepo, &target)
	assert.Error(t, err, "client is offline")
}
