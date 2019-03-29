package formatter

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
)

const (
	defaultVolumeQuietFormat = "{{.Name}}"
	defaultVolumeTableFormat = "table {{.Driver}}\t{{.Name}}"

	volumeNameHeader = "VOLUME NAME"
	mountpointHeader = "MOUNTPOINT"
	linksHeader      = "LINKS"
	// Status header ?
)

// NewVolumeFormat returns a format for use with a volume Context
func NewVolumeFormat(source string, quiet bool) Format {
	switch source {
	case TableFormatKey:
		if quiet {
			return defaultVolumeQuietFormat
		}
		return defaultVolumeTableFormat
	case RawFormatKey:
		if quiet {
			return `name: {{.Name}}`
		}
		return `name: {{.Name}}\ndriver: {{.Driver}}\n`
	}
	return Format(source)
}

// VolumeWrite writes formatted volumes using the Context
func VolumeWrite(ctx Context, volumes []*types.Volume) error {
	render := func(format func(subContext SubContext) error) error {
		for _, volume := range volumes {
			if err := format(&volumeContext{v: *volume}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newVolumeContext(), render)
}

type volumeContext struct {
	HeaderContext
	v types.Volume
}

func newVolumeContext() *volumeContext {
	volumeCtx := volumeContext{}
	volumeCtx.Header = SubHeaderContext{
		"Name":       volumeNameHeader,
		"Driver":     DriverHeader,
		"Scope":      ScopeHeader,
		"Mountpoint": mountpointHeader,
		"Labels":     LabelsHeader,
		"Links":      linksHeader,
		"Size":       SizeHeader,
	}
	return &volumeCtx
}

func (c *volumeContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (c *volumeContext) Name() string {
	return c.v.Name
}

func (c *volumeContext) Driver() string {
	return c.v.Driver
}

func (c *volumeContext) Scope() string {
	return c.v.Scope
}

func (c *volumeContext) Mountpoint() string {
	return c.v.Mountpoint
}

func (c *volumeContext) Labels() string {
	if c.v.Labels == nil {
		return ""
	}

	var joinLabels []string
	for k, v := range c.v.Labels {
		joinLabels = append(joinLabels, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(joinLabels, ",")
}

func (c *volumeContext) Label(name string) string {
	if c.v.Labels == nil {
		return ""
	}
	return c.v.Labels[name]
}

func (c *volumeContext) Links() string {
	if c.v.UsageData == nil {
		return "N/A"
	}
	return fmt.Sprintf("%d", c.v.UsageData.RefCount)
}

func (c *volumeContext) Size() string {
	if c.v.UsageData == nil {
		return "N/A"
	}
	return units.HumanSize(float64(c.v.UsageData.Size))
}
