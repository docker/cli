package container

import (
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/archive"
)

const (
	defaultDiffTableFormat = "table {{.Type}}\t{{.Path}}"

	changeTypeHeader = "CHANGE TYPE"
	pathHeader       = "PATH"
)

// NewDiffFormat returns a format for use with a diff Context
func NewDiffFormat(source string) formatter.Format {
	switch source {
	case formatter.TableFormatKey:
		return defaultDiffTableFormat
	}
	return formatter.Format(source)
}

// DiffFormatWrite writes formatted diff using the Context
func DiffFormatWrite(ctx formatter.Context, changes []container.ContainerChangeResponseItem) error {

	render := func(format func(subContext formatter.SubContext) error) error {
		for _, change := range changes {
			if err := format(&diffContext{c: change}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newDiffContext(), render)
}

type diffContext struct {
	formatter.HeaderContext
	c container.ContainerChangeResponseItem
}

func newDiffContext() *diffContext {
	diffCtx := diffContext{}
	diffCtx.Header = formatter.SubHeaderContext{
		"Type": changeTypeHeader,
		"Path": pathHeader,
	}
	return &diffCtx
}

func (d *diffContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(d)
}

func (d *diffContext) Type() string {
	var kind string
	switch d.c.Kind {
	case archive.ChangeModify:
		kind = "C"
	case archive.ChangeAdd:
		kind = "A"
	case archive.ChangeDelete:
		kind = "D"
	}
	return kind

}

func (d *diffContext) Path() string {
	return d.c.Path
}
