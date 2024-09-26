package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type createOptions struct {
	name       string
	scope      string
	driver     string
	driverOpts opts.MapOpts
	labels     opts.ListOpts
	internal   bool
	ipv6       *bool
	attachable bool
	ingress    bool
	configOnly bool
	configFrom string
	ipam       ipamOptions
}

type ipamOptions struct {
	driver       string
	subnets      []string
	ipRanges     []string
	gateways     []string
	auxAddresses opts.MapOpts
	driverOpts   opts.MapOpts
}

func newCreateCommand(dockerCLI command.Cli) *cobra.Command {
	var ipv6 bool
	options := createOptions{
		driverOpts: *opts.NewMapOpts(nil, nil),
		labels:     opts.NewListOpts(opts.ValidateLabel),
		ipam: ipamOptions{
			auxAddresses: *opts.NewMapOpts(nil, nil),
			driverOpts:   *opts.NewMapOpts(nil, nil),
		},
	}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] NETWORK",
		Short: "Create a network",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.name = args[0]

			if cmd.Flag("ipv6").Changed {
				options.ipv6 = &ipv6
			}

			return runCreate(cmd.Context(), dockerCLI.Client(), dockerCLI.Out(), options)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.driver, "driver", "d", "bridge", "Driver to manage the Network")
	flags.VarP(&options.driverOpts, "opt", "o", "Set driver specific options")
	flags.Var(&options.labels, "label", "Set metadata on a network")
	flags.BoolVar(&options.internal, "internal", false, "Restrict external access to the network")
	flags.BoolVar(&ipv6, "ipv6", false, "Enable or disable IPv6 networking")
	flags.BoolVar(&options.attachable, "attachable", false, "Enable manual container attachment")
	flags.SetAnnotation("attachable", "version", []string{"1.25"})
	flags.BoolVar(&options.ingress, "ingress", false, "Create swarm routing-mesh network")
	flags.SetAnnotation("ingress", "version", []string{"1.29"})
	flags.StringVar(&options.scope, "scope", "", "Control the network's scope")
	flags.SetAnnotation("scope", "version", []string{"1.30"})
	flags.BoolVar(&options.configOnly, "config-only", false, "Create a configuration only network")
	flags.SetAnnotation("config-only", "version", []string{"1.30"})
	flags.StringVar(&options.configFrom, "config-from", "", "The network from which to copy the configuration")
	flags.SetAnnotation("config-from", "version", []string{"1.30"})

	flags.StringVar(&options.ipam.driver, "ipam-driver", "default", "IP Address Management Driver")
	flags.StringSliceVar(&options.ipam.subnets, "subnet", []string{}, "Subnet in CIDR format that represents a network segment")
	flags.StringSliceVar(&options.ipam.ipRanges, "ip-range", []string{}, "Allocate container ip from a sub-range")
	flags.StringSliceVar(&options.ipam.gateways, "gateway", []string{}, "IPv4 or IPv6 Gateway for the master subnet")

	flags.Var(&options.ipam.auxAddresses, "aux-address", "Auxiliary IPv4 or IPv6 addresses used by Network driver")
	flags.Var(&options.ipam.driverOpts, "ipam-opt", "Set IPAM driver specific options")

	return cmd
}

func runCreate(ctx context.Context, apiClient client.NetworkAPIClient, output io.Writer, options createOptions) error {
	ipamCfg, err := createIPAMConfig(options.ipam)
	if err != nil {
		return err
	}

	var configFrom *network.ConfigReference
	if options.configFrom != "" {
		configFrom = &network.ConfigReference{
			Network: options.configFrom,
		}
	}
	resp, err := apiClient.NetworkCreate(ctx, options.name, network.CreateOptions{
		Driver:     options.driver,
		Options:    options.driverOpts.GetAll(),
		IPAM:       ipamCfg,
		Internal:   options.internal,
		EnableIPv6: options.ipv6,
		Attachable: options.attachable,
		Ingress:    options.ingress,
		Scope:      options.scope,
		ConfigOnly: options.configOnly,
		ConfigFrom: configFrom,
		Labels:     opts.ConvertKVStringsToMap(options.labels.GetAll()),
	})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(output, "%s\n", resp.ID)
	return nil
}

// Consolidates the ipam configuration as a group from different related configurations
// user can configure network with multiple non-overlapping subnets and hence it is
// possible to correlate the various related parameters and consolidate them.
// createIPAMConfig consolidates subnets, ip-ranges, gateways and auxiliary addresses into
// structured ipam data.
//
//nolint:gocyclo
func createIPAMConfig(options ipamOptions) (*network.IPAM, error) {
	if len(options.subnets) < len(options.ipRanges) || len(options.subnets) < len(options.gateways) {
		return nil, errors.Errorf("every ip-range or gateway must have a corresponding subnet")
	}
	iData := map[string]*network.IPAMConfig{}

	// Populate non-overlapping subnets into consolidation map
	for _, s := range options.subnets {
		for k := range iData {
			ok1, err := subnetMatches(s, k)
			if err != nil {
				return nil, err
			}
			ok2, err := subnetMatches(k, s)
			if err != nil {
				return nil, err
			}
			if ok1 || ok2 {
				return nil, errors.Errorf("multiple overlapping subnet configuration is not supported")
			}
		}
		iData[s] = &network.IPAMConfig{Subnet: s, AuxAddress: map[string]string{}}
	}

	// Validate and add valid ip ranges
	for _, r := range options.ipRanges {
		match := false
		for _, s := range options.subnets {
			if _, _, err := net.ParseCIDR(r); err != nil {
				return nil, err
			}
			ok, err := subnetMatches(s, r)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			if iData[s].IPRange != "" {
				return nil, errors.Errorf("cannot configure multiple ranges (%s, %s) on the same subnet (%s)", r, iData[s].IPRange, s)
			}
			d := iData[s]
			d.IPRange = r
			match = true
		}
		if !match {
			return nil, errors.Errorf("no matching subnet for range %s", r)
		}
	}

	// Validate and add valid gateways
	for _, g := range options.gateways {
		match := false
		for _, s := range options.subnets {
			ok, err := subnetMatches(s, g)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			if iData[s].Gateway != "" {
				return nil, errors.Errorf("cannot configure multiple gateways (%s, %s) for the same subnet (%s)", g, iData[s].Gateway, s)
			}
			d := iData[s]
			d.Gateway = g
			match = true
		}
		if !match {
			return nil, errors.Errorf("no matching subnet for gateway %s", g)
		}
	}

	// Validate and add aux-addresses
	for key, aa := range options.auxAddresses.GetAll() {
		match := false
		for _, s := range options.subnets {
			ok, err := subnetMatches(s, aa)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			iData[s].AuxAddress[key] = aa
			match = true
		}
		if !match {
			return nil, errors.Errorf("no matching subnet for aux-address %s", aa)
		}
	}

	idl := make([]network.IPAMConfig, 0, len(iData))
	for _, v := range iData {
		idl = append(idl, *v)
	}

	return &network.IPAM{
		Driver:  options.driver,
		Config:  idl,
		Options: options.driverOpts.GetAll(),
	}, nil
}

func subnetMatches(subnet, data string) (bool, error) {
	var ip net.IP

	_, s, err := net.ParseCIDR(subnet)
	if err != nil {
		return false, errors.Wrap(err, "invalid subnet")
	}

	if strings.Contains(data, "/") {
		ip, _, err = net.ParseCIDR(data)
		if err != nil {
			return false, err
		}
	} else {
		ip = net.ParseIP(data)
	}

	return s.Contains(ip), nil
}
