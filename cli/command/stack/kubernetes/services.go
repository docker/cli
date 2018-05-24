package kubernetes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/kubernetes/labels"
	"github.com/docker/docker/api/types/filters"
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
	names := f.Get("name")
	sort.Strings(names)
	for _, n := range names {
		if strings.HasPrefix(n, stackName+"_") {
			// also accepts with unprefixed service name (for compat with existing swarm scripts where service names are prefixed by stack names)
			names = append(names, strings.TrimPrefix(n, stackName+"_"))
		}
	}
	selectors := append(generateSelector(parseLabelFilters(f.Get("label"))), labels.SelectorForStack(stackName, names...))
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

	// Convert Replicas sets and kubernetes services to swam services and formatter informations
	services, info, err := convertToServices(replicasList, daemonsList, servicesList)
	if err != nil {
		return err
	}

	if opts.Quiet {
		info = map[string]formatter.ServiceListInfo{}
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
		Format: formatter.NewServiceListFormat(format, opts.Quiet),
	}
	return formatter.ServiceListWrite(servicesCtx, services, info)
}
