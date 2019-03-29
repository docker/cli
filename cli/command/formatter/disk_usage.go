package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
)

const (
	defaultDiskUsageImageTableFormat      = "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedSince}}\t{{.VirtualSize}}\t{{.SharedSize}}\t{{.UniqueSize}}\t{{.Containers}}"
	defaultDiskUsageContainerTableFormat  = "table {{.ID}}\t{{.Image}}\t{{.Command}}\t{{.LocalVolumes}}\t{{.Size}}\t{{.RunningFor}}\t{{.Status}}\t{{.Names}}"
	defaultDiskUsageVolumeTableFormat     = "table {{.Name}}\t{{.Links}}\t{{.Size}}"
	defaultDiskUsageBuildCacheTableFormat = "table {{.ID}}\t{{.CacheType}}\t{{.Size}}\t{{.CreatedSince}}\t{{.LastUsedSince}}\t{{.UsageCount}}\t{{.Shared}}"
	defaultDiskUsageTableFormat           = "table {{.Type}}\t{{.TotalCount}}\t{{.Active}}\t{{.Size}}\t{{.Reclaimable}}"

	typeHeader        = "TYPE"
	totalHeader       = "TOTAL"
	activeHeader      = "ACTIVE"
	reclaimableHeader = "RECLAIMABLE"
	containersHeader  = "CONTAINERS"
	sharedSizeHeader  = "SHARED SIZE"
	uniqueSizeHeader  = "UNIQUE SIZE"
)

// DiskUsageContext contains disk usage specific information required by the formatter, encapsulate a Context struct.
type DiskUsageContext struct {
	Context
	Verbose     bool
	LayersSize  int64
	Images      []*types.ImageSummary
	Containers  []*types.Container
	Volumes     []*types.Volume
	BuildCache  []*types.BuildCache
	BuilderSize int64
}

func (ctx *DiskUsageContext) startSubsection(format string) (*template.Template, error) {
	ctx.buffer = bytes.NewBufferString("")
	ctx.header = ""
	ctx.Format = Format(format)
	ctx.preFormat()

	return ctx.parseFormat()
}

// NewDiskUsageFormat returns a format for rendering an DiskUsageContext
func NewDiskUsageFormat(source string, verbose bool) Format {
	switch {
	case verbose && source == RawFormatKey:
		format := `{{range .Images}}type: Image
` + NewImageFormat(source, false, true) + `
{{end -}}
{{range .Containers}}type: Container
` + NewContainerFormat(source, false, true) + `
{{end -}}
{{range .Volumes}}type: Volume
` + NewVolumeFormat(source, false) + `
{{end -}}
{{range .BuildCache}}type: Build Cache
` + NewBuildCacheFormat(source, false) + `
{{end -}}`
		return format
	case !verbose && source == TableFormatKey:
		return Format(defaultDiskUsageTableFormat)
	case !verbose && source == RawFormatKey:
		format := `type: {{.Type}}
total: {{.TotalCount}}
active: {{.Active}}
size: {{.Size}}
reclaimable: {{.Reclaimable}}
`
		return Format(format)
	default:
		return Format(source)
	}
}

func (ctx *DiskUsageContext) Write() (err error) {
	if ctx.Verbose {
		return ctx.verboseWrite()
	}
	ctx.buffer = bytes.NewBufferString("")
	ctx.preFormat()

	tmpl, err := ctx.parseFormat()
	if err != nil {
		return err
	}

	err = ctx.contextFormat(tmpl, &diskUsageImagesContext{
		totalSize: ctx.LayersSize,
		images:    ctx.Images,
	})
	if err != nil {
		return err
	}
	err = ctx.contextFormat(tmpl, &diskUsageContainersContext{
		containers: ctx.Containers,
	})
	if err != nil {
		return err
	}

	err = ctx.contextFormat(tmpl, &diskUsageVolumesContext{
		volumes: ctx.Volumes,
	})
	if err != nil {
		return err
	}

	err = ctx.contextFormat(tmpl, &diskUsageBuilderContext{
		builderSize: ctx.BuilderSize,
		buildCache:  ctx.BuildCache,
	})
	if err != nil {
		return err
	}

	diskUsageContainersCtx := diskUsageContainersContext{containers: []*types.Container{}}
	diskUsageContainersCtx.Header = SubHeaderContext{
		"Type":        typeHeader,
		"TotalCount":  totalHeader,
		"Active":      activeHeader,
		"Size":        SizeHeader,
		"Reclaimable": reclaimableHeader,
	}
	ctx.postFormat(tmpl, &diskUsageContainersCtx)

	return err
}

type diskUsageContext struct {
	Images     []*imageContext
	Containers []*containerContext
	Volumes    []*volumeContext
	BuildCache []*buildCacheContext
}

func (ctx *DiskUsageContext) verboseWrite() error {
	duc := &diskUsageContext{
		Images:     make([]*imageContext, 0, len(ctx.Images)),
		Containers: make([]*containerContext, 0, len(ctx.Containers)),
		Volumes:    make([]*volumeContext, 0, len(ctx.Volumes)),
		BuildCache: make([]*buildCacheContext, 0, len(ctx.BuildCache)),
	}
	trunc := ctx.Format.IsTable()

	// First images
	for _, i := range ctx.Images {
		repo := "<none>"
		tag := "<none>"
		if len(i.RepoTags) > 0 && !isDangling(*i) {
			// Only show the first tag
			ref, err := reference.ParseNormalizedNamed(i.RepoTags[0])
			if err != nil {
				continue
			}
			if nt, ok := ref.(reference.NamedTagged); ok {
				repo = reference.FamiliarName(ref)
				tag = nt.Tag()
			}
		}

		duc.Images = append(duc.Images, &imageContext{
			repo:  repo,
			tag:   tag,
			trunc: trunc,
			i:     *i,
		})
	}

	// Now containers
	for _, c := range ctx.Containers {
		// Don't display the virtual size
		c.SizeRootFs = 0
		duc.Containers = append(duc.Containers, &containerContext{trunc: trunc, c: *c})
	}

	// And volumes
	for _, v := range ctx.Volumes {
		duc.Volumes = append(duc.Volumes, &volumeContext{v: *v})
	}

	// And build cache
	buildCacheSort(ctx.BuildCache)
	for _, v := range ctx.BuildCache {
		duc.BuildCache = append(duc.BuildCache, &buildCacheContext{v: v, trunc: trunc})
	}

	if ctx.Format == TableFormatKey {
		return ctx.verboseWriteTable(duc)
	}

	ctx.preFormat()
	tmpl, err := ctx.parseFormat()
	if err != nil {
		return err
	}
	return tmpl.Execute(ctx.Output, duc)
}

func (ctx *DiskUsageContext) verboseWriteTable(duc *diskUsageContext) error {
	tmpl, err := ctx.startSubsection(defaultDiskUsageImageTableFormat)
	if err != nil {
		return err
	}
	ctx.Output.Write([]byte("Images space usage:\n\n"))
	for _, img := range duc.Images {
		if err := ctx.contextFormat(tmpl, img); err != nil {
			return err
		}
	}
	ctx.postFormat(tmpl, newImageContext())

	tmpl, err = ctx.startSubsection(defaultDiskUsageContainerTableFormat)
	if err != nil {
		return err
	}
	ctx.Output.Write([]byte("\nContainers space usage:\n\n"))
	for _, c := range duc.Containers {
		if err := ctx.contextFormat(tmpl, c); err != nil {
			return err
		}
	}
	ctx.postFormat(tmpl, newContainerContext())

	tmpl, err = ctx.startSubsection(defaultDiskUsageVolumeTableFormat)
	if err != nil {
		return err
	}
	ctx.Output.Write([]byte("\nLocal Volumes space usage:\n\n"))
	for _, v := range duc.Volumes {
		if err := ctx.contextFormat(tmpl, v); err != nil {
			return err
		}
	}
	ctx.postFormat(tmpl, newVolumeContext())

	tmpl, err = ctx.startSubsection(defaultDiskUsageBuildCacheTableFormat)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Output, "\nBuild cache usage: %s\n\n", units.HumanSize(float64(ctx.BuilderSize)))
	for _, v := range duc.BuildCache {
		if err := ctx.contextFormat(tmpl, v); err != nil {
			return err
		}
	}
	ctx.postFormat(tmpl, newBuildCacheContext())

	return nil
}

type diskUsageImagesContext struct {
	HeaderContext
	totalSize int64
	images    []*types.ImageSummary
}

func (c *diskUsageImagesContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (c *diskUsageImagesContext) Type() string {
	return "Images"
}

func (c *diskUsageImagesContext) TotalCount() string {
	return fmt.Sprintf("%d", len(c.images))
}

func (c *diskUsageImagesContext) Active() string {
	used := 0
	for _, i := range c.images {
		if i.Containers > 0 {
			used++
		}
	}

	return fmt.Sprintf("%d", used)
}

func (c *diskUsageImagesContext) Size() string {
	return units.HumanSize(float64(c.totalSize))

}

func (c *diskUsageImagesContext) Reclaimable() string {
	var used int64

	for _, i := range c.images {
		if i.Containers != 0 {
			if i.VirtualSize == -1 || i.SharedSize == -1 {
				continue
			}
			used += i.VirtualSize - i.SharedSize
		}
	}

	reclaimable := c.totalSize - used
	if c.totalSize > 0 {
		return fmt.Sprintf("%s (%v%%)", units.HumanSize(float64(reclaimable)), (reclaimable*100)/c.totalSize)
	}
	return units.HumanSize(float64(reclaimable))
}

type diskUsageContainersContext struct {
	HeaderContext
	containers []*types.Container
}

func (c *diskUsageContainersContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (c *diskUsageContainersContext) Type() string {
	return "Containers"
}

func (c *diskUsageContainersContext) TotalCount() string {
	return fmt.Sprintf("%d", len(c.containers))
}

func (c *diskUsageContainersContext) isActive(container types.Container) bool {
	return strings.Contains(container.State, "running") ||
		strings.Contains(container.State, "paused") ||
		strings.Contains(container.State, "restarting")
}

func (c *diskUsageContainersContext) Active() string {
	used := 0
	for _, container := range c.containers {
		if c.isActive(*container) {
			used++
		}
	}

	return fmt.Sprintf("%d", used)
}

func (c *diskUsageContainersContext) Size() string {
	var size int64

	for _, container := range c.containers {
		size += container.SizeRw
	}

	return units.HumanSize(float64(size))
}

func (c *diskUsageContainersContext) Reclaimable() string {
	var reclaimable int64
	var totalSize int64

	for _, container := range c.containers {
		if !c.isActive(*container) {
			reclaimable += container.SizeRw
		}
		totalSize += container.SizeRw
	}

	if totalSize > 0 {
		return fmt.Sprintf("%s (%v%%)", units.HumanSize(float64(reclaimable)), (reclaimable*100)/totalSize)
	}

	return units.HumanSize(float64(reclaimable))
}

type diskUsageVolumesContext struct {
	HeaderContext
	volumes []*types.Volume
}

func (c *diskUsageVolumesContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (c *diskUsageVolumesContext) Type() string {
	return "Local Volumes"
}

func (c *diskUsageVolumesContext) TotalCount() string {
	return fmt.Sprintf("%d", len(c.volumes))
}

func (c *diskUsageVolumesContext) Active() string {

	used := 0
	for _, v := range c.volumes {
		if v.UsageData.RefCount > 0 {
			used++
		}
	}

	return fmt.Sprintf("%d", used)
}

func (c *diskUsageVolumesContext) Size() string {
	var size int64

	for _, v := range c.volumes {
		if v.UsageData.Size != -1 {
			size += v.UsageData.Size
		}
	}

	return units.HumanSize(float64(size))
}

func (c *diskUsageVolumesContext) Reclaimable() string {
	var reclaimable int64
	var totalSize int64

	for _, v := range c.volumes {
		if v.UsageData.Size != -1 {
			if v.UsageData.RefCount == 0 {
				reclaimable += v.UsageData.Size
			}
			totalSize += v.UsageData.Size
		}
	}

	if totalSize > 0 {
		return fmt.Sprintf("%s (%v%%)", units.HumanSize(float64(reclaimable)), (reclaimable*100)/totalSize)
	}

	return units.HumanSize(float64(reclaimable))
}

type diskUsageBuilderContext struct {
	HeaderContext
	builderSize int64
	buildCache  []*types.BuildCache
}

func (c *diskUsageBuilderContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (c *diskUsageBuilderContext) Type() string {
	return "Build Cache"
}

func (c *diskUsageBuilderContext) TotalCount() string {
	return fmt.Sprintf("%d", len(c.buildCache))
}

func (c *diskUsageBuilderContext) Active() string {
	numActive := 0
	for _, bc := range c.buildCache {
		if bc.InUse {
			numActive++
		}
	}
	return fmt.Sprintf("%d", numActive)
}

func (c *diskUsageBuilderContext) Size() string {
	return units.HumanSize(float64(c.builderSize))
}

func (c *diskUsageBuilderContext) Reclaimable() string {
	var inUseBytes int64
	for _, bc := range c.buildCache {
		if bc.InUse && !bc.Shared {
			inUseBytes += bc.Size
		}
	}

	return units.HumanSize(float64(c.builderSize - inUseBytes))
}
