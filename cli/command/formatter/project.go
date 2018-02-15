package formatter

import "github.com/docker/cli/project"

const (
	defaultProjectQuietFormat = "{{.RootDir}}"
	defaultProjectTableFormat = "table {{.RootDir}}\t{{.ID}}"
	projectRootDirHeader      = "ROOT DIRECTORY"
	projectIDHeader           = "ID"
)

// NewProjectFormat returns a format for use with a project Context
func NewProjectFormat(source string, quiet bool) Format {
	switch source {
	case TableFormatKey:
		if quiet {
			return defaultProjectQuietFormat
		}
		return defaultProjectTableFormat
	}
	return Format(source)
}

// ProjectWrite writes formatted projects using the Context
func ProjectWrite(ctx Context, projects []project.Project) error {
	render := func(format func(subContext subContext) error) error {
		for _, p := range projects {
			if err := format(&projectContext{v: p}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newProjectContext(), render)
}

type projectHeaderContext map[string]string

type projectContext struct {
	HeaderContext
	v project.Project
}

func newProjectContext() *projectContext {
	projectCtx := projectContext{}
	projectCtx.header = projectHeaderContext{
		"RootDir": projectRootDirHeader,
		"ID":      projectIDHeader,
	}
	return &projectCtx
}

func (c *projectContext) MarshalJSON() ([]byte, error) {
	return marshalJSON(c)
}

func (c *projectContext) RootDir() string {
	return c.v.RootDir()
}

func (c *projectContext) ID() string {
	return c.v.ID()
}
