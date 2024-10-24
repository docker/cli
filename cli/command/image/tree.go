package image

import (
	"context"
	"fmt"
	"sort"

	"github.com/containerd/platforms"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-units"
	"github.com/morikuni/aec"
)

type treeOptions struct {
	all     bool
	filters filters.Args
}

func runTree(ctx context.Context, dockerCLI command.Cli, opts treeOptions) error {
	images, err := dockerCLI.Client().ImageList(ctx, imagetypes.ListOptions{
		All:       opts.all,
		Filters:   opts.filters,
		Manifests: true,
	})
	if err != nil {
		return err
	}

	warningColor := aec.LightYellowF
	if !dockerCLI.Out().IsTerminal() {
		warningColor = noColor{}
	}
	_, _ = fmt.Fprintln(dockerCLI.Out(), warningColor.Apply("WARNING: This is an experimental feature. The output may change and shouldn't be depended on."))
	_, _ = fmt.Fprintln(dockerCLI.Out(), "")

	out := dockerCLI.Out()
	_, width := out.GetTtySize()
	if width == 0 {
		width = 80
	}
	if width < 20 {
		width = 20
	}

	headerColor := aec.NewBuilder(aec.DefaultF, aec.Bold).ANSI
	topNameColor := aec.NewBuilder(aec.BlueF, aec.Bold).ANSI
	normalColor := aec.NewBuilder(aec.DefaultF).ANSI
	greenColor := aec.NewBuilder(aec.GreenF).ANSI
	untaggedColor := aec.NewBuilder(aec.Faint).ANSI
	if !out.IsTerminal() {
		headerColor = noColor{}
		topNameColor = noColor{}
		normalColor = noColor{}
		greenColor = noColor{}
		untaggedColor = noColor{}
	}

	columns := buildTableColumns(int(width), greenColor)
	imageRows, anyImageHasChildren := buildTableRows(images)
	columns = formatColumnsForOutput(int(width), columns, imageRows)
	table := imageTreeTable{
		columns:        columns,
		headerColor:    headerColor,
		indexNameColor: topNameColor,
		untaggedColor:  untaggedColor,
		normalColor:    normalColor,
		spacing:        anyImageHasChildren,
	}

	table.printTable(out, imageRows)
	return nil
}

func buildTableRows(images []imagetypes.Summary) ([]ImageIndexRow, bool) {
	imageRows := make([]ImageIndexRow, 0, len(images))
	var hasChildren bool
	for _, img := range images {
		details := rowDetails{
			ID:        img.ID,
			DiskUsage: units.HumanSizeWithPrecision(float64(img.Size), 3),
			InUse:     img.Containers > 0,
		}

		var totalContent int64
		children := make([]ImageManifestRow, 0, len(img.Manifests))
		for _, im := range img.Manifests {
			if im.Kind != imagetypes.ManifestKindImage {
				continue
			}

			im := im
			sub := ImageManifestRow{
				Platform:  platforms.Format(im.ImageData.Platform),
				Available: im.Available,
				Details: rowDetails{
					ID:          im.ID,
					DiskUsage:   units.HumanSizeWithPrecision(float64(im.Size.Total), 3),
					InUse:       len(im.ImageData.Containers) > 0,
					ContentSize: units.HumanSizeWithPrecision(float64(im.Size.Content), 3),
				},
			}

			if sub.Details.InUse {
				// Mark top-level parent image as used if any of its subimages are used.
				details.InUse = true
			}

			totalContent += im.Size.Content
			children = append(children, sub)

			// Add extra spacing between images if there's at least one entry with children.
			hasChildren = true
		}

		details.ContentSize = units.HumanSizeWithPrecision(float64(totalContent), 3)

		imageRows = append(imageRows, ImageIndexRow{
			Names:    img.RepoTags,
			Details:  details,
			Children: children,
			created:  img.Created,
		})
	}

	sort.Slice(imageRows, func(i, j int) bool {
		return imageRows[i].created > imageRows[j].created
	})

	return imageRows, hasChildren
}

func buildTableColumns(ttyWidth int, usedColumnColor aec.ANSI) []imgColumn {
	columns := []imgColumn{
		{
			Title: "Image",
			Align: alignLeft,
			Width: 0,
		},
		{
			Title: "ID",
			Align: alignLeft,
			Width: 12,
			DetailsValue: func(d *rowDetails) string {
				return stringid.TruncateID(d.ID)
			},
		},
		{
			Title: "Disk usage",
			Align: alignRight,
			Width: 10,
			DetailsValue: func(d *rowDetails) string {
				return d.DiskUsage
			},
		},
		{
			Title: "Content size",
			Align: alignRight,
			Width: 12,
			DetailsValue: func(d *rowDetails) string {
				return d.ContentSize
			},
		},
		{
			Title: "In Use",
			Align: alignCenter,
			Width: 6,
			Color: &usedColumnColor,
			DetailsValue: func(d *rowDetails) string {
				if d.InUse {
					return "✔"
				}
				return " "
			},
		},
	}

	return columns
}
