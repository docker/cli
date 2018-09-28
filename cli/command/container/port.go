package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"
)

type portOptions struct {
	container string

	port string
}

// NewPortCommand creates a new cobra.Command for `docker port`
func NewPortCommand(dockerCli command.Cli) *cobra.Command {
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
			return runPort(dockerCli, &opts)
		},
	}
	return cmd
}

func runPort(dockerCli command.Cli, opts *portOptions) error {
	ctx := context.Background()

	c, err := dockerCli.Client().ContainerInspect(ctx, opts.container)
	if err != nil {
		return err
	}

	if opts.port != "" {
		port := opts.port
		proto := "tcp"
		parts := strings.SplitN(port, "/", 2)

		if len(parts) == 2 && len(parts[1]) != 0 {
			port = parts[0]
			proto = parts[1]
		}
		newP, err := nat.NewPort(proto, port)
		if err != nil {
			return err
		}
		if frontends, exists := c.NetworkSettings.Ports[newP]; exists && frontends != nil {
			for _, frontend := range frontends {
				fmt.Fprintf(dockerCli.Out(), "%s:%s\n", frontend.HostIP, frontend.HostPort)
			}
			return nil
		}
		portStart, portEnd, _ := nat.ParsePortRange(port)
		if portEnd > portStart {
			for nport := portStart; nport <= portEnd; nport++ {
				nowP, _ := nat.NewPort(proto, fmt.Sprint(nport))
				if frontends, exists := c.NetworkSettings.Ports[nowP]; exists && frontends != nil {
					for _, frontend := range frontends {
						fmt.Fprintf(dockerCli.Out(), "%s -> %s:%s\n", nowP, frontend.HostIP, frontend.HostPort)
					}
				} else {
					fmt.Fprintf(dockerCli.Out(), "Error: No public port '%s' published for %s\n", nowP, opts.container)
				}
			}
		}
		return nil
	}

	for from, frontends := range c.NetworkSettings.Ports {
		for _, frontend := range frontends {
			fmt.Fprintf(dockerCli.Out(), "%s -> %s:%s\n", from, frontend.HostIP, frontend.HostPort)
		}
	}

	return nil
}
