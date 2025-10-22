package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

const (
	defaultConfigTableFormat                     = "table {{.ID}}\t{{.Name}}\t{{.CreatedAt}}\t{{.UpdatedAt}}"
	configIDHeader                               = "ID"
	configCreatedHeader                          = "CREATED"
	configUpdatedHeader                          = "UPDATED"
	configInspectPrettyTemplate formatter.Format = `ID:			{{.ID}}
Name:			{{.Name}}
{{- if .Labels }}
Labels:
{{- range $k, $v := .Labels }}
 - {{ $k }}{{if $v }}={{ $v }}{{ end }}
{{- end }}{{ end }}
Created at:            	{{.CreatedAt}}
Updated at:            	{{.UpdatedAt}}
Data:
{{.Data}}`
)

// newFormat returns a Format for rendering using a configContext.
func newFormat(source string, quiet bool) formatter.Format {
	switch source {
	case formatter.PrettyFormatKey:
		return configInspectPrettyTemplate
	case formatter.TableFormatKey:
		if quiet {
			return formatter.DefaultQuietFormat
		}
		return defaultConfigTableFormat
	}
	return formatter.Format(source)
}

// formatWrite writes the context
func formatWrite(fmtCtx formatter.Context, configs client.ConfigListResult) error {
	cCtx := &configContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"ID":        configIDHeader,
				"Name":      formatter.NameHeader,
				"CreatedAt": configCreatedHeader,
				"UpdatedAt": configUpdatedHeader,
				"Labels":    formatter.LabelsHeader,
			},
		},
	}
	return fmtCtx.Write(cCtx, func(format func(subContext formatter.SubContext) error) error {
		for _, config := range configs.Items {
			configCtx := &configContext{c: config}
			if err := format(configCtx); err != nil {
				return err
			}
		}
		return nil
	})
}

type configContext struct {
	formatter.HeaderContext
	c swarm.Config
}

func (c *configContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *configContext) ID() string {
	return c.c.ID
}

func (c *configContext) Name() string {
	return c.c.Spec.Annotations.Name
}

func (c *configContext) CreatedAt() string {
	return units.HumanDuration(time.Now().UTC().Sub(c.c.Meta.CreatedAt)) + " ago"
}

func (c *configContext) UpdatedAt() string {
	return units.HumanDuration(time.Now().UTC().Sub(c.c.Meta.UpdatedAt)) + " ago"
}

func (c *configContext) Labels() string {
	mapLabels := c.c.Spec.Annotations.Labels
	if mapLabels == nil {
		return ""
	}
	joinLabels := make([]string, 0, len(mapLabels))
	for k, v := range mapLabels {
		joinLabels = append(joinLabels, k+"="+v)
	}
	return strings.Join(joinLabels, ",")
}

func (c *configContext) Label(name string) string {
	if c.c.Spec.Annotations.Labels == nil {
		return ""
	}
	return c.c.Spec.Annotations.Labels[name]
}

// inspectFormatWrite renders the context for a list of configs
func inspectFormatWrite(fmtCtx formatter.Context, refs []string, getRef inspect.GetRefFunc) error {
	if fmtCtx.Format != configInspectPrettyTemplate {
		return inspect.Inspect(fmtCtx.Output, refs, string(fmtCtx.Format), getRef)
	}
	return fmtCtx.Write(&configInspectContext{}, func(format func(subContext formatter.SubContext) error) error {
		for _, ref := range refs {
			configI, _, err := getRef(ref)
			if err != nil {
				return err
			}
			config, ok := configI.(swarm.Config)
			if !ok {
				return fmt.Errorf("got wrong object to inspect :%v", ok)
			}
			if err := format(&configInspectContext{Config: config}); err != nil {
				return err
			}
		}
		return nil
	})
}

type configInspectContext struct {
	swarm.Config
	formatter.SubContext
}

func (ctx *configInspectContext) ID() string {
	return ctx.Config.ID
}

func (ctx *configInspectContext) Name() string {
	return ctx.Config.Spec.Name
}

func (ctx *configInspectContext) Labels() map[string]string {
	return ctx.Config.Spec.Labels
}

func (ctx *configInspectContext) CreatedAt() string {
	return formatter.PrettyPrint(ctx.Config.CreatedAt)
}

func (ctx *configInspectContext) UpdatedAt() string {
	return formatter.PrettyPrint(ctx.Config.UpdatedAt)
}

func (ctx *configInspectContext) Data() string {
	if ctx.Config.Spec.Data == nil {
		return ""
	}
	return string(ctx.Config.Spec.Data)
}
