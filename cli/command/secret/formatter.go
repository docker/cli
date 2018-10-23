package secret

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/docker/docker/api/types/swarm"
	units "github.com/docker/go-units"
)

const (
	defaultSecretTableFormat                     = "table {{.ID}}\t{{.Name}}\t{{.Driver}}\t{{.CreatedAt}}\t{{.UpdatedAt}}"
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

// NewFormat returns a Format for rendering using a secret Context
func NewFormat(source string, quiet bool) formatter.Format {
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

// FormatWrite writes the context
func FormatWrite(ctx formatter.Context, secrets []swarm.Secret) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, secret := range secrets {
			secretCtx := &secretContext{s: secret}
			if err := format(secretCtx); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newSecretContext(), render)
}

func newSecretContext() *secretContext {
	sCtx := &secretContext{}

	sCtx.Header = formatter.SubHeaderContext{
		"ID":        secretIDHeader,
		"Name":      formatter.NameHeader,
		"Driver":    formatter.DriverHeader,
		"CreatedAt": secretCreatedHeader,
		"UpdatedAt": secretUpdatedHeader,
		"Labels":    formatter.LabelsHeader,
	}
	return sCtx
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
	var joinLabels []string
	for k, v := range mapLabels {
		joinLabels = append(joinLabels, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(joinLabels, ",")
}

func (c *secretContext) Label(name string) string {
	if c.s.Spec.Annotations.Labels == nil {
		return ""
	}
	return c.s.Spec.Annotations.Labels[name]
}

// InspectFormatWrite renders the context for a list of secrets
func InspectFormatWrite(ctx formatter.Context, refs []string, getRef inspect.GetRefFunc) error {
	if ctx.Format != secretInspectPrettyTemplate {
		return inspect.Inspect(ctx.Output, refs, string(ctx.Format), getRef)
	}
	render := func(format func(subContext formatter.SubContext) error) error {
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
	}
	return ctx.Write(&secretInspectContext{}, render)
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
	return command.PrettyPrint(ctx.Secret.CreatedAt)
}

func (ctx *secretInspectContext) UpdatedAt() string {
	return command.PrettyPrint(ctx.Secret.UpdatedAt)
}
