package proxy

import (
	"io"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api-proxy/errors"
	"github.com/docker/engine-api-proxy/routes"
	"github.com/gorilla/mux"
)

// MiddlewareRoute is the definition of a route with handlers for modifying the
// request and the response that match the route.
type MiddlewareRoute struct {
	Route           *routes.Route
	RequestHandler  RequestHandler
	ResponseHandler ResponseHandler
}

// RequestHandler accepts a request, and the route that was matched, and returns
// a new request object. The new request will be made against the backend.
type RequestHandler func(*mux.Route, *http.Request) (*http.Request, error)

// ResponseHandler accepts a response and the body of a response, and returns a
// new body for the response
type ResponseHandler func(*http.Response, io.ReadCloser) (int, io.ReadCloser, error)

type routeHandler struct {
	routes      []MiddlewareRoute
	passthru    *passthru
	cancellable bool
	route       *mux.Route
}

func newRouteHandler(route *mux.Route, routes []MiddlewareRoute, passthru *passthru, cancellable bool) *routeHandler {
	return &routeHandler{
		routes:      routes,
		passthru:    passthru,
		cancellable: cancellable,
		route:       route,
	}
}

func (m *routeHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	req, err := m.wrapRequest(req)
	if err != nil {
		if httperror, ok := err.(*errors.HTTPError); ok {
			httperror.Write(writer)
			return
		}
		writeProxyError(writer)
		return
	}

	if err := m.passthru.Passthru(writer, req, m, m.cancellable); err != nil {
		log.Warn(err)
		writeProxyError(writer)
	}
}

func (m *routeHandler) wrapRequest(req *http.Request) (*http.Request, error) {
	var err error
	for _, route := range m.routes {
		if route.RequestHandler == nil {
			continue
		}
		req, err = route.RequestHandler(m.route, req)
		if err != nil {
			log.Warnf("Error in handler %s", err)
			return nil, err
		}
	}
	return req, nil
}

func (m *routeHandler) RewriteBody(resp *http.Response, body io.ReadCloser) (int, io.ReadCloser, error) {
	var size = -1
	var err error
	// apply the rewriters in reverse order
	for i := len(m.routes) - 1; i >= 0; i-- {
		if m.routes[i].ResponseHandler == nil {
			continue
		}
		size, body, err = m.routes[i].ResponseHandler(resp, body)
		if err != nil {
			return -1, nil, err
		}
	}
	return size, body, nil
}

func newHandlerFromMiddleware(middlewareRoutes []MiddlewareRoute, dailer BackendDialer) (http.Handler, error) {
	mapping := map[*routes.Route][]MiddlewareRoute{}
	for _, route := range middlewareRoutes {
		mapping[route.Route] = append(mapping[route.Route], route)
	}

	// Add defaults for cancellable routes
	for route := range cancellableRoutes {
		if _, exists := mapping[route]; exists {
			continue
		}
		mapping[route] = []MiddlewareRoute{}
	}

	router := mux.NewRouter()
	passthru := newPassthru(dailer)

	// TODO: what if order of routes matters? Order routes using constants
	for route, chain := range mapping {
		cancellable := cancellableRoutes[route]

		log.Debugf("Adding route %s", route)
		muxRoute := route.AsMuxRoute()
		handler := newRouteHandler(muxRoute, chain, passthru, cancellable)
		router.AddRoute(muxRoute.Handler(handler))

		versioned := route.Versioned()
		log.Debugf("Adding route %s", versioned)
		muxRoute = versioned.AsMuxRoute()
		handler = newRouteHandler(muxRoute, chain, passthru, cancellable)
		router.AddRoute(muxRoute.Handler(handler))
	}
	router.Handle("/{any:.*}", newRouteHandler(&mux.Route{}, nil, passthru, false))
	return router, nil
}

var cancellableRoutes = map[*routes.Route]bool{
	routes.PluginPull:     true,
	routes.PluginPush:     true,
	routes.ImageBuild:     true,
	routes.ImageCreate:    true,
	routes.ImagePush:      true,
	routes.Events:         true,
	routes.ContainerLogs:  true,
	routes.ContainerStats: true,
	routes.ServiceLogs:    true,
}
