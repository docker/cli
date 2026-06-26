package stack

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/client"
)

// validateStackName checks if the provided string is a valid stack name (namespace).
// It currently only does a rudimentary check if the string is empty, or consists
// of only whitespace and quoting characters.
func validateStackName(namespace string) error {
	v := strings.TrimFunc(namespace, quotesOrWhitespace)
	if v == "" {
		return fmt.Errorf("invalid stack name: %q", namespace)
	}
	return nil
}

func validateStackNames(namespaces []string) error {
	for _, ns := range namespaces {
		if err := validateStackName(ns); err != nil {
			return err
		}
	}
	return nil
}

func quotesOrWhitespace(r rune) bool {
	return unicode.IsSpace(r) || r == '"' || r == '\''
}

func getStackFilter(namespace string) client.Filters {
	return make(client.Filters).Add("label", convert.LabelNamespace+"="+namespace)
}

func getStackFilterFromOpt(namespace string, opt opts.FilterOpt) client.Filters {
	filter := opt.Value()
	filter.Add("label", convert.LabelNamespace+"="+namespace)
	return filter
}

func getAllStacksFilter() client.Filters {
	return make(client.Filters).Add("label", convert.LabelNamespace)
}

func getStackServices(ctx context.Context, apiclient client.APIClient, namespace string) (client.ServiceListResult, error) {
	return apiclient.ServiceList(ctx, client.ServiceListOptions{Filters: getStackFilter(namespace)})
}

func getStackNetworks(ctx context.Context, apiclient client.APIClient, namespace string) (client.NetworkListResult, error) {
	return apiclient.NetworkList(ctx, client.NetworkListOptions{Filters: getStackFilter(namespace)})
}

func getStackSecrets(ctx context.Context, apiclient client.APIClient, namespace string) (client.SecretListResult, error) {
	return apiclient.SecretList(ctx, client.SecretListOptions{Filters: getStackFilter(namespace)})
}

func getStackConfigs(ctx context.Context, apiclient client.APIClient, namespace string) (client.ConfigListResult, error) {
	return apiclient.ConfigList(ctx, client.ConfigListOptions{Filters: getStackFilter(namespace)})
}

func getStackTasks(ctx context.Context, apiclient client.APIClient, namespace string) (client.TaskListResult, error) {
	return apiclient.TaskList(ctx, client.TaskListOptions{Filters: getStackFilter(namespace)})
}
