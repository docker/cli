package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type createOptions struct {
	name       string
	scope      string
	driver     string
	driverOpts opts.MapOpts
	labels     opts.ListOpts
	internal   bool
	ipv4       *bool
	ipv6       *bool
	attachable bool
	ingress    bool
	configOnly bool
	configFrom string
	ipam       ipamOptions
}

type ipamOptions struct {
	driver       string
	subnets      []string    // TODO(thaJeztah): change to []net.IPNet? This won't accept a bare address (without "/xxx"); we need a flag-type to handle []netip.Prefix directly
	ipRanges     []net.IPNet // TODO(thaJeztah): we need a flag-type to handle []netip.Prefix directly
	gateways     []net.IP    // TODO(thaJeztah): we need a flag-type to handle []netip.Addr directly
	auxAddresses opts.MapOpts
	driverOpts   opts.MapOpts
}

func newCreateCommand(dockerCLI command.Cli) *cobra.Command {
	var ipv4, ipv6 bool
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

			if cmd.Flag("ipv4").Changed {
				options.ipv4 = &ipv4
			}
			if cmd.Flag("ipv6").Changed {
				options.ipv6 = &ipv6
			}

			return runCreate(cmd.Context(), dockerCLI.Client(), dockerCLI.Out(), options)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.driver, "driver", "d", "bridge", "Driver to manage the Network")
	flags.VarP(&options.driverOpts, "opt", "o", "Set driver specific options")
	flags.Var(&options.labels, "label", "Set metadata on a network")
	flags.BoolVar(&options.internal, "internal", false, "Restrict external access to the network")
	flags.BoolVar(&ipv4, "ipv4", true, "Enable or disable IPv4 address assignment")
	flags.BoolVar(&ipv6, "ipv6", false, "Enable or disable IPv6 address assignment")
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
	flags.IPNetSliceVar(&options.ipam.ipRanges, "ip-range", nil, "Allocate container ip from a sub-range")
	flags.IPSliceVar(&options.ipam.gateways, "gateway", nil, "IPv4 or IPv6 Gateway for the master subnet")

	flags.Var(&options.ipam.auxAddresses, "aux-address", "Auxiliary IPv4 or IPv6 addresses used by Network driver")
	flags.Var(&options.ipam.driverOpts, "ipam-opt", "Set IPAM driver specific options")

	return cmd
}

func runCreate(ctx context.Context, apiClient client.NetworkAPIClient, output io.Writer, options createOptions) error {
	ipamCfg, err := createIPAMConfig(options.ipam)
	if err != nil {
		return err
	}

	resp, err := apiClient.NetworkCreate(ctx, options.name, client.NetworkCreateOptions{
		Driver:     options.driver,
		Options:    options.driverOpts.GetAll(),
		IPAM:       ipamCfg,
		Internal:   options.internal,
		EnableIPv4: options.ipv4,
		EnableIPv6: options.ipv6,
		Attachable: options.attachable,
		Ingress:    options.ingress,
		Scope:      options.scope,
		ConfigOnly: options.configOnly,
		ConfigFrom: options.configFrom,
		Labels:     opts.ConvertKVStringsToMap(options.labels.GetSlice()),
	})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(output, resp.ID)
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
		return nil, errors.New("every ip-range or gateway must have a corresponding subnet")
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
				return nil, errors.New("multiple overlapping subnet configuration is not supported")
			}
		}
		sn, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, err
		}
		iData[s] = &network.IPAMConfig{Subnet: sn, AuxAddress: map[string]netip.Addr{}}
	}

	// Validate and add valid ip ranges
	for _, r := range options.ipRanges {
		match := false
		for _, s := range options.subnets {
			ok, err := subnetMatches(s, r.String())
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}

			// Using "IsValid" to check if a valid IPRange was already set.
			if iData[s].IPRange.IsValid() {
				return nil, fmt.Errorf("cannot configure multiple ranges (%s, %s) on the same subnet (%s)", r.String(), iData[s].IPRange.String(), s)
			}
			if ipRange, ok := toPrefix(r); ok {
				iData[s].IPRange = ipRange
				match = true
			}
		}
		if !match {
			return nil, fmt.Errorf("no matching subnet for range %s", r.String())
		}
	}

	// Validate and add valid gateways
	for _, g := range options.gateways {
		match := false
		for _, s := range options.subnets {
			ok, err := subnetMatches(s, g.String())
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			if iData[s].Gateway.IsValid() {
				return nil, fmt.Errorf("cannot configure multiple gateways (%s, %s) for the same subnet (%s)", g, iData[s].Gateway, s)
			}
			d := iData[s]
			d.Gateway = toNetipAddr(g)
			match = true
		}
		if !match {
			return nil, fmt.Errorf("no matching subnet for gateway %s", g)
		}
	}

	// Validate and add aux-addresses
	for name, aa := range options.auxAddresses.GetAll() {
		if aa == "" {
			continue
		}
		auxAddr, err := netip.ParseAddr(aa)
		if err != nil {
			return nil, err
		}
		match := false
		for _, s := range options.subnets {
			ok, err := subnetMatches(s, auxAddr.String())
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			iData[s].AuxAddress[name] = auxAddr
			match = true
		}
		if !match {
			return nil, fmt.Errorf("no matching subnet for aux-address %s", aa)
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
		return false, fmt.Errorf("invalid subnet: %w", err)
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
