package task

import (
	"fmt"
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

const (
	defaultTaskTableFormat = "table {{.ID}}\t{{.Name}}\t{{.Image}}\t{{.Node}}\t{{.DesiredState}}\t{{.CurrentState}}\t{{.Error}}\t{{.Ports}}"

	nodeHeader         = "NODE"
	taskIDHeader       = "ID"
	desiredStateHeader = "DESIRED STATE"
	currentStateHeader = "CURRENT STATE"

	maxErrLength = 30
)

// newTaskFormat returns a Format for rendering using a taskContext.
func newTaskFormat(source string, quiet bool) formatter.Format {
	switch source {
	case formatter.TableFormatKey:
		if quiet {
			return formatter.DefaultQuietFormat
		}
		return defaultTaskTableFormat
	case formatter.RawFormatKey:
		if quiet {
			return `id: {{.ID}}`
		}
		return `id: {{.ID}}\nname: {{.Name}}\nimage: {{.Image}}\nnode: {{.Node}}\ndesired_state: {{.DesiredState}}\ncurrent_state: {{.CurrentState}}\nerror: {{.Error}}\nports: {{.Ports}}\n`
	}
	return formatter.Format(source)
}

// formatWrite writes the context.
func formatWrite(fmtCtx formatter.Context, tasks client.TaskListResult, names map[string]string, nodes map[string]string) error {
	taskCtx := &taskContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"ID":           taskIDHeader,
				"Name":         formatter.NameHeader,
				"Image":        formatter.ImageHeader,
				"Node":         nodeHeader,
				"DesiredState": desiredStateHeader,
				"CurrentState": currentStateHeader,
				"Error":        formatter.ErrorHeader,
				"Ports":        formatter.PortsHeader,
			},
		},
	}
	return fmtCtx.Write(taskCtx, func(format func(subContext formatter.SubContext) error) error {
		for _, task := range tasks.Items {
			if err := format(&taskContext{
				trunc: fmtCtx.Trunc,
				task:  task,
				name:  names[task.ID],
				node:  nodes[task.ID],
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

type taskContext struct {
	formatter.HeaderContext
	trunc bool
	task  swarm.Task
	name  string
	node  string
}

func (c *taskContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *taskContext) ID() string {
	if c.trunc {
		return formatter.TruncateID(c.task.ID)
	}
	return c.task.ID
}

func (c *taskContext) Name() string {
	return c.name
}

func (c *taskContext) Image() string {
	image := c.task.Spec.ContainerSpec.Image
	if c.trunc {
		ref, err := reference.ParseNormalizedNamed(image)
		if err == nil {
			// update image string for display, (strips any digest)
			if nt, ok := ref.(reference.NamedTagged); ok {
				if namedTagged, err := reference.WithTag(reference.TrimNamed(nt), nt.Tag()); err == nil {
					image = reference.FamiliarString(namedTagged)
				}
			}
		}
	}
	return image
}

func (c *taskContext) Node() string {
	return c.node
}

func (c *taskContext) DesiredState() string {
	return formatter.PrettyPrint(c.task.DesiredState)
}

func (c *taskContext) CurrentState() string {
	return fmt.Sprintf("%s %s ago",
		formatter.PrettyPrint(c.task.Status.State),
		strings.ToLower(units.HumanDuration(time.Since(c.task.Status.Timestamp))),
	)
}

func (c *taskContext) Error() string {
	// Trim and quote the error message.
	taskErr := c.task.Status.Err
	if c.trunc {
		taskErr = formatter.Ellipsis(taskErr, maxErrLength)
	}
	if len(taskErr) > 0 {
		taskErr = fmt.Sprintf(`"%s"`, taskErr)
	}
	return taskErr
}

func (c *taskContext) Ports() string {
	if len(c.task.Status.PortStatus.Ports) == 0 {
		return ""
	}
	ports := make([]string, 0, len(c.task.Status.PortStatus.Ports))
	for _, pConfig := range c.task.Status.PortStatus.Ports {
		ports = append(ports, fmt.Sprintf("*:%d->%d/%s",
			pConfig.PublishedPort,
			pConfig.TargetPort,
			pConfig.Protocol,
		))
	}
	return strings.Join(ports, ",")
}
