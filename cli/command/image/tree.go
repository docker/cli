package image

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/containerd/platforms"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/streams"
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

type treeView struct {
	images []topImage

	// imageSpacing indicates whether there should be extra spacing between images.
	imageSpacing bool
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

	view := treeView{
		images: make([]topImage, 0, len(images)),
	}
	for _, img := range images {
		details := imageDetails{
			ID:        img.ID,
			DiskUsage: units.HumanSizeWithPrecision(float64(img.Size), 3),
			InUse:     img.Containers > 0,
		}

		var totalContent int64
		children := make([]subImage, 0, len(img.Manifests))
		for _, im := range img.Manifests {
			if im.Kind != imagetypes.ManifestKindImage {
				continue
			}

			im := im
			sub := subImage{
				Platform:  platforms.Format(im.ImageData.Platform),
				Available: im.Available,
				Details: imageDetails{
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
			view.imageSpacing = true
		}

		details.ContentSize = units.HumanSizeWithPrecision(float64(totalContent), 3)

		view.images = append(view.images, topImage{
			Names:    img.RepoTags,
			Details:  details,
			Children: children,
			created:  img.Created,
		})
	}

	sort.Slice(view.images, func(i, j int) bool {
		return view.images[i].created > view.images[j].created
	})

	return printImageTree(dockerCLI, view)
}

type imageDetails struct {
	ID          string
	DiskUsage   string
	InUse       bool
	ContentSize string
}

type topImage struct {
	Names    []string
	Details  imageDetails
	Children []subImage

	created int64
}

type subImage struct {
	Platform  string
	Available bool
	Details   imageDetails
}

const columnSpacing = 3

func printImageTree(dockerCLI command.Cli, view treeView) error {
	out := dockerCLI.Out()
	_, width := out.GetTtySize()
	if width == 0 {
		width = 80
	}
	if width < 20 {
		width = 20
	}

	warningColor := aec.LightYellowF
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
		warningColor = noColor{}
		untaggedColor = noColor{}
	}

	_, _ = fmt.Fprintln(out, warningColor.Apply("WARNING: This is an experimental feature. The output may change and shouldn't be depended on."))
	_, _ = fmt.Fprintln(out, "")

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
			DetailsValue: func(d *imageDetails) string {
				return stringid.TruncateID(d.ID)
			},
		},
		{
			Title: "Disk usage",
			Align: alignRight,
			Width: 10,
			DetailsValue: func(d *imageDetails) string {
				return d.DiskUsage
			},
		},
		{
			Title: "Content size",
			Align: alignRight,
			Width: 12,
			DetailsValue: func(d *imageDetails) string {
				return d.ContentSize
			},
		},
		{
			Title: "In Use",
			Align: alignCenter,
			Width: 6,
			Color: &greenColor,
			DetailsValue: func(d *imageDetails) string {
				if d.InUse {
					return "✔"
				}
				return " "
			},
		},
	}

	nameWidth := int(width)
	for idx, h := range columns {
		if h.Width == 0 {
			continue
		}
		d := h.Width
		if idx > 0 {
			d += columnSpacing
		}
		// If the first column gets too short, remove remaining columns
		if nameWidth-d < 12 {
			columns = columns[:idx]
			break
		}
		nameWidth -= d
	}

	images := view.images
	// Try to make the first column as narrow as possible
	widest := widestFirstColumnValue(columns, images)
	if nameWidth > widest {
		nameWidth = widest
	}
	columns[0].Width = nameWidth

	// Print columns
	for i, h := range columns {
		if i > 0 {
			_, _ = fmt.Fprint(out, strings.Repeat(" ", columnSpacing))
		}

		_, _ = fmt.Fprint(out, h.Print(headerColor, strings.ToUpper(h.Title)))
	}

	_, _ = fmt.Fprintln(out)

	// Print images
	for _, img := range images {
		printNames(out, columns, img, topNameColor, untaggedColor)
		printDetails(out, columns, normalColor, img.Details)

		if len(img.Children) > 0 || view.imageSpacing {
			_, _ = fmt.Fprintln(out)
		}
		printChildren(out, columns, img, normalColor)
		_, _ = fmt.Fprintln(out)
	}

	return nil
}

func printDetails(out *streams.Out, headers []imgColumn, defaultColor aec.ANSI, details imageDetails) {
	for _, h := range headers {
		if h.DetailsValue == nil {
			continue
		}

		_, _ = fmt.Fprint(out, strings.Repeat(" ", columnSpacing))
		clr := defaultColor
		if h.Color != nil {
			clr = *h.Color
		}
		val := h.DetailsValue(&details)
		_, _ = fmt.Fprint(out, h.Print(clr, val))
	}
}

func printChildren(out *streams.Out, headers []imgColumn, img topImage, normalColor aec.ANSI) {
	for idx, sub := range img.Children {
		clr := normalColor
		if !sub.Available {
			clr = normalColor.With(aec.Faint)
		}

		if idx != len(img.Children)-1 {
			_, _ = fmt.Fprint(out, headers[0].Print(clr, "├─ "+sub.Platform))
		} else {
			_, _ = fmt.Fprint(out, headers[0].Print(clr, "└─ "+sub.Platform))
		}

		printDetails(out, headers, clr, sub.Details)
		_, _ = fmt.Fprintln(out, "")
	}
}

func printNames(out *streams.Out, headers []imgColumn, img topImage, color, untaggedColor aec.ANSI) {
	if len(img.Names) == 0 {
		_, _ = fmt.Fprint(out, headers[0].Print(untaggedColor, "<untagged>"))
	}

	for nameIdx, name := range img.Names {
		if nameIdx != 0 {
			_, _ = fmt.Fprintln(out, "")
		}
		_, _ = fmt.Fprint(out, headers[0].Print(color, name))
	}
}

type alignment int

const (
	alignLeft alignment = iota
	alignCenter
	alignRight
)

type imgColumn struct {
	Title string
	Width int
	Align alignment

	DetailsValue func(*imageDetails) string
	Color        *aec.ANSI
}

func truncateRunes(s string, length int) string {
	runes := []rune(s)
	if len(runes) > length {
		return string(runes[:length-3]) + "..."
	}
	return s
}

func (h imgColumn) Print(clr aec.ANSI, s string) string {
	switch h.Align {
	case alignCenter:
		return h.PrintC(clr, s)
	case alignRight:
		return h.PrintR(clr, s)
	case alignLeft:
	}
	return h.PrintL(clr, s)
}

func (h imgColumn) PrintC(clr aec.ANSI, s string) string {
	ln := utf8.RuneCountInString(s)

	if ln > h.Width {
		return clr.Apply(truncateRunes(s, h.Width))
	}

	fill := h.Width - ln

	l := fill / 2
	r := fill - l

	return strings.Repeat(" ", l) + clr.Apply(s) + strings.Repeat(" ", r)
}

func (h imgColumn) PrintL(clr aec.ANSI, s string) string {
	ln := utf8.RuneCountInString(s)
	if ln > h.Width {
		return clr.Apply(truncateRunes(s, h.Width))
	}

	return clr.Apply(s) + strings.Repeat(" ", h.Width-ln)
}

func (h imgColumn) PrintR(clr aec.ANSI, s string) string {
	ln := utf8.RuneCountInString(s)
	if ln > h.Width {
		return clr.Apply(truncateRunes(s, h.Width))
	}

	return strings.Repeat(" ", h.Width-ln) + clr.Apply(s)
}

type noColor struct{}

func (a noColor) With(_ ...aec.ANSI) aec.ANSI {
	return a
}

func (a noColor) Apply(s string) string {
	return s
}

func (a noColor) String() string {
	return ""
}

// widestFirstColumnValue calculates the width needed to fully display the image names and platforms.
func widestFirstColumnValue(headers []imgColumn, images []topImage) int {
	width := len(headers[0].Title)
	for _, img := range images {
		for _, name := range img.Names {
			if len(name) > width {
				width = len(name)
			}
		}
		for _, sub := range img.Children {
			pl := len(sub.Platform) + len("└─ ")
			if pl > width {
				width = pl
			}
		}
	}
	return width
}
