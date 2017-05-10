package pipeline

import (
	"net/http"

	"github.com/docker/docker/client"
	"github.com/docker/engine-api-proxy/json"
	"github.com/docker/engine-api-proxy/proxy"
	"github.com/docker/engine-api-proxy/routes"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

func newNetworkScopePath(lookup LookupScope, client client.APIClient) *scopePath {
	inspector := func(ctx context.Context, nameOrID string) (*labeled, error) {
		network, _, err := client.NetworkInspectWithRaw(ctx, nameOrID, false)
		if err != nil {
			return nil, err
		}
		return &labeled{ID: network.ID, Labels: network.Labels}, nil
	}
	return &scopePath{lookup: lookup, inspector: inspector}
}

func newNetworkCreateRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:          routes.NetworkCreate,
		RequestHandler: reqWithLookup(lookup, objectCreateRequest),
	}
}

func newNetworkConnectRoute(route *routes.Route, lookup LookupScope, client client.APIClient) proxy.MiddlewareRoute {
	containerScopePath := newContainerScopePath(lookup, client)
	connect := &networkConnectRoute{
		containerScopePath: containerScopePath,
		lookup:             lookup,
	}
	return proxy.MiddlewareRoute{
		Route:          route,
		RequestHandler: connect.request,
	}
}

type networkConnectRoute struct {
	containerScopePath *scopePath
	lookup             LookupScope
}

func (n *networkConnectRoute) request(route *mux.Route, req *http.Request) (*http.Request, error) {
	scope := n.lookup()

	net, err := json.DecodeNetworkConnect(req.Body)
	if err != nil {
		return nil, err
	}

	scoped, err := n.containerScopePath.scope(req.Context(), net.Container, scope)
	if err != nil {
		return nil, err
	}
	net.Container = scoped
	return json.EncodeBody(&net, req)
}
