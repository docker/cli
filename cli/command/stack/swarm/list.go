package swarm

import (
	"context"

	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

// GetStacks lists the swarm stacks.
func GetStacks(ctx context.Context, apiClient client.ServiceAPIClient) ([]*formatter.Stack, error) {
	services, err := apiClient.ServiceList(
		ctx,
		types.ServiceListOptions{Filters: getAllStacksFilter()})
	if err != nil {
		return nil, err
	}
	m := make(map[string]*formatter.Stack)
	for _, service := range services {
		labels := service.Spec.Labels
		name, ok := labels[convert.LabelNamespace]
		if !ok {
			return nil, errors.Errorf("cannot get label %s for service %s",
				convert.LabelNamespace, service.ID)
		}
		ztack, ok := m[name]
		if !ok {
			m[name] = &formatter.Stack{
				Name:     name,
				Services: 1,
			}
		} else {
			ztack.Services++
		}
	}
	stacks := make([]*formatter.Stack, 0, len(m))
	for _, stack := range m {
		stacks = append(stacks, stack)
	}
	return stacks, nil
}
