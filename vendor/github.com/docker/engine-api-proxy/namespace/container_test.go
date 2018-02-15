package pipeline

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/engine-api-proxy/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerListResponse(t *testing.T) {
	containers := []types.Container{{Names: []string{"/myproject_foo"}}}
	expected := []types.Container{{Names: []string{"/foo"}}}
	_, encoded, err := json.Encode(containers)
	require.NoError(t, err)

	size, reader, err := containerListResponse(defaultLookup, nil, encoded)

	require.NoError(t, err)
	assert.Equal(t, 181, size)
	actual, err := json.DecodeContainers(reader)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
