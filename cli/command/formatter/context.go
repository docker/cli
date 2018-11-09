package formatter

const (
	// ClientContextTableFormat is the default client context format
	ClientContextTableFormat = "table {{.Name}}{{if .Current}} *{{end}}\t{{.Description}}\t{{.DockerEndpoint}}\t{{.KubernetesEndpoint}}\t{{.StackOrchestrator}}"

	dockerEndpointHeader     = "DOCKER ENDPOINT"
	kubernetesEndpointHeader = "KUBERNETES ENDPOINT"
	stackOrchestrastorHeader = "ORCHESTRATOR"
	quietContextFormat       = "{{.Name}}"
)

// NewClientContextFormat returns a Format for rendering using a Context
func NewClientContextFormat(source string, quiet bool) Format {
	if quiet {
		return Format(quietContextFormat)
	}
	if source == TableFormatKey {
		return Format(ClientContextTableFormat)
	}
	return Format(source)
}

// ClientContext is a context for display
type ClientContext struct {
	Name               string
	Description        string
	DockerEndpoint     string
	KubernetesEndpoint string
	StackOrchestrator  string
	Current            bool
}

// ClientContextWrite writes formatted contexts using the Context
func ClientContextWrite(ctx Context, contexts []*ClientContext) error {
	render := func(format func(subContext SubContext) error) error {
		for _, context := range contexts {
			if err := format(&clientContextContext{c: context}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newClientContextContext(), render)
}

type clientContextContext struct {
	HeaderContext
	c *ClientContext
}

func newClientContextContext() *clientContextContext {
	ctx := clientContextContext{}
	ctx.Header = SubHeaderContext{
		"Name":               NameHeader,
		"Description":        DescriptionHeader,
		"DockerEndpoint":     dockerEndpointHeader,
		"KubernetesEndpoint": kubernetesEndpointHeader,
		"StackOrchestrator":  stackOrchestrastorHeader,
	}
	return &ctx
}

func (c *clientContextContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (c *clientContextContext) Current() bool {
	return c.c.Current
}

func (c *clientContextContext) Name() string {
	return c.c.Name
}

func (c *clientContextContext) Description() string {
	return c.c.Description
}

func (c *clientContextContext) DockerEndpoint() string {
	return c.c.DockerEndpoint
}

func (c *clientContextContext) KubernetesEndpoint() string {
	return c.c.KubernetesEndpoint
}

func (c *clientContextContext) StackOrchestrator() string {
	return c.c.StackOrchestrator
}
