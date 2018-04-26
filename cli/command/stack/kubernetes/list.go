package kubernetes

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetStacks lists the kubernetes stacks
func GetStacks(dockerCli command.Cli, opts options.List, kopts Options) ([]*formatter.Stack, error) {
	kubeCli, err := WrapCli(dockerCli, kopts)
	if err != nil {
		return nil, err
	}
	if opts.AllNamespaces || len(opts.Namespaces) == 0 {
		return getStacks(kubeCli, opts)
	}
	return getStacksWithNamespaces(kubeCli, opts)
}

func getStacks(kubeCli *KubeCli, opts options.List) ([]*formatter.Stack, error) {
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

func getStacksWithNamespaces(kubeCli *KubeCli, opts options.List) ([]*formatter.Stack, error) {
	stacks := []*formatter.Stack{}
	for _, namespace := range removeDuplicates(opts.Namespaces) {
		kubeCli.kubeNamespace = namespace
		ss, err := getStacks(kubeCli, opts)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, ss...)
	}
	return stacks, nil
}

func removeDuplicates(namespaces []string) []string {
	found := make(map[string]bool)
	results := namespaces[:0]
	for _, n := range namespaces {
		if !found[n] {
			results = append(results, n)
			found[n] = true
		}
	}
	return results
}
