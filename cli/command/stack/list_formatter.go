package stack

import (
	"strconv"

	"github.com/docker/cli/cli/command/formatter"
)

// stackTableFormat is the default Swarm stack format
const stackTableFormat formatter.Format = "table {{.Name}}\t{{.Services}}"

// stackSummary contains deployed stack information.
type stackSummary struct {
	Name     string // Name is the name of the stack.
	Services int    // Services is the number services in the stack.
}

// stackWrite writes formatted stacks using the Context
func stackWrite(fmtCtx formatter.Context, stacks []stackSummary) error {
	stackCtx := &stackContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"Name":     formatter.NameHeader,
				"Services": "SERVICES",
			},
		},
	}
	return fmtCtx.Write(stackCtx, func(format func(subContext formatter.SubContext) error) error {
		for _, stack := range stacks {
			if err := format(&stackContext{s: stack}); err != nil {
				return err
			}
		}
		return nil
	})
}

type stackContext struct {
	formatter.HeaderContext
	s stackSummary
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
