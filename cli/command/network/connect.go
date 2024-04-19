package network

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/network"
	"github.com/spf13/cobra"
)

type connectOptions struct {
	network      string
	container    string
	ipaddress    string
	ipv6address  string
	links        opts.ListOpts
	aliases      []string
	linklocalips []string
	driverOpts   []string
}

func newConnectCommand(dockerCli command.Cli) *cobra.Command {
	options := connectOptions{
		links: opts.NewListOpts(opts.ValidateLink),
	}

	cmd := &cobra.Command{
		Use:   "connect [OPTIONS] NETWORK CONTAINER",
		Short: "Connect a container to a network",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.network = args[0]
			options.container = args[1]
			return runConnect(cmd.Context(), dockerCli, options)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completion.NetworkNames(dockerCli)(cmd, args, toComplete)
			}
			network := args[0]
			return completion.ContainerNames(dockerCli, true, not(isConnected(network)))(cmd, args, toComplete)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&options.ipaddress, "ip", "", `IPv4 address (e.g., "172.30.100.104")`)
	flags.StringVar(&options.ipv6address, "ip6", "", `IPv6 address (e.g., "2001:db8::33")`)
	flags.Var(&options.links, "link", "Add link to another container")
	flags.StringSliceVar(&options.aliases, "alias", []string{}, "Add network-scoped alias for the container")
	flags.StringSliceVar(&options.linklocalips, "link-local-ip", []string{}, "Add a link-local address for the container")
	flags.StringSliceVar(&options.driverOpts, "driver-opt", []string{}, "driver options for the network")
	return cmd
}

func runConnect(ctx context.Context, dockerCli command.Cli, options connectOptions) error {
	client := dockerCli.Client()

	driverOpts, err := convertDriverOpt(options.driverOpts)
	if err != nil {
		return err
	}
	epConfig := &network.EndpointSettings{
		IPAMConfig: &network.EndpointIPAMConfig{
			IPv4Address:  options.ipaddress,
			IPv6Address:  options.ipv6address,
			LinkLocalIPs: options.linklocalips,
		},
		Links:      options.links.GetAll(),
		Aliases:    options.aliases,
		DriverOpts: driverOpts,
	}

	return client.NetworkConnect(ctx, options.network, options.container, epConfig)
}

func convertDriverOpt(options []string) (map[string]string, error) {
	driverOpt := make(map[string]string)
	for _, opt := range options {
		k, v, ok := strings.Cut(opt, "=")
		// TODO(thaJeztah): we should probably not accept whitespace here (both for key and value).
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid key/value pair format in driver options")
		}
		driverOpt[k] = strings.TrimSpace(v)
	}
	return driverOpt, nil
}
