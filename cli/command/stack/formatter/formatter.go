package formatter

import (
	"strconv"

	"github.com/docker/cli/cli/command/formatter"
)

const (
	// KubernetesStackTableFormat is the default Kubernetes stack format
	KubernetesStackTableFormat = "table {{.Name}}\t{{.Services}}\t{{.Orchestrator}}\t{{.Namespace}}"
	// SwarmStackTableFormat is the default Swarm stack format
	SwarmStackTableFormat = "table {{.Name}}\t{{.Services}}\t{{.Orchestrator}}"

	stackServicesHeader      = "SERVICES"
	stackOrchestrastorHeader = "ORCHESTRATOR"
	stackNamespaceHeader     = "NAMESPACE"

	// TableFormatKey is an alias for formatter.TableFormatKey
	TableFormatKey = formatter.TableFormatKey
)

// Context is an alias for formatter.Context
type Context = formatter.Context

// Format is an alias for formatter.Format
type Format = formatter.Format

// Stack contains deployed stack information.
type Stack struct {
	// Name is the name of the stack
	Name string
	// Services is the number of the services
	Services int
	// Orchestrator is the platform where the stack is deployed
	Orchestrator string
	// Namespace is the Kubernetes namespace assigned to the stack
	Namespace string
}

// StackWrite writes formatted stacks using the Context
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
		"Name":         formatter.NameHeader,
		"Services":     stackServicesHeader,
		"Orchestrator": stackOrchestrastorHeader,
		"Namespace":    stackNamespaceHeader,
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

func (s *stackContext) Orchestrator() string {
	return s.s.Orchestrator
}

func (s *stackContext) Namespace() string {
	return s.s.Namespace
}
