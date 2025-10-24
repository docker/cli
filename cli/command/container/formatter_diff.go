package container

import (
	"github.com/docker/cli/cli/command/formatter"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const (
	defaultDiffTableFormat = "table {{.Type}}\t{{.Path}}"

	changeTypeHeader = "CHANGE TYPE"
	pathHeader       = "PATH"
)

// newDiffFormat returns a format for use with a diff [formatter.Context].
func newDiffFormat(source string) formatter.Format {
	if source == formatter.TableFormatKey {
		return defaultDiffTableFormat
	}
	return formatter.Format(source)
}

// diffFormatWrite writes formatted diff using the [formatter.Context].
func diffFormatWrite(fmtCtx formatter.Context, changes client.ContainerDiffResult) error {
	return fmtCtx.Write(newDiffContext(), func(format func(subContext formatter.SubContext) error) error {
		for _, change := range changes.Changes {
			if err := format(&diffContext{c: change}); err != nil {
				return err
			}
		}
		return nil
	})
}

type diffContext struct {
	formatter.HeaderContext
	c container.FilesystemChange
}

func newDiffContext() *diffContext {
	return &diffContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"Type": changeTypeHeader,
				"Path": pathHeader,
			},
		},
	}
}

func (d *diffContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(d)
}

func (d *diffContext) Type() string {
	return d.c.Kind.String()
}

func (d *diffContext) Path() string {
	return d.c.Path
}
