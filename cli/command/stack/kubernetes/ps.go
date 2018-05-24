package kubernetes

import (
	"fmt"
	"sort"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/task"
	"github.com/docker/docker/api/types/swarm"
	apiv1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var supportedPSFilters = map[string]bool{
	"name":    true,
	"service": true,
	"node":    true,
}

// RunPS is the kubernetes implementation of docker stack ps
func RunPS(dockerCli *KubeCli, options options.PS) error {
	filters := options.Filter.Value()
	if err := filters.Validate(supportedPSFilters); err != nil {
		return err
	}
	client, err := dockerCli.composeClient()
	if err != nil {
		return err
	}
	stacks, err := client.Stacks(false)
	if err != nil {
		return err
	}
	stackName := options.Namespace
	_, err = stacks.Get(stackName)
	if apierrs.IsNotFound(err) {
		return fmt.Errorf("nothing found in stack: %s", stackName)
	}
	if err != nil {
		return err
	}
	pods, err := fetchPods(stackName, client.Pods(), filters)
	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("nothing found in stack: %s", stackName)
	}
	return printTasks(dockerCli, options, stackName, client, pods)
}

func printTasks(dockerCli command.Cli, options options.PS, namespace string, client corev1.NodesGetter, pods []apiv1.Pod) error {
	format := options.Format
	if format == "" {
		format = task.DefaultFormat(dockerCli.ConfigFile(), options.Quiet)
	}

	tasks := make([]swarm.Task, len(pods))
	for i, pod := range pods {
		tasks[i] = podToTask(pod)
	}
	sort.Stable(tasksBySlot(tasks))

	names := map[string]string{}
	nodes := map[string]string{}

	n, err := client.Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for i, task := range tasks {
		nodeValue, err := resolveNode(pods[i].Spec.NodeName, n, options.NoResolve)
		if err != nil {
			return err
		}
		names[task.ID] = fmt.Sprintf("%s_%s", namespace, pods[i].Name)
		nodes[task.ID] = nodeValue
	}

	tasksCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewTaskFormat(format, options.Quiet),
		Trunc:  !options.NoTrunc,
	}

	return formatter.TaskWrite(tasksCtx, tasks, names, nodes)
}

func resolveNode(name string, nodes *apiv1.NodeList, noResolve bool) (string, error) {
	// Here we have a name and we need to resolve its identifier. To mimic swarm behavior
	// we need to resolve to the id when noResolve is set, otherwise we return the name.
	if noResolve {
		for _, node := range nodes.Items {
			if node.Name == name {
				return string(node.UID), nil
			}
		}
		return "", fmt.Errorf("could not find node '%s'", name)
	}
	return name, nil
}
