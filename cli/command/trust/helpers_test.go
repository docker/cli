package trust

import (
	"testing"

	"github.com/docker/distribution/reference"
	"github.com/stretchr/testify/assert"
)

func TestGetTag(t *testing.T) {
	ref, err := reference.ParseNormalizedNamed("ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2")
	assert.NoError(t, err)
	tag, err := getTag(ref)
	assert.Error(t, err)
	assert.EqualError(t, err, "cannot use a digest reference for IMAGE:TAG")

	ref, err = reference.ParseNormalizedNamed("alpine:latest")
	assert.NoError(t, err)
	tag, err = getTag(ref)
	assert.NoError(t, err)
	assert.Equal(t, tag, "latest")

	ref, err = reference.ParseNormalizedNamed("alpine")
	assert.NoError(t, err)
	tag, err = getTag(ref)
	assert.NoError(t, err)
	assert.Equal(t, tag, "")
}
