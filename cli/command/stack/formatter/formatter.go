package formatter

import (
	"strconv"

	"github.com/docker/cli/cli/command/formatter"
)

const (
	// SwarmStackTableFormat is the default Swarm stack format
	//
	// Deprecated: this type was for internal use and will be removed in the next release.
	SwarmStackTableFormat formatter.Format = "table {{.Name}}\t{{.Services}}"

	stackServicesHeader = "SERVICES"

	// TableFormatKey is an alias for formatter.TableFormatKey
	//
	// Deprecated: this type was for internal use and will be removed in the next release.
	TableFormatKey = formatter.TableFormatKey
)

// Context is an alias for formatter.Context
//
// Deprecated: this type was for internal use and will be removed in the next release.
type Context = formatter.Context

// Format is an alias for formatter.Format
//
// Deprecated: this type was for internal use and will be removed in the next release.
type Format = formatter.Format

// Stack contains deployed stack information.
//
// Deprecated: this type was for internal use and will be removed in the next release.
type Stack struct {
	// Name is the name of the stack
	Name string
	// Services is the number of the services
	Services int
}

// StackWrite writes formatted stacks using the Context
//
// Deprecated: this function was for internal use and will be removed in the next release.
func StackWrite(ctx formatter.Context, stacks []*Stack) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, stack := range stacks {
			if err := format(&stackContext{s: stack}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newStackContext(), render)
}

type stackContext struct {
	formatter.HeaderContext
	s *Stack
}

func newStackContext() *stackContext {
	stackCtx := stackContext{}
	stackCtx.Header = formatter.SubHeaderContext{
		"Name":     formatter.NameHeader,
		"Services": stackServicesHeader,
	}
	return &stackCtx
}

func (s *stackContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(s)
}

func (s *stackContext) Name() string {
	return s.s.Name
}

func (s *stackContext) Services() string {
	return strconv.Itoa(s.s.Services)
}
