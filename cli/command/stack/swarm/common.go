package swarm

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

func getStackFilter(namespace string) filters.Args {
	filter := filters.NewArgs()
	filter.Add("label", convert.LabelNamespace+"="+namespace)
	return filter
}

func getStackServiceFilter(namespace string) filters.Args {
	return getStackFilter(namespace)
}

func getStackFilterFromOpt(namespace string, opt opts.FilterOpt) filters.Args {
	filter := opt.Value()
	filter.Add("label", convert.LabelNamespace+"="+namespace)
	return filter
}

func getAllStacksFilter() filters.Args {
	filter := filters.NewArgs()
	filter.Add("label", convert.LabelNamespace)
	return filter
}

func getStackServices(ctx context.Context, apiclient client.APIClient, namespace string) ([]swarm.Service, error) {
	return apiclient.ServiceList(ctx, types.ServiceListOptions{Filters: getStackServiceFilter(namespace)})
}

func getStackNetworks(ctx context.Context, apiclient client.APIClient, namespace string) ([]types.NetworkResource, error) {
	return apiclient.NetworkList(ctx, types.NetworkListOptions{Filters: getStackFilter(namespace)})
}

func getStackSecrets(ctx context.Context, apiclient client.APIClient, namespace string) ([]swarm.Secret, error) {
	return apiclient.SecretList(ctx, types.SecretListOptions{Filters: getStackFilter(namespace)})
}

func getStackConfigs(ctx context.Context, apiclient client.APIClient, namespace string) ([]swarm.Config, error) {
	return apiclient.ConfigList(ctx, types.ConfigListOptions{Filters: getStackFilter(namespace)})
}

// validateStackName checks if the provided string is a valid stack name (namespace).
//
// It currently only does a rudimentary check if the string is empty, or consists
// of only whitespace and quoting characters.
func validateStackName(namespace string) error {
	v := strings.TrimFunc(namespace, quotesOrWhitespace)
	if len(v) == 0 {
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
