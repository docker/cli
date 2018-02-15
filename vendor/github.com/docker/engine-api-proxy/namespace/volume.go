package pipeline

import (
	"io"
	"net/http"

	"github.com/docker/docker/client"
	json "github.com/docker/engine-api-proxy/json"
	"github.com/docker/engine-api-proxy/proxy"
	"github.com/docker/engine-api-proxy/routes"
	"golang.org/x/net/context"
)

// newVolumeListRoute creates the /volumes route, for `docker volume ls`
func newVolumeListRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           routes.VolumeList,
		RequestHandler:  reqWithLookup(lookup, objectListRequest),
		ResponseHandler: respWithLookup(lookup, volumeListResponse),
	}
}

// volumeListResponse handles "/volumes" route's response
func volumeListResponse(lookup LookupScope, _ *http.Response, body io.ReadCloser) (int, io.ReadCloser, error) {
	scope := lookup()

	volumeListBody, err := json.DecodeVolumeList(body)
	if err != nil {
		return -1, nil, err
	}

	for i, volume := range volumeListBody.Volumes {
		volumeListBody.Volumes[i].Name = scope.DescopeName(volume.Name)
	}

	return json.Encode(&volumeListBody)
}

func newVolumeScopePath(lookup LookupScope, client client.APIClient) *scopePath {
	inspector := func(ctx context.Context, nameOrID string) (*labeled, error) {
		volume, _, err := client.VolumeInspectWithRaw(ctx, nameOrID)
		if err != nil {
			return nil, err
		}
		return &labeled{Labels: volume.Labels}, nil
	}
	return &scopePath{lookup: lookup, inspector: inspector}
}

func newVolumeCreateRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           routes.VolumeCreate,
		RequestHandler:  reqWithLookup(lookup, objectCreateRequest),
		ResponseHandler: respWithLookup(lookup, objectInspectResponse),
	}
}
