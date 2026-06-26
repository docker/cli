package checkpoint

import (
	"github.com/docker/cli/cli/command/formatter"
	"github.com/moby/moby/api/types/checkpoint"
)

const (
	defaultCheckpointFormat = "table {{.Name}}"
	checkpointNameHeader    = "CHECKPOINT NAME"
)

// newFormat returns a format for use with a checkpointContext.
func newFormat(source string) formatter.Format {
	if source == formatter.TableFormatKey {
		return defaultCheckpointFormat
	}
	return formatter.Format(source)
}

// formatWrite writes formatted checkpoints using the Context
func formatWrite(fmtCtx formatter.Context, checkpoints []checkpoint.Summary) error {
	cpContext := &checkpointContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"Name": checkpointNameHeader,
			},
		},
	}
	return fmtCtx.Write(cpContext, func(format func(subContext formatter.SubContext) error) error {
		for _, cp := range checkpoints {
			if err := format(&checkpointContext{c: cp}); err != nil {
				return err
			}
		}
		return nil
	})
}

type checkpointContext struct {
	formatter.HeaderContext
	c checkpoint.Summary
}

func (c *checkpointContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *checkpointContext) Name() string {
	return c.c.Name
}
