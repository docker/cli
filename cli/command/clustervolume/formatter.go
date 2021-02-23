package clustervolume

import (
	"fmt"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/docker/docker/api/types/swarm"
)

const (
	defaultClusterVolumeTableFormat = "table {{.ID}}\t{{.Name}}\t{{.Group}}\t{{.Driver}}\t{{.Availability}}\t{{.Status}}"
	// TODO(dperny): fill in template
	volumeInspectPrettyTemplate = ``

	volumeIDHeader           = "ID"
	volumeGroupHeader        = "GROUP"
	volumeAvailabilityHeader = "AVAILABILITY"
)

func NewFormat(source string) formatter.Format {
	switch source {
	case formatter.PrettyFormatKey:
		return volumeInspectPrettyTemplate
	case formatter.TableFormatKey:
		return defaultClusterVolumeTableFormat
	}

	return formatter.Format(source)
}

type clusterVolumeContext struct {
	formatter.HeaderContext
	swarm.Volume
}

func newClusterVolumeContext() *clusterVolumeContext {
	cvCtx := &clusterVolumeContext{}

	cvCtx.Header = formatter.SubHeaderContext{
		"ID":           volumeIDHeader,
		"Name":         formatter.NameHeader,
		"Group":        volumeGroupHeader,
		"Driver":       formatter.DriverHeader,
		"Availability": volumeAvailabilityHeader,
		"Status":       formatter.StatusHeader,
	}

	return cvCtx
}

func (ctx *clusterVolumeContext) ID() string {
	return ctx.Volume.ID
}

func (ctx *clusterVolumeContext) Name() string {
	return ctx.Volume.Spec.Annotations.Name
}

func (ctx *clusterVolumeContext) Group() string {
	return ctx.Volume.Spec.Group
}

func (ctx *clusterVolumeContext) Driver() string {
	return ctx.Volume.Spec.Driver.Name
}

func (ctx *clusterVolumeContext) Availability() string {
	return string(ctx.Volume.Spec.Availability)
}

func (ctx *clusterVolumeContext) Status() string {
	if ctx.Volume.VolumeInfo == nil || ctx.Volume.VolumeInfo.VolumeID == "" {
		return "pending creation"
	}

	l := len(ctx.Volume.PublishStatus)
	switch l {
	case 0:
		return "created"
	case 1:
		return "in use (1 node)"
	default:
		return fmt.Sprintf("in use (%d nodes)", l)
	}
}

func FormatWrite(ctx formatter.Context, volumes []swarm.Volume) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, volume := range volumes {
			volumeCtx := &clusterVolumeContext{Volume: volume}
			if err := format(volumeCtx); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newClusterVolumeContext(), render)
}

func InspectFormatWrite(ctx formatter.Context, refs []string, getRef inspect.GetRefFunc) error {
	if ctx.Format != volumeInspectPrettyTemplate {
		return inspect.Inspect(ctx.Output, refs, string(ctx.Format), getRef)
	}
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, ref := range refs {
			volumeI, _, err := getRef(ref)
			if err != nil {
				return err
			}
			volume, ok := volumeI.(swarm.Volume)
			if !ok {
				return fmt.Errorf("got wrong object type to inspect: %v", ok)
			}
			if err := format(&clusterVolumeContext{Volume: volume}); err != nil {
				return err
			}
		}
		return nil
	}

	return ctx.Write(&clusterVolumeContext{}, render)
}
