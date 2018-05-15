package kubernetes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetStacks lists the kubernetes stacks
func GetStacks(kubeCli *KubeCli, opts options.List) ([]*formatter.Stack, error) {
	if opts.AllNamespaces || len(opts.Namespaces) == 0 {
		if isAllNamespacesDisabled(kubeCli.ConfigFile().Kubernetes) {
			opts.AllNamespaces = true
		}
		return getStacksWithAllNamespaces(kubeCli, opts)
	}
	return getStacksWithNamespaces(kubeCli, opts, removeDuplicates(opts.Namespaces))
}

func isAllNamespacesDisabled(kubeCliConfig *configfile.KubernetesConfig) bool {
	return kubeCliConfig == nil || kubeCliConfig != nil && kubeCliConfig.AllNamespaces != "disabled"
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

func getStacksWithAllNamespaces(kubeCli *KubeCli, opts options.List) ([]*formatter.Stack, error) {
	stacks, err := getStacks(kubeCli, opts)
	if !apierrs.IsForbidden(err) {
		return stacks, err
	}
	namespaces, err2 := getUserVisibleNamespaces(*kubeCli)
	if err2 != nil {
		return nil, errors.Wrap(err2, "failed to query user visible namespaces")
	}
	if namespaces == nil {
		// UCP API not present, fall back to Kubernetes error
		return nil, err
	}
	opts.AllNamespaces = false
	return getStacksWithNamespaces(kubeCli, opts, namespaces)
}

func getUserVisibleNamespaces(dockerCli command.Cli) ([]string, error) {
	host := dockerCli.Client().DaemonHost()
	endpoint, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	endpoint.Scheme = "https"
	endpoint.Path = "/kubernetesNamespaces"
	resp, err := dockerCli.Client().HTTPClient().Get(endpoint.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "received %d status and unable to read response", resp.StatusCode)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		nms := &core_v1.NamespaceList{}
		if err := json.Unmarshal(body, nms); err != nil {
			return nil, errors.Wrapf(err, "unmarshal failed: %s", string(body))
		}
		namespaces := make([]string, len(nms.Items))
		for i, namespace := range nms.Items {
			namespaces[i] = namespace.Name
		}
		return namespaces, nil
	case http.StatusNotFound:
		// UCP API not present
		return nil, nil
	default:
		return nil, fmt.Errorf("received %d status while retrieving namespaces: %s", resp.StatusCode, string(body))
	}
}

func getStacksWithNamespaces(kubeCli *KubeCli, opts options.List, namespaces []string) ([]*formatter.Stack, error) {
	stacks := []*formatter.Stack{}
	for _, namespace := range namespaces {
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
