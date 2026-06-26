package trust

import (
	"testing"

	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
	"github.com/theupdateframework/notary/client"
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
	notaryRepo, err := client.NewFileCachedRepository(t.TempDir(), "gun", "https://localhost", nil, nil, trustpinning.TrustPinConfig{})
	assert.NilError(t, err)
	_, err = GetSignableRoles(notaryRepo, &client.Target{})
	const expected = "client is offline"
	assert.Error(t, err, expected)
}

func TestENVTrustServer(t *testing.T) {
	t.Setenv("DOCKER_CONTENT_TRUST_SERVER", "https://notary-test.example.com:5000")
	output, err := Server("testserver")
	const expected = "https://notary-test.example.com:5000"
	assert.NilError(t, err)
	assert.Equal(t, output, expected)
}

func TestHTTPENVTrustServer(t *testing.T) {
	t.Setenv("DOCKER_CONTENT_TRUST_SERVER", "http://notary-test.example.com:5000")
	_, err := Server("testserver")
	const expected = "valid https URL required for trust server"
	assert.ErrorContains(t, err, expected, "Expected error with invalid scheme")
}

func TestOfficialTrustServer(t *testing.T) {
	output, err := Server("docker.io")
	const expected = NotaryServer
	assert.NilError(t, err)
	assert.Equal(t, output, expected)

	output, err = Server("index.docker.io")
	assert.NilError(t, err)
	assert.Equal(t, output, expected)
}

func TestNonOfficialTrustServer(t *testing.T) {
	output, err := Server("testserver")
	const expected = "https://testserver"
	assert.NilError(t, err)
	assert.Equal(t, output, expected)
}
