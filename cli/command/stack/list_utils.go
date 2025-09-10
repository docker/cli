package stack

import (
	"context"
	"errors"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/moby/moby/client"
)

// getStacks lists the swarm stacks with the number of services they contain.
func getStacks(ctx context.Context, apiClient client.ServiceAPIClient) ([]stackSummary, error) {
	services, err := apiClient.ServiceList(ctx, client.ServiceListOptions{
		Filters: getAllStacksFilter(),
	})
	if err != nil {
		return nil, err
	}

	idx := make(map[string]int, len(services))
	out := make([]stackSummary, 0, len(services))

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
		out = append(out, stackSummary{Name: name, Services: 1})
	}
	return out, nil
}
