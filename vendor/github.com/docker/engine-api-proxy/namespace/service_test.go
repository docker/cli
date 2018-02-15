package pipeline

import (
	gojson "encoding/json"
	"net/http"
	"testing"

	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/engine-api-proxy/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// utcZero is necessary because json encode/decoded adds the UTC location
// to all time fields. This causes reflect.DeepEqual() to return false because
// the time fields are utcZero, instead of time.Time{}
var utcZero = time.Time{}.UTC()
var metaZero = swarm.Meta{CreatedAt: utcZero, UpdatedAt: utcZero}

func TestSpecedListResponse(t *testing.T) {
	secrets := []swarm.Secret{
		{
			ID: "abcd",
			Spec: swarm.SecretSpec{
				Annotations: swarm.Annotations{Name: "myproject_foo"},
				Data:        []byte("password"),
			},
		},
	}
	expected := []swarm.Secret{
		{
			ID: "abcd",
			Spec: swarm.SecretSpec{
				Annotations: swarm.Annotations{Name: "foo"},
				Data:        []byte("password"),
			},
			Meta: metaZero,
		},
	}
	_, encoded, err := json.Encode(secrets)
	require.NoError(t, err)

	_, reader, err := specedListResponse(defaultLookup, nil, encoded)
	require.NoError(t, err)

	actual := []swarm.Secret{}
	err = gojson.NewDecoder(reader).Decode(&actual)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestSpecedInspectResponse(t *testing.T) {
	service := swarm.Service{
		ID: "abcd",
		Spec: swarm.ServiceSpec{
			Annotations:  swarm.Annotations{Name: "myproject_foo"},
			UpdateConfig: &swarm.UpdateConfig{Parallelism: 2},
		},
	}
	expected := swarm.Service{
		ID: "abcd",
		Spec: swarm.ServiceSpec{
			Annotations:  swarm.Annotations{Name: "foo"},
			UpdateConfig: &swarm.UpdateConfig{Parallelism: 2},
		},
		Meta: metaZero,
	}

	_, encoded, err := json.Encode(service)
	require.NoError(t, err)

	resp := &http.Response{StatusCode: http.StatusOK}
	_, reader, err := specedInspectResponse(defaultLookup, resp, encoded)
	require.NoError(t, err)

	actual := swarm.Service{}
	err = gojson.NewDecoder(reader).Decode(&actual)
	require.NoError(t, err)

	assert.Equal(t, expected, actual)
}
