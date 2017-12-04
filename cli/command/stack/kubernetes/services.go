package kubernetes

import (
	"fmt"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/kubernetes/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RunServices is the kubernetes implementation of docker stack services
func RunServices(dockerCli *KubeCli, opts options.Services) error {
	// Initialize clients
	client, err := dockerCli.composeClient()
	if err != nil {
		return nil
	}
	stacks, err := dockerCli.stacks()
	if err != nil {
		return err
	}
	replicas := client.ReplicaSets()

	if _, err := stacks.Get(opts.Namespace, metav1.GetOptions{}); err != nil {
		fmt.Fprintf(dockerCli.Err(), "Nothing found in stack: %s\n", opts.Namespace)
		return nil
	}

	replicasList, err := replicas.List(metav1.ListOptions{LabelSelector: labels.SelectorForStack(opts.Namespace)})
	if err != nil {
		return err
	}

	servicesList, err := client.Services().List(metav1.ListOptions{LabelSelector: labels.SelectorForStack(opts.Namespace)})
	if err != nil {
		return err
	}

	// Convert Replicas sets and kubernetes services to swam services and formatter informations
	services, info, err := replicasToServices(replicasList, servicesList)
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
