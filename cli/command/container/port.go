package container

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/fvbommel/sortorder"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type portOptions struct {
	container string

	port string
}

// newPortCommand creates a new cobra.Command for "docker container port".
func newPortCommand(dockerCLI command.Cli) *cobra.Command {
	var opts portOptions

	cmd := &cobra.Command{
		Use:   "port CONTAINER [PRIVATE_PORT[/PROTO]]",
		Short: "List port mappings or a specific mapping for the container",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.container = args[0]
			if len(args) > 1 {
				opts.port = args[1]
			}
			return runPort(cmd.Context(), dockerCLI, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container port, docker port",
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, false),
		DisableFlagsInUseLine: true,
	}
	return cmd
}

// runPort shows the port mapping for a given container. Optionally, it
// allows showing the mapping for a specific (container)port and proto.
//
// TODO(thaJeztah): currently this defaults to show the TCP port if no
// proto is specified. We should consider changing this to "any" protocol
// for the given private port.
func runPort(ctx context.Context, dockerCli command.Cli, opts *portOptions) error {
	c, err := dockerCli.Client().ContainerInspect(ctx, opts.container, client.ContainerInspectOptions{})
	if err != nil {
		return err
	}

	var out []string
	if opts.port != "" {
		port, err := network.ParsePort(opts.port)
		if err != nil {
			return err
		}
		frontends, exists := c.Container.NetworkSettings.Ports[port]
		if !exists || len(frontends) == 0 {
			return fmt.Errorf("no public port '%s' published for %s", opts.port, opts.container)
		}
		for _, frontend := range frontends {
			out = append(out, net.JoinHostPort(frontend.HostIP.String(), frontend.HostPort))
		}
	} else {
		for from, frontends := range c.Container.NetworkSettings.Ports {
			for _, frontend := range frontends {
				out = append(out, fmt.Sprintf("%s -> %s", from, net.JoinHostPort(frontend.HostIP.String(), frontend.HostPort)))
			}
		}
	}

	if len(out) > 0 {
		sort.Slice(out, func(i, j int) bool {
			return sortorder.NaturalLess(out[i], out[j])
		})
		_, _ = fmt.Fprintln(dockerCli.Out(), strings.Join(out, "\n"))
	}

	return nil
}
