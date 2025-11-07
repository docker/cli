package formatter

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	"github.com/distribution/reference"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/build"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

const (
	defaultDiskUsageImageTableFormat      Format = "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedSince}}\t{{.Size}}\t{{.SharedSize}}\t{{.UniqueSize}}\t{{.Containers}}"
	defaultDiskUsageContainerTableFormat  Format = "table {{.ID}}\t{{.Image}}\t{{.Command}}\t{{.LocalVolumes}}\t{{.Size}}\t{{.RunningFor}}\t{{.Status}}\t{{.Names}}"
	defaultDiskUsageVolumeTableFormat     Format = "table {{.Name}}\t{{.Links}}\t{{.Size}}"
	defaultDiskUsageBuildCacheTableFormat Format = "table {{.ID}}\t{{.CacheType}}\t{{.Size}}\t{{.CreatedSince}}\t{{.LastUsedSince}}\t{{.UsageCount}}\t{{.Shared}}"
	defaultDiskUsageTableFormat           Format = "table {{.Type}}\t{{.TotalCount}}\t{{.Active}}\t{{.Size}}\t{{.Reclaimable}}"

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
	Verbose bool

	ImageDiskUsage      client.ImagesDiskUsage
	BuildCacheDiskUsage client.BuildCacheDiskUsage
	ContainerDiskUsage  client.ContainersDiskUsage
	VolumeDiskUsage     client.VolumesDiskUsage
}

func (ctx *DiskUsageContext) startSubsection(format Format) (*template.Template, error) {
	ctx.buffer = &bytes.Buffer{}
	ctx.header = ""
	ctx.Format = format
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
		return defaultDiskUsageTableFormat
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
	ctx.buffer = &bytes.Buffer{}
	ctx.preFormat()

	tmpl, err := ctx.parseFormat()
	if err != nil {
		return err
	}

	err = ctx.contextFormat(tmpl, &diskUsageImagesContext{
		totalCount:  ctx.ImageDiskUsage.TotalCount,
		activeCount: ctx.ImageDiskUsage.ActiveCount,
		totalSize:   ctx.ImageDiskUsage.TotalSize,
		reclaimable: ctx.ImageDiskUsage.Reclaimable,
		images:      ctx.ImageDiskUsage.Items,
	})
	if err != nil {
		return err
	}
	err = ctx.contextFormat(tmpl, &diskUsageContainersContext{
		totalCount:  ctx.ContainerDiskUsage.TotalCount,
		activeCount: ctx.ContainerDiskUsage.ActiveCount,
		totalSize:   ctx.ContainerDiskUsage.TotalSize,
		reclaimable: ctx.ContainerDiskUsage.Reclaimable,
		containers:  ctx.ContainerDiskUsage.Items,
	})
	if err != nil {
		return err
	}

	err = ctx.contextFormat(tmpl, &diskUsageVolumesContext{
		totalCount:  ctx.VolumeDiskUsage.TotalCount,
		activeCount: ctx.VolumeDiskUsage.ActiveCount,
		totalSize:   ctx.VolumeDiskUsage.TotalSize,
		reclaimable: ctx.VolumeDiskUsage.Reclaimable,
		volumes:     ctx.VolumeDiskUsage.Items,
	})
	if err != nil {
		return err
	}

	err = ctx.contextFormat(tmpl, &diskUsageBuilderContext{
		totalCount:  ctx.BuildCacheDiskUsage.TotalCount,
		activeCount: ctx.BuildCacheDiskUsage.ActiveCount,
		builderSize: ctx.BuildCacheDiskUsage.TotalSize,
		reclaimable: ctx.BuildCacheDiskUsage.Reclaimable,
		buildCache:  ctx.BuildCacheDiskUsage.Items,
	})
	if err != nil {
		return err
	}

	diskUsageContainersCtx := diskUsageContainersContext{containers: []container.Summary{}}
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
	Containers []*ContainerContext
	Volumes    []*volumeContext
	BuildCache []*buildCacheContext
}

func (ctx *DiskUsageContext) verboseWrite() error {
	duc := &diskUsageContext{
		Images:     make([]*imageContext, 0, len(ctx.ImageDiskUsage.Items)),
		Containers: make([]*ContainerContext, 0, len(ctx.ContainerDiskUsage.Items)),
		Volumes:    make([]*volumeContext, 0, len(ctx.VolumeDiskUsage.Items)),
		BuildCache: make([]*buildCacheContext, 0, len(ctx.BuildCacheDiskUsage.Items)),
	}
	trunc := ctx.Format.IsTable()

	// First images
	for _, i := range ctx.ImageDiskUsage.Items {
		repo := "<none>"
		tag := "<none>"
		if len(i.RepoTags) > 0 && !isDangling(i) {
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
			i:     i,
		})
	}

	// Now containers
	for _, c := range ctx.ContainerDiskUsage.Items {
		// Don't display the virtual size
		c.SizeRootFs = 0
		duc.Containers = append(duc.Containers, &ContainerContext{trunc: trunc, c: c})
	}

	// And volumes
	for _, v := range ctx.VolumeDiskUsage.Items {
		duc.Volumes = append(duc.Volumes, &volumeContext{v: v})
	}

	// And build cache
	buildCacheSort(ctx.BuildCacheDiskUsage.Items)
	for _, v := range ctx.BuildCacheDiskUsage.Items {
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
	_, _ = ctx.Output.Write([]byte("Images space usage:\n\n"))
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
	_, _ = ctx.Output.Write([]byte("\nContainers space usage:\n\n"))
	for _, c := range duc.Containers {
		if err := ctx.contextFormat(tmpl, c); err != nil {
			return err
		}
	}
	ctx.postFormat(tmpl, NewContainerContext())

	tmpl, err = ctx.startSubsection(defaultDiskUsageVolumeTableFormat)
	if err != nil {
		return err
	}
	_, _ = ctx.Output.Write([]byte("\nLocal Volumes space usage:\n\n"))
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
	_, _ = fmt.Fprintf(ctx.Output, "\nBuild cache usage: %s\n\n", units.HumanSize(float64(ctx.BuildCacheDiskUsage.TotalSize)))
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
	totalSize   int64
	reclaimable int64
	totalCount  int64
	activeCount int64
	images      []image.Summary
}

func (c *diskUsageImagesContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (*diskUsageImagesContext) Type() string {
	return "Images"
}

func (c *diskUsageImagesContext) TotalCount() string {
	return strconv.FormatInt(c.totalCount, 10)
}

func (c *diskUsageImagesContext) Active() string {
	return strconv.FormatInt(c.activeCount, 10)
}

func (c *diskUsageImagesContext) Size() string {
	return units.HumanSize(float64(c.totalSize))
}

func (c *diskUsageImagesContext) Reclaimable() string {
	if c.totalSize > 0 {
		return fmt.Sprintf("%s (%v%%)", units.HumanSize(float64(c.reclaimable)), (c.reclaimable*100)/c.totalSize)
	}
	return units.HumanSize(float64(c.reclaimable))
}

type diskUsageContainersContext struct {
	HeaderContext
	totalCount  int64
	activeCount int64
	totalSize   int64
	reclaimable int64
	containers  []container.Summary
}

func (c *diskUsageContainersContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (*diskUsageContainersContext) Type() string {
	return "Containers"
}

func (c *diskUsageContainersContext) TotalCount() string {
	return strconv.FormatInt(c.totalCount, 10)
}

func (c *diskUsageContainersContext) Active() string {
	return strconv.FormatInt(c.activeCount, 10)
}

func (c *diskUsageContainersContext) Size() string {
	return units.HumanSize(float64(c.totalSize))
}

func (c *diskUsageContainersContext) Reclaimable() string {
	if c.totalSize > 0 {
		return fmt.Sprintf("%s (%v%%)", units.HumanSize(float64(c.reclaimable)), (c.reclaimable*100)/c.totalSize)
	}

	return units.HumanSize(float64(c.reclaimable))
}

type diskUsageVolumesContext struct {
	HeaderContext
	totalCount  int64
	activeCount int64
	totalSize   int64
	reclaimable int64
	volumes     []volume.Volume
}

func (c *diskUsageVolumesContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (*diskUsageVolumesContext) Type() string {
	return "Local Volumes"
}

func (c *diskUsageVolumesContext) TotalCount() string {
	return strconv.FormatInt(c.totalCount, 10)
}

func (c *diskUsageVolumesContext) Active() string {
	return strconv.FormatInt(c.activeCount, 10)
}

func (c *diskUsageVolumesContext) Size() string {
	return units.HumanSize(float64(c.totalSize))
}

func (c *diskUsageVolumesContext) Reclaimable() string {
	if c.totalSize > 0 {
		return fmt.Sprintf("%s (%v%%)", units.HumanSize(float64(c.reclaimable)), (c.reclaimable*100)/c.totalSize)
	}

	return units.HumanSize(float64(c.reclaimable))
}

type diskUsageBuilderContext struct {
	HeaderContext
	totalCount  int64
	activeCount int64
	builderSize int64
	reclaimable int64
	buildCache  []build.CacheRecord
}

func (c *diskUsageBuilderContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (*diskUsageBuilderContext) Type() string {
	return "Build Cache"
}

func (c *diskUsageBuilderContext) TotalCount() string {
	return strconv.FormatInt(c.totalCount, 10)
}

func (c *diskUsageBuilderContext) Active() string {
	return strconv.FormatInt(c.activeCount, 10)
}

func (c *diskUsageBuilderContext) Size() string {
	return units.HumanSize(float64(c.builderSize))
}

func (c *diskUsageBuilderContext) Reclaimable() string {
	return units.HumanSize(float64(c.reclaimable))
}
