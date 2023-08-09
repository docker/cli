package checkpoint

import (
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types"
)

const (
	defaultCheckpointFormat = "table {{.Name}}"

	checkpointNameHeader = "CHECKPOINT NAME"
)

// NewFormat returns a format for use with a checkpoint Context
func NewFormat(source string) formatter.Format {
	switch source {
	case formatter.TableFormatKey:
		return defaultCheckpointFormat
	}
	return formatter.Format(source)
}

// FormatWrite writes formatted checkpoints using the Context
func FormatWrite(ctx formatter.Context, checkpoints []types.Checkpoint) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, checkpoint := range checkpoints {
			if err := format(&checkpointContext{c: checkpoint}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newCheckpointContext(), render)
}

type checkpointContext struct {
	formatter.HeaderContext
	c types.Checkpoint
}

// newCheckpointContext initializes a new checkpoint context with default headers.
func newCheckpointContext() *checkpointContext {
	cpCtx := checkpointContext{}
	cpCtx.Header = formatter.SubHeaderContext{
		"Name": checkpointNameHeader,
	}
	return &cpCtx
}

// MarshalJSON converts the checkpoint context to JSON format.
func (c *checkpointContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

// Name returns the name of the checkpoint from the context.
func (c *checkpointContext) Name() string {
	return c.c.Name
}
