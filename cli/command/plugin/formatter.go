package plugin

import (
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/moby/moby/api/types/plugin"
	"github.com/moby/moby/client"
)

const (
	defaultPluginTableFormat = "table {{.ID}}\t{{.Name}}\t{{.Description}}\t{{.Enabled}}"

	enabledHeader  = "ENABLED"
	pluginIDHeader = "ID"

	rawFormat = `plugin_id: {{.ID}}
name: {{.Name}}
description: {{.Description}}
enabled: {{.Enabled}}
`
)

// newFormat returns a Format for rendering using a pluginContext.
func newFormat(source string, quiet bool) formatter.Format {
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
		return rawFormat
	}
	return formatter.Format(source)
}

// formatWrite writes the context
func formatWrite(fmtCtx formatter.Context, plugins client.PluginListResult) error {
	pluginCtx := &pluginContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"ID":              pluginIDHeader,
				"Name":            formatter.NameHeader,
				"Description":     formatter.DescriptionHeader,
				"Enabled":         enabledHeader,
				"PluginReference": formatter.ImageHeader,
			},
		},
	}
	return fmtCtx.Write(pluginCtx, func(format func(subContext formatter.SubContext) error) error {
		for _, p := range plugins.Items {
			if err := format(&pluginContext{
				trunc: fmtCtx.Trunc,
				p:     p,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

type pluginContext struct {
	formatter.HeaderContext
	trunc bool
	p     plugin.Plugin
}

func (c *pluginContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *pluginContext) ID() string {
	if c.trunc {
		return formatter.TruncateID(c.p.ID)
	}
	return c.p.ID
}

func (c *pluginContext) Name() string {
	return c.p.Name
}

func (c *pluginContext) Description() string {
	desc := strings.ReplaceAll(c.p.Config.Description, "\n", "")
	desc = strings.ReplaceAll(desc, "\r", "")
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
