package kubernetes

import (
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetStacks lists the kubernetes stacks.
func GetStacks(kubeCli *KubeCli, opts options.List) ([]*formatter.Stack, error) {
	composeClient, err := kubeCli.composeClient()
	if err != nil {
		return nil, err
	}
	stackSvc, err := composeClient.Stacks(opts.AllNamespaces)
	if err != nil {
		return nil, err
	}
	stacks, err := stackSvc.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var formattedStacks []*formatter.Stack
	for _, stack := range stacks {
		formattedStacks = append(formattedStacks, &formatter.Stack{
			Name:         stack.name,
			Services:     len(stack.getServices()),
			Orchestrator: "Kubernetes",
			Namespace:    stack.namespace,
		})
	}
	return formattedStacks, nil
}
