package swarm

import (
	"context"

	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/compose/convert"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
)

// GetStacks lists the swarm stacks with the number of services they contain.
//
// Deprecated: this function was for internal use and will be removed in the next release.
func GetStacks(ctx context.Context, apiClient client.ServiceAPIClient) ([]formatter.Stack, error) {
	services, err := apiClient.ServiceList(ctx, client.ServiceListOptions{
		Filters: getAllStacksFilter(),
	})
	if err != nil {
		return nil, err
	}

	idx := make(map[string]int, len(services))
	out := make([]formatter.Stack, 0, len(services))

	for _, svc := range services {
		name, ok := svc.Spec.Labels[convert.LabelNamespace]
		if !ok {
			return nil, errors.New("cannot get label " + convert.LabelNamespace + " for service " + svc.ID)
		}
		if i, ok := idx[name]; ok {
			out[i].Services++
			continue
		}
		idx[name] = len(out)
		out = append(out, formatter.Stack{Name: name, Services: 1})
	}
	return out, nil
}
