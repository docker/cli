// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package idresolver

import (
	"context"
	"errors"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

// IDResolver provides ID to Name resolution.
type IDResolver struct {
	client    client.APIClient
	noResolve bool
	cache     map[string]string
}

// New creates a new IDResolver.
func New(apiClient client.APIClient, noResolve bool) *IDResolver {
	return &IDResolver{
		client:    apiClient,
		noResolve: noResolve,
		cache:     make(map[string]string),
	}
}

func (r *IDResolver) get(ctx context.Context, t any, id string) (string, error) {
	switch t.(type) {
	case swarm.Node:
		res, err := r.client.NodeInspect(ctx, id, client.NodeInspectOptions{})
		if err != nil {
			// TODO(thaJeztah): should error-handling be more specific, or is it ok to ignore any error?
			return id, nil //nolint:nilerr // ignore nil-error being returned, as this is a best-effort.
		}
		if res.Node.Spec.Annotations.Name != "" {
			return res.Node.Spec.Annotations.Name, nil
		}
		if res.Node.Description.Hostname != "" {
			return res.Node.Description.Hostname, nil
		}
		return id, nil
	case swarm.Service:
		res, err := r.client.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
		if err != nil {
			// TODO(thaJeztah): should error-handling be more specific, or is it ok to ignore any error?
			return id, nil //nolint:nilerr // ignore nil-error being returned, as this is a best-effort.
		}
		return res.Service.Spec.Annotations.Name, nil
	default:
		return "", errors.New("unsupported type")
	}
}

// Resolve will attempt to resolve an ID to a Name by querying the manager.
// Results are stored into a cache.
// If the `-n` flag is used in the command-line, resolution is disabled.
func (r *IDResolver) Resolve(ctx context.Context, t any, id string) (string, error) {
	if r.noResolve {
		return id, nil
	}
	if name, ok := r.cache[id]; ok {
		return name, nil
	}
	name, err := r.get(ctx, t, id)
	if err != nil {
		return "", err
	}
	r.cache[id] = name
	return name, nil
}
