package trust

import (
	"testing"

	"github.com/docker/distribution/reference"
	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
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
