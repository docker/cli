package pipeline

import (
	"io"
	"net/http"

	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/engine-api-proxy/errors"
	"github.com/docker/engine-api-proxy/json"
	"github.com/docker/engine-api-proxy/proxy"
	"github.com/docker/engine-api-proxy/routes"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

func reqWithLookup(
	lookup LookupScope,
	handler func(LookupScope, *mux.Route, *http.Request) (*http.Request, error),
) proxy.RequestHandler {
	return func(route *mux.Route, req *http.Request) (*http.Request, error) {
		return handler(lookup, route, req)
	}
}

func respWithLookup(
	lookup LookupScope,
	handler func(LookupScope, *http.Response, io.ReadCloser) (int, io.ReadCloser, error),
) proxy.ResponseHandler {
	return func(resp *http.Response, body io.ReadCloser) (int, io.ReadCloser, error) {
		return handler(lookup, resp, body)
	}
}

func newObjectListRoute(route *routes.Route, lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           route,
		RequestHandler:  reqWithLookup(lookup, objectListRequest),
		ResponseHandler: respWithLookup(lookup, objectListResponse),
	}
}

func objectListRequest(lookup LookupScope, _ *mux.Route, req *http.Request) (*http.Request, error) {
	scope := lookup()

	if err := req.ParseForm(); err != nil {
		return nil, err
	}
	query := req.Form
	reqFilters, err := filters.FromParam(query.Get("filters"))
	if err != nil {
		return nil, err
	}

	scope.UpdateFilter(reqFilters)
	filterJSON, err := filters.ToParam(reqFilters)
	if err != nil {
		return nil, err
	}

	query.Set("filters", filterJSON)
	req.URL.RawQuery = query.Encode()
	return req, nil
}

func objectListResponse(lookup LookupScope, _ *http.Response, body io.ReadCloser) (int, io.ReadCloser, error) {
	scope := lookup()
	nameds, err := json.DecodeNameds(body)
	if err != nil {
		return -1, nil, err
	}

	for i, named := range nameds {
		nameds[i].Name = scope.DescopeName(named.Name)
	}
	return json.Encode(nameds)
}

type scopePathFunc func(lookup LookupScope, client client.APIClient) *scopePath

func newObjectInspectRoute(
	route *routes.Route,
	scopePathFunc scopePathFunc,
	lookup LookupScope,
	client client.APIClient,
) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           route,
		RequestHandler:  scopePathFunc(lookup, client).request,
		ResponseHandler: respWithLookup(lookup, objectInspectResponse),
	}
}

func objectInspectResponse(lookup LookupScope, resp *http.Response, body io.ReadCloser) (int, io.ReadCloser, error) {
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusAlreadyReported {
		return -1, body, nil
	}

	scope := lookup()
	obj, err := json.DecodeNamed(body)
	if err != nil {
		return -1, nil, err
	}
	obj.Name = scope.DescopeName(obj.Name)
	return json.Encode(&obj)
}

func objectCreateRequest(lookup LookupScope, _ *mux.Route, req *http.Request) (*http.Request, error) {
	scope := lookup()

	named, err := json.DecodeNamed(req.Body)
	if err != nil {
		return nil, err
	}
	named.Name = scope.ScopeName(named.Name)
	if named.Labels == nil {
		named.Labels = make(map[string]string)
	}
	scope.AddLabels(named.Labels)
	return json.EncodeBody(&named, req)
}

func newObjectUpdateRoute(route *routes.Route, lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:          route,
		RequestHandler: reqWithLookup(lookup, newObjectUpdateRequest),
	}
}

func newObjectUpdateRequest(lookup LookupScope, _ *mux.Route, req *http.Request) (*http.Request, error) {
	scope := lookup()

	named, err := json.DecodeNamed(req.Body)
	if err != nil {
		return nil, err
	}
	named.Name = scope.ScopeName(named.Name)
	return json.EncodeBody(&named, req)
}

func newScopeObjectPath(
	route *routes.Route,
	scopePathFunc scopePathFunc,
	lookup LookupScope,
	client client.APIClient,
) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:          route,
		RequestHandler: scopePathFunc(lookup, client).request,
	}
}

// inspector is the common interface for all client inspect methods
type inspector func(ctx context.Context, nameOrID string) (*labeled, error)

type labeled struct {
	ID     string
	Labels map[string]string
}

type scopePath struct {
	lookup    LookupScope
	inspector inspector
	// getVars is a shim for testing, could be removed with a patch to
	// gorilla/mux to allow setting the relevant context values
	getVars func(req *http.Request) map[string]string
}

func (n *scopePath) request(route *mux.Route, req *http.Request) (*http.Request, error) {
	if n.getVars == nil {
		n.getVars = mux.Vars
	}
	nameOrID := n.getVars(req)["name"]
	version := n.getVars(req)["version"]
	scope := n.lookup()

	// TODO: maybe cache these lookups?
	scoped, err := n.scope(req.Context(), nameOrID, scope)
	if err != nil {
		return nil, err
	}

	url, err := route.URL("name", scoped, "version", version)
	url.RawQuery = req.URL.RawQuery
	req.URL = url

	return req, err
}

// needsScoping determines whether a docker object identifier (id or name) needs
// to be scoped (prefixed) before the request is sent to the actual Docker daemon.
// This function is used in the request handler of the proxy.
func (n *scopePath) needsScoping(ctx context.Context, nameOrID string, scope Scoper) (bool, error) {
	obj, err := n.inspector(ctx, nameOrID)
	switch {
	case err == nil:
		// nameOrID matched either an id, or a name (prefixed or not).
		// Only return the request unmodified if it's an id or
		// the container name isn't already scoped.
		if scope.IsInScope(obj.Labels) && isUnscopedName(nameOrID, scope) {
			return false, nil
		}
	case client.IsErrNotFound(err):
		// nothing found with that nameOrID, so mutate the name
	default:
		return false, err
	}
	return true, nil
}

// scope returns scoped name, or an error if the scoped name matches
// an object that is out of scope
func (n *scopePath) scope(ctx context.Context, nameOrID string, scope Scoper) (string, error) {
	needs, err := n.needsScoping(ctx, nameOrID, scope)
	switch {
	case err != nil:
		return "", err
	case !needs:
		return nameOrID, err
	}

	scoped := scope.ScopeName(nameOrID)
	obj, err := n.inspector(ctx, scoped)
	switch {
	case err == nil:
		if !scope.IsInScope(obj.Labels) {
			return "", errors.NewHTTPError(
				http.StatusNotFound, fmt.Sprintf("%q not found", nameOrID))
		}
	case client.IsErrNotFound(err):
	default:
		return "", err
	}
	return scoped, nil
}

func isUnscopedName(nameOrID string, scope Scoper) bool {
	return nameOrID == scope.DescopeName(nameOrID)
}
