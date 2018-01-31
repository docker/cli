package v1beta2 // import "github.com/docker/cli/kubernetes/compose/v1beta2"

import (
	"errors"
)

// Scale sets the number of replicas for a given service
func (s *Stack) Scale(service string, replicas int) (*Stack, error) {
	stack, err := s.Clone()
	if err != nil {
		return nil, err
	}
	for i, svc := range stack.Spec.Stack.Services {
		if svc.Name != service {
			continue
		}
		r := uint64(replicas)
		stack.Spec.Stack.Services[i].Deploy.Replicas = &r
		return stack, nil
	}
	return nil, errors.New(service + " not found")
}
