package network

import (
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
)

const (
	defaultNetworkTableFormat = "table {{.ID}}\t{{.Name}}\t{{.Driver}}\t{{.Scope}}"

	networkIDHeader = "NETWORK ID"
	ipv6Header      = "IPV6"
	internalHeader  = "INTERNAL"
)

// NewFormat returns a Format for rendering using a network Context
func NewFormat(source string, quiet bool) formatter.Format {
	switch source {
	case formatter.TableFormatKey:
		if quiet {
			return formatter.DefaultQuietFormat
		}
		return defaultNetworkTableFormat
	case formatter.RawFormatKey:
		if quiet {
			return `network_id: {{.ID}}`
		}
		return `network_id: {{.ID}}\nname: {{.Name}}\ndriver: {{.Driver}}\nscope: {{.Scope}}\n`
	}
	return formatter.Format(source)
}

// FormatWrite writes the context
func FormatWrite(ctx formatter.Context, networks []types.NetworkResource) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, network := range networks {
			networkCtx := &networkContext{trunc: ctx.Trunc, n: network}
			if err := format(networkCtx); err != nil {
				return err
			}
		}
		return nil
	}
	networkCtx := networkContext{}
	networkCtx.Header = formatter.SubHeaderContext{
		"ID":        networkIDHeader,
		"Name":      formatter.NameHeader,
		"Driver":    formatter.DriverHeader,
		"Scope":     formatter.ScopeHeader,
		"IPv6":      ipv6Header,
		"Internal":  internalHeader,
		"Labels":    formatter.LabelsHeader,
		"CreatedAt": formatter.CreatedAtHeader,
	}
	return ctx.Write(&networkCtx, render)
}

type networkContext struct {
	formatter.HeaderContext
	trunc bool
	n     types.NetworkResource
}

func (c *networkContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *networkContext) ID() string {
	if c.trunc {
		return stringid.TruncateID(c.n.ID)
	}
	return c.n.ID
}

func (c *networkContext) Name() string {
	return c.n.Name
}

func (c *networkContext) Driver() string {
	return c.n.Driver
}

func (c *networkContext) Scope() string {
	return c.n.Scope
}

func (c *networkContext) IPv6() string {
	return fmt.Sprintf("%v", c.n.EnableIPv6)
}

func (c *networkContext) Internal() string {
	return fmt.Sprintf("%v", c.n.Internal)
}

func (c *networkContext) Labels() string {
	if c.n.Labels == nil {
		return ""
	}

	var joinLabels []string
	for k, v := range c.n.Labels {
		joinLabels = append(joinLabels, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(joinLabels, ",")
}

func (c *networkContext) Label(name string) string {
	if c.n.Labels == nil {
		return ""
	}
	return c.n.Labels[name]
}

func (c *networkContext) CreatedAt() string {
	return c.n.Created.String()
}
