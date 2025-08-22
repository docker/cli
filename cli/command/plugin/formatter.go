package plugin

import (
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/moby/moby/api/types/plugin"
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

// NewFormat returns a Format for rendering using a plugin Context
//
// Deprecated: this function was only used internally and will be removed in the next release.
func NewFormat(source string, quiet bool) formatter.Format {
	return newFormat(source, quiet)
}

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

// FormatWrite writes the context
//
// Deprecated: this function was only used internally and will be removed in the next release.
func FormatWrite(fmtCtx formatter.Context, plugins []*plugin.Plugin) error {
	return formatWrite(fmtCtx, plugins)
}

// formatWrite writes the context
func formatWrite(fmtCtx formatter.Context, plugins []*plugin.Plugin) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, p := range plugins {
			pluginCtx := &pluginContext{trunc: fmtCtx.Trunc, p: *p}
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
	return fmtCtx.Write(&pluginCtx, render)
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
