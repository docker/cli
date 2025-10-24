package secret

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
	defaultSecretTableFormat                     = "table {{.ID}}\t{{.Name}}\t{{.Driver}}\t{{.CreatedAt}}\t{{.UpdatedAt}}" // #nosec G101
	secretIDHeader                               = "ID"
	secretCreatedHeader                          = "CREATED"
	secretUpdatedHeader                          = "UPDATED"
	secretInspectPrettyTemplate formatter.Format = `ID:              {{.ID}}
Name:              {{.Name}}
{{- if .Labels }}
Labels:
{{- range $k, $v := .Labels }}
 - {{ $k }}{{if $v }}={{ $v }}{{ end }}
{{- end }}{{ end }}
Driver:            {{.Driver}}
Created at:        {{.CreatedAt}}
Updated at:        {{.UpdatedAt}}`
)

// newFormat returns a Format for rendering using a secretContext.
func newFormat(source string, quiet bool) formatter.Format {
	switch source {
	case formatter.PrettyFormatKey:
		return secretInspectPrettyTemplate
	case formatter.TableFormatKey:
		if quiet {
			return formatter.DefaultQuietFormat
		}
		return defaultSecretTableFormat
	}
	return formatter.Format(source)
}

// formatWrite writes the context
func formatWrite(fmtCtx formatter.Context, secrets client.SecretListResult) error {
	sCtx := &secretContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"ID":        secretIDHeader,
				"Name":      formatter.NameHeader,
				"Driver":    formatter.DriverHeader,
				"CreatedAt": secretCreatedHeader,
				"UpdatedAt": secretUpdatedHeader,
				"Labels":    formatter.LabelsHeader,
			},
		},
	}
	return fmtCtx.Write(sCtx, func(format func(subContext formatter.SubContext) error) error {
		for _, secret := range secrets.Items {
			secretCtx := &secretContext{s: secret}
			if err := format(secretCtx); err != nil {
				return err
			}
		}
		return nil
	})
}

type secretContext struct {
	formatter.HeaderContext
	s swarm.Secret
}

func (c *secretContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *secretContext) ID() string {
	return c.s.ID
}

func (c *secretContext) Name() string {
	return c.s.Spec.Annotations.Name
}

func (c *secretContext) CreatedAt() string {
	return units.HumanDuration(time.Now().UTC().Sub(c.s.Meta.CreatedAt)) + " ago"
}

func (c *secretContext) Driver() string {
	if c.s.Spec.Driver == nil {
		return ""
	}
	return c.s.Spec.Driver.Name
}

func (c *secretContext) UpdatedAt() string {
	return units.HumanDuration(time.Now().UTC().Sub(c.s.Meta.UpdatedAt)) + " ago"
}

func (c *secretContext) Labels() string {
	mapLabels := c.s.Spec.Annotations.Labels
	if mapLabels == nil {
		return ""
	}
	joinLabels := make([]string, 0, len(mapLabels))
	for k, v := range mapLabels {
		joinLabels = append(joinLabels, k+"="+v)
	}
	return strings.Join(joinLabels, ",")
}

func (c *secretContext) Label(name string) string {
	if c.s.Spec.Annotations.Labels == nil {
		return ""
	}
	return c.s.Spec.Annotations.Labels[name]
}

// inspectFormatWrite renders the context for a list of secrets.
func inspectFormatWrite(fmtCtx formatter.Context, refs []string, getRef inspect.GetRefFunc) error {
	if fmtCtx.Format != secretInspectPrettyTemplate {
		return inspect.Inspect(fmtCtx.Output, refs, string(fmtCtx.Format), getRef)
	}
	return fmtCtx.Write(&secretInspectContext{}, func(format func(subContext formatter.SubContext) error) error {
		for _, ref := range refs {
			secretI, _, err := getRef(ref)
			if err != nil {
				return err
			}
			secret, ok := secretI.(swarm.Secret)
			if !ok {
				return fmt.Errorf("got wrong object to inspect :%v", ok)
			}
			if err := format(&secretInspectContext{Secret: secret}); err != nil {
				return err
			}
		}
		return nil
	})
}

type secretInspectContext struct {
	swarm.Secret
	formatter.SubContext
}

func (ctx *secretInspectContext) ID() string {
	return ctx.Secret.ID
}

func (ctx *secretInspectContext) Name() string {
	return ctx.Secret.Spec.Name
}

func (ctx *secretInspectContext) Labels() map[string]string {
	return ctx.Secret.Spec.Labels
}

func (ctx *secretInspectContext) Driver() string {
	if ctx.Secret.Spec.Driver == nil {
		return ""
	}
	return ctx.Secret.Spec.Driver.Name
}

func (ctx *secretInspectContext) CreatedAt() string {
	return formatter.PrettyPrint(ctx.Secret.CreatedAt)
}

func (ctx *secretInspectContext) UpdatedAt() string {
	return formatter.PrettyPrint(ctx.Secret.UpdatedAt)
}
