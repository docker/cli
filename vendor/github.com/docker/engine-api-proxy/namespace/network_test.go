package pipeline

import (
	gojson "encoding/json"
	"net/http"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/engine-api-proxy/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkConnectRequest(t *testing.T) {
	path := &scopePath{
		inspector: newMockInspector(&labeled{ID: "aaaaaa"}, nil),
	}
	route := &networkConnectRoute{
		lookup:             defaultLookup,
		containerScopePath: path,
	}
	network := types.NetworkDisconnect{
		Container: "gibs",
		Force:     true,
	}
	expected := types.NetworkDisconnect{
		Container: "myproject_gibs",
		Force:     true,
	}

	_, body, err := json.Encode(network)
	require.NoError(t, err)
	input, err := http.NewRequest("POST", "/network/foo", body)

	req, err := route.request(nil, input)
	require.NoError(t, err)

	actual := types.NetworkDisconnect{}
	err = gojson.NewDecoder(req.Body).Decode(&actual)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
