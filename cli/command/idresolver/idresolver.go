// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package idresolver

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
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
		node, _, err := r.client.NodeInspectWithRaw(ctx, id)
		if err != nil {
			// TODO(thaJeztah): should error-handling be more specific, or is it ok to ignore any error?
			return id, nil //nolint:nilerr // ignore nil-error being returned, as this is a best-effort.
		}
		if node.Spec.Annotations.Name != "" {
			return node.Spec.Annotations.Name, nil
		}
		if node.Description.Hostname != "" {
			return node.Description.Hostname, nil
		}
		return id, nil
	case swarm.Service:
		service, _, err := r.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
		if err != nil {
			// TODO(thaJeztah): should error-handling be more specific, or is it ok to ignore any error?
			return id, nil //nolint:nilerr // ignore nil-error being returned, as this is a best-effort.
		}
		return service.Spec.Annotations.Name, nil
	default:
		return "", errors.Errorf("unsupported type")
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
