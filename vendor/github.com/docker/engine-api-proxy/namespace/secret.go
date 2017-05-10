package pipeline

import (
	"github.com/docker/docker/client"
	"github.com/docker/engine-api-proxy/proxy"
	"github.com/docker/engine-api-proxy/routes"
	"golang.org/x/net/context"
)

func newSecretScopePath(lookup LookupScope, client client.APIClient) *scopePath {
	inspector := func(ctx context.Context, nameOrID string) (*labeled, error) {
		secret, _, err := client.SecretInspectWithRaw(ctx, nameOrID)
		if err != nil {
			return nil, err
		}
		return &labeled{ID: secret.ID, Labels: secret.Spec.Labels}, nil
	}
	return &scopePath{lookup: lookup, inspector: inspector}
}

func newSecretCreateRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:          routes.SecretCreate,
		RequestHandler: reqWithLookup(lookup, objectCreateRequest),
	}
}

func newSecretInspectRoute(lookup LookupScope, client client.APIClient) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           routes.SecretInspect,
		RequestHandler:  newSecretScopePath(lookup, client).request,
		ResponseHandler: respWithLookup(lookup, specedInspectResponse),
	}
}

func newSecretListRoute(lookup LookupScope) proxy.MiddlewareRoute {
	return proxy.MiddlewareRoute{
		Route:           routes.SecretList,
		RequestHandler:  reqWithLookup(lookup, objectListRequest),
		ResponseHandler: respWithLookup(lookup, specedListResponse),
	}
}
