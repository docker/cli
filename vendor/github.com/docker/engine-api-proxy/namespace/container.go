package pipeline

import (
	"io"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	json "github.com/docker/engine-api-proxy/json"
	"github.com/docker/engine-api-proxy/proxy"
	"github.com/docker/engine-api-proxy/routes"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

func newContainerCreateRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:          routes.ContainerCreate,
		RequestHandler: reqWithLookup(lookup, containerCreateRequest),
	}
}

func newContainerCommitRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:          routes.ContainerCommit,
		RequestHandler: reqWithLookup(lookup, containerCommitRequest),
	}
}

func containerCreateRequest(lookup LookupScope, _ *mux.Route, req *http.Request) (*http.Request, error) {
	scope := lookup()

	query := req.URL.Query()
	// add prefix to container name when receiving a user-defined container name
	if name := query.Get("name"); name != "" {
		query.Set("name", scope.ScopeName(name))
	}
	req.URL.RawQuery = query.Encode()

	// decode request body
	body, err := json.DecodeContainerCreate(req.Body)
	if err != nil {
		return nil, err
	}

	// add container labels
	if body.Config.Labels == nil {
		body.Config.Labels = make(map[string]string)
	}
	scope.AddLabels(body.Config.Labels)

	// scope network names
	hostConfig := body.HostConfig
	if hostConfig.NetworkMode.IsUserDefined() {
		v := hostConfig.NetworkMode.UserDefined()
		hostConfig.NetworkMode = container.NetworkMode(scope.ScopeName(v))
	}
	if body.NetworkingConfig != nil {
		var newEndpointsConfig map[string]*network.EndpointSettings = make(map[string]*network.EndpointSettings)
		for key, _ := range body.NetworkingConfig.EndpointsConfig {
			scopedKey := scope.ScopeName(key)
			newEndpointsConfig[scopedKey] = body.NetworkingConfig.EndpointsConfig[key]
		}
		body.NetworkingConfig.EndpointsConfig = newEndpointsConfig
	}

	return json.EncodeBody(&body, req)
}

// containerCommitRequest is the request handler for the /commit path.
// ($ docker container commit)
func containerCommitRequest(lookup LookupScope, route *mux.Route, req *http.Request) (*http.Request, error) {
	scope := lookup()
	query := req.URL.Query()
	containerName := query.Get("container")
	if containerName != "" {
		query.Set("container", scope.ScopeName(containerName))
	}
	req.URL.RawQuery = query.Encode()
	return req, nil
}

func newScopeContainerPath(route *routes.Route, lookup LookupScope, client client.APIClient) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:          route,
		RequestHandler: newContainerScopePath(lookup, client).request,
	}
}

// newConatinerInspectRoute returns a route which scopes the inspect response.
// The request is scoped by scopePath.
func newContainerInspectRoute(lookup LookupScope, client client.APIClient) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           routes.ContainerInspect,
		RequestHandler:  newContainerScopePath(lookup, client).request,
		ResponseHandler: respWithLookup(lookup, containerInspectResponse),
	}
}

func newContainerScopePath(lookup LookupScope, client client.APIClient) *scopePath {
	inspector := func(ctx context.Context, nameOrID string) (*labeled, error) {
		container, _, err := client.ContainerInspectWithRaw(ctx, nameOrID, false)
		if err != nil {
			return nil, err
		}
		return &labeled{ID: container.ID, Labels: container.Config.Labels}, nil
	}
	return &scopePath{lookup: lookup, inspector: inspector}
}

func containerInspectResponse(lookup LookupScope, resp *http.Response, body io.ReadCloser) (int, io.ReadCloser, error) {
	if resp.StatusCode != http.StatusOK {
		return -1, body, nil
	}

	scope := lookup()
	container, err := json.DecodeContainer(body)
	if err != nil {
		return -1, nil, err
	}
	container.Name = descopeContainerName(scope, container.Name)
	// TODO: descope any references to scoped resources
	return json.Encode(&container)
}

func newContainerListRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           routes.ContainerList,
		RequestHandler:  reqWithLookup(lookup, objectListRequest),
		ResponseHandler: respWithLookup(lookup, containerListResponse),
	}
}

func containerListResponse(lookup LookupScope, _ *http.Response, body io.ReadCloser) (int, io.ReadCloser, error) {
	scope := lookup()
	containers, err := json.DecodeContainers(body)
	if err != nil {
		return -1, nil, err
	}

	for i, container := range containers {
		// TODO: does this need to descope image, volume, network name as well?
		containers[i].Names = descopeNames(scope, container.Names)
	}

	return json.Encode(&containers)
}

func descopeNames(scope Scoper, names []string) []string {
	result := []string{}
	for _, name := range names {
		result = append(result, descopeContainerName(scope, name))
	}
	return result
}

func descopeContainerName(scope Scoper, name string) string {
	return "/" + scope.DescopeName(strings.TrimPrefix(name, "/"))
}
