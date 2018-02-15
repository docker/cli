package pipeline

import (
	"io"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/engine-api-proxy/json"
	"github.com/docker/engine-api-proxy/proxy"
	"github.com/docker/engine-api-proxy/routes"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

func newServiceListRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           routes.ServiceList,
		RequestHandler:  reqWithLookup(lookup, objectListRequest),
		ResponseHandler: respWithLookup(lookup, specedListResponse),
	}
}

func specedListResponse(lookup LookupScope, _ *http.Response, body io.ReadCloser) (int, io.ReadCloser, error) {
	scope := lookup()
	services, err := json.DecodeSpeceds(body)
	if err != nil {
		return -1, nil, err
	}

	for i, service := range services {
		// TODO: does this need to descope image, volume, network name as well?
		services[i].Spec.Name = scope.DescopeName(service.Spec.Name)
	}
	return json.Encode(&services)
}

func newServiceCreateRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:          routes.ServiceCreate,
		RequestHandler: reqWithLookup(lookup, serviceCreateRequest),
	}
}

func serviceCreateRequest(lookup LookupScope, _ *mux.Route, req *http.Request) (*http.Request, error) {
	scope := lookup()

	// decode request
	body, err := json.DecodeServiceCreate(req.Body)
	if err != nil {
		return nil, err
	}

	// scope name
	if body.Name != "" {
		body.Name = scope.ScopeName(body.Name)
	}

	// add labels
	if body.Labels == nil {
		body.Labels = make(map[string]string)
	}
	scope.AddLabels(body.Labels)

	// scope network names
	// NOTHING TO DO APPARENTLY

	// scope secret names
	for i, _ := range body.TaskTemplate.ContainerSpec.Secrets {
		body.TaskTemplate.ContainerSpec.Secrets[i].SecretName = scope.ScopeName(body.TaskTemplate.ContainerSpec.Secrets[i].SecretName)
	}

	// scope volume names
	for i, m := range body.TaskTemplate.ContainerSpec.Mounts {
		// volumes types:
		//	- volume
		// 	- bind
		// 	- tmpfs

		// ONLY SUPPORT NAMED VOLUMES FOR NOW
		if m.Type == mount.TypeVolume {
			body.TaskTemplate.ContainerSpec.Mounts[i].Source = scope.ScopeName(body.TaskTemplate.ContainerSpec.Mounts[i].Source)
		}
	}

	return json.EncodeBody(&body, req)
}

func newServiceInspectRoute(lookup LookupScope, client client.APIClient) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           routes.ServiceInspect,
		RequestHandler:  newServiceScopePath(lookup, client).request,
		ResponseHandler: respWithLookup(lookup, specedInspectResponse),
	}
}

func newServiceScopePath(lookup LookupScope, client client.APIClient) *scopePath {
	inspector := func(ctx context.Context, nameOrID string) (*labeled, error) {
		serviceInspectOpts := types.ServiceInspectOptions{
			InsertDefaults: true,
		}
		service, _, err := client.ServiceInspectWithRaw(ctx, nameOrID, serviceInspectOpts)
		if err != nil {
			return nil, err
		}
		return &labeled{ID: service.ID, Labels: service.Spec.Labels}, nil
	}
	return &scopePath{lookup: lookup, inspector: inspector}
}

func specedInspectResponse(lookup LookupScope, resp *http.Response, body io.ReadCloser) (int, io.ReadCloser, error) {
	if resp.StatusCode != http.StatusOK {
		return -1, body, nil
	}

	scope := lookup()
	service, err := json.DecodeSpeced(body)
	if err != nil {
		return -1, nil, err
	}
	service.Spec.Name = scope.DescopeName(service.Spec.Name)
	// TODO: descope any references to scoped resources
	return json.Encode(&service)
}
