package plugin

import (
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
)

const (
	defaultPluginTableFormat = "table {{.ID}}\t{{.Name}}\t{{.Description}}\t{{.Enabled}}"

	enabledHeader  = "ENABLED"
	pluginIDHeader = "ID"
)

// NewFormat returns a Format for rendering using a plugin Context
func NewFormat(source string, quiet bool) formatter.Format {
	switch source {
	case formatter.TableFormatKey:
		if quiet {
			return formatter.DefaultQuietFormat
		}
		return defaultPluginTableFormat
	case formatter.RawFormatKey:
		if quiet {
			return `plugin_id: {{.ID}}`
		}
		return `plugin_id: {{.ID}}\nname: {{.Name}}\ndescription: {{.Description}}\nenabled: {{.Enabled}}\n`
	}
	return formatter.Format(source)
}

// FormatWrite writes the context
func FormatWrite(ctx formatter.Context, plugins []*types.Plugin) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, plugin := range plugins {
			pluginCtx := &pluginContext{trunc: ctx.Trunc, p: *plugin}
			if err := format(pluginCtx); err != nil {
				return err
			}
		}
		return nil
	}
	pluginCtx := pluginContext{}
	pluginCtx.Header = formatter.SubHeaderContext{
		"ID":              pluginIDHeader,
		"Name":            formatter.NameHeader,
		"Description":     formatter.DescriptionHeader,
		"Enabled":         enabledHeader,
		"PluginReference": formatter.ImageHeader,
	}
	return ctx.Write(&pluginCtx, render)
}

type pluginContext struct {
	formatter.HeaderContext
	trunc bool
	p     types.Plugin
}

func (c *pluginContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *pluginContext) ID() string {
	if c.trunc {
		return stringid.TruncateID(c.p.ID)
	}
	return c.p.ID
}

func (c *pluginContext) Name() string {
	return c.p.Name
}

func (c *pluginContext) Description() string {
	desc := strings.Replace(c.p.Config.Description, "\n", "", -1)
	desc = strings.Replace(desc, "\r", "", -1)
	if c.trunc {
		desc = formatter.Ellipsis(desc, 45)
	}

	return desc
}

func (c *pluginContext) Enabled() bool {
	return c.p.Enabled
}

func (c *pluginContext) PluginReference() string {
	return c.p.PluginReference
}
