package kubernetes

import (
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/kubernetes/labels"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var supportedServicesFilters = map[string]bool{
	"mode":  true,
	"name":  true,
	"label": true,
}

func generateSelector(labels map[string][]string) []string {
	var result []string
	for k, v := range labels {
		for _, val := range v {
			result = append(result, fmt.Sprintf("%s=%s", k, val))
		}
		if len(v) == 0 {
			result = append(result, k)
		}
	}
	return result
}

func parseLabelFilters(rawFilters []string) map[string][]string {
	labels := map[string][]string{}
	for _, rawLabel := range rawFilters {
		v := strings.SplitN(rawLabel, "=", 2)
		key := v[0]
		if len(v) > 1 {
			labels[key] = append(labels[key], v[1])
		} else if _, ok := labels[key]; !ok {
			labels[key] = []string{}
		}
	}
	return labels
}

func generateLabelSelector(f filters.Args, stackName string) string {
	selectors := append(generateSelector(parseLabelFilters(f.Get("label"))), labels.SelectorForStack(stackName))
	return strings.Join(selectors, ",")
}

func getResourcesForServiceList(dockerCli *KubeCli, filters filters.Args, labelSelector string) (*appsv1beta2.ReplicaSetList, *appsv1beta2.DaemonSetList, *corev1.ServiceList, error) {
	client, err := dockerCli.composeClient()
	if err != nil {
		return nil, nil, nil, err
	}
	modes := filters.Get("mode")
	replicas := &appsv1beta2.ReplicaSetList{}
	if len(modes) == 0 || filters.ExactMatch("mode", "replicated") {
		if replicas, err = client.ReplicaSets().List(metav1.ListOptions{LabelSelector: labelSelector}); err != nil {
			return nil, nil, nil, err
		}
	}
	daemons := &appsv1beta2.DaemonSetList{}
	if len(modes) == 0 || filters.ExactMatch("mode", "global") {
		if daemons, err = client.DaemonSets().List(metav1.ListOptions{LabelSelector: labelSelector}); err != nil {
			return nil, nil, nil, err
		}
	}
	services, err := client.Services().List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, nil, nil, err
	}
	return replicas, daemons, services, nil
}

// RunServices is the kubernetes implementation of docker stack services
func RunServices(dockerCli *KubeCli, opts options.Services) error {
	filters := opts.Filter.Value()
	if err := filters.Validate(supportedServicesFilters); err != nil {
		return err
	}
	client, err := dockerCli.composeClient()
	if err != nil {
		return nil
	}
	stacks, err := client.Stacks(false)
	if err != nil {
		return nil
	}
	stackName := opts.Namespace
	_, err = stacks.Get(stackName)
	if apierrs.IsNotFound(err) {
		return fmt.Errorf("nothing found in stack: %s", stackName)
	}
	if err != nil {
		return err
	}

	labelSelector := generateLabelSelector(filters, stackName)
	replicasList, daemonsList, servicesList, err := getResourcesForServiceList(dockerCli, filters, labelSelector)
	if err != nil {
		return err
	}

	// Convert Replicas sets and kubernetes services to swarm services and formatter information
	services, info, err := convertToServices(replicasList, daemonsList, servicesList)
	if err != nil {
		return err
	}
	services = filterServicesByName(services, filters.Get("name"), stackName)

	if opts.Quiet {
		info = map[string]service.ListInfo{}
	}

	format := opts.Format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().ServicesFormat) > 0 && !opts.Quiet {
			format = dockerCli.ConfigFile().ServicesFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	servicesCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: service.NewListFormat(format, opts.Quiet),
	}
	return service.ListFormatWrite(servicesCtx, services, info)
}

func filterServicesByName(services []swarm.Service, names []string, stackName string) []swarm.Service {
	if len(names) == 0 {
		return services
	}
	prefix := stackName + "_"
	// Accepts unprefixed service name (for compatibility with existing swarm scripts where service names are prefixed by stack names)
	for i, n := range names {
		if !strings.HasPrefix(n, prefix) {
			names[i] = stackName + "_" + n
		}
	}
	// Filter services
	result := []swarm.Service{}
	for _, s := range services {
		for _, n := range names {
			if strings.HasPrefix(s.Spec.Name, n) {
				result = append(result, s)
			}
		}
	}
	return result
}
