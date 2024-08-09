package image

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/docker/cli/cli/command"
	"github.com/morikuni/aec"

	"github.com/containerd/platforms"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-units"
)

type treeOptions struct {
	all     bool
	filters filters.Args
}

func runTree(ctx context.Context, dockerCLI command.Cli, opts treeOptions) error {
	images, err := dockerCLI.Client().ImageList(ctx, imagetypes.ListOptions{
		All:            opts.all,
		ContainerCount: true,
		Filters:        opts.filters,
		Manifests:      true,
	})
	if err != nil {
		return err
	}

	view := make([]topImage, 0, len(images))
	for _, img := range images {
		details := imageDetails{
			ID:        img.ID,
			DiskUsage: units.HumanSizeWithPrecision(float64(img.Size), 3),
			Used:      img.Containers > 0,
		}

		children := make([]subImage, 0, len(img.Manifests))
		for _, im := range img.Manifests {
			if im.Kind != imagetypes.ManifestKindImage {
				continue
			}

			imgData := im.ImageData
			platform := imgData.Platform

			sub := subImage{
				Platform:  platforms.Format(platform),
				Available: im.Available,
				Details: imageDetails{
					ID:        im.ID,
					DiskUsage: units.HumanSizeWithPrecision(float64(im.Size.Total), 3),
					Used:      len(imgData.Containers) > 0,
				},
			}

			children = append(children, sub)
		}

		view = append(view, topImage{
			Names:    img.RepoTags,
			Details:  details,
			Children: children,
		})
	}

	return printImageTree(dockerCLI, view)
}

type imageDetails struct {
	ID        string
	DiskUsage string
	Used      bool
}

type topImage struct {
	Names    []string
	Details  imageDetails
	Children []subImage
}

type subImage struct {
	Platform  string
	Available bool
	Details   imageDetails
}

const headerSpacing = 3

//nolint:gocyclo
func printImageTree(dockerCLI command.Cli, images []topImage) error {
	out := dockerCLI.Out()
	_, width := out.GetTtySize()
	if width == 0 {
		width = 80
	}
	if width < 20 {
		width = 20
	}

	headers := []header{
		{Title: "Image", Width: 0, Left: true},
		{Title: "ID", Width: 12},
		{Title: "Disk usage", Width: 10},
		{Title: "Used", Width: 4},
	}

	nameWidth := int(width)
	for idx, h := range headers {
		if h.Width == 0 {
			continue
		}
		d := h.Width + headerSpacing
		// If the first column gets too short, remove remaining columns
		if nameWidth-d < 12 {
			headers = headers[:idx]
			break
		}
		nameWidth -= d
	}

	maxImageName := len(headers[0].Title)
	for _, img := range images {
		for _, name := range img.Names {
			if len(name) > maxImageName {
				maxImageName = len(name)
			}
		}
		for _, sub := range img.Children {
			if len(sub.Platform) > maxImageName {
				maxImageName = len(sub.Platform)
			}
		}
	}

	if nameWidth > maxImageName+headerSpacing {
		nameWidth = maxImageName + headerSpacing
	}

	if nameWidth < 0 {
		headers = headers[:1]
		nameWidth = int(width)
	}
	headers[0].Width = nameWidth

	headerColor := aec.NewBuilder(aec.DefaultF, aec.Bold).ANSI
	topNameColor := aec.NewBuilder(aec.BlueF, aec.Underline, aec.Bold).ANSI
	normalColor := aec.NewBuilder(aec.DefaultF).ANSI
	normalFaintedColor := aec.NewBuilder(aec.DefaultF).Faint().ANSI
	greenColor := aec.NewBuilder(aec.GreenF).ANSI
	if !out.IsTerminal() {
		headerColor = noColor{}
		topNameColor = noColor{}
		normalColor = noColor{}
		normalFaintedColor = noColor{}
		greenColor = noColor{}
	}

	// Print headers
	for i, h := range headers {
		if i > 0 {
			_, _ = fmt.Fprint(out, strings.Repeat(" ", headerSpacing))
		}

		_, _ = fmt.Fprint(out, h.PrintC(headerColor, h.Title))
	}

	_, _ = fmt.Fprintln(out)

	printDetails := func(clr aec.ANSI, details imageDetails) {
		if len(headers) <= 1 {
			return
		}
		truncID := stringid.TruncateID(details.ID)
		_, _ = fmt.Fprint(out, headers[1].Print(clr, truncID))
		_, _ = fmt.Fprint(out, strings.Repeat(" ", headerSpacing))

		if len(headers) <= 2 {
			return
		}
		_, _ = fmt.Fprint(out, headers[2].Print(clr, details.DiskUsage))
		_, _ = fmt.Fprint(out, strings.Repeat(" ", headerSpacing))

		if len(headers) <= 3 {
			return
		}
		if details.Used {
			_, _ = fmt.Fprint(out, headers[3].Print(greenColor, "✔"))
		} else {
			_, _ = fmt.Fprint(out, headers[3].Print(clr, " "))
		}
		_, _ = fmt.Fprint(out, strings.Repeat(" ", headerSpacing))
	}

	// Print images
	for _, img := range images {
		_, _ = fmt.Fprintln(out, "")
		for nameIdx, name := range img.Names {
			if nameIdx != 0 {
				_, _ = fmt.Fprintln(out, "")
			}
			_, _ = fmt.Fprint(out, headers[0].Print(topNameColor, name))
		}
		_, _ = fmt.Fprint(out, strings.Repeat(" ", headerSpacing))

		printDetails(normalColor, img.Details)

		_, _ = fmt.Fprintln(out, "")
		for idx, sub := range img.Children {
			clr := normalColor
			if !sub.Available {
				clr = normalFaintedColor
			}

			if idx != len(img.Children)-1 {
				_, _ = fmt.Fprint(out, headers[0].Print(clr, "├─ "+sub.Platform))
			} else {
				_, _ = fmt.Fprint(out, headers[0].Print(clr, "└─ "+sub.Platform))
			}

			_, _ = fmt.Fprint(out, strings.Repeat(" ", headerSpacing))
			printDetails(clr, sub.Details)

			_, _ = fmt.Fprintln(out, "")
		}
	}

	return nil
}

type header struct {
	Title string
	Width int
	Left  bool
}

func truncateRunes(s string, length int) string {
	runes := []rune(s)
	if len(runes) > length {
		return string(runes[:length])
	}
	return s
}

func (h header) Print(clr aec.ANSI, s string) (out string) {
	if h.Left {
		return h.PrintL(clr, s)
	}
	return h.PrintC(clr, s)
}

func (h header) PrintC(clr aec.ANSI, s string) (out string) {
	ln := utf8.RuneCountInString(s)
	if h.Left {
		return h.PrintL(clr, s)
	}

	if ln > h.Width {
		return clr.Apply(truncateRunes(s, h.Width))
	}

	fill := h.Width - ln

	l := fill / 2
	r := fill - l

	return strings.Repeat(" ", l) + clr.Apply(s) + strings.Repeat(" ", r)
}

func (h header) PrintL(clr aec.ANSI, s string) string {
	ln := utf8.RuneCountInString(s)
	if ln > h.Width {
		return clr.Apply(truncateRunes(s, h.Width))
	}

	return clr.Apply(s) + strings.Repeat(" ", h.Width-ln)
}

type noColor struct{}

func (a noColor) With(ansi ...aec.ANSI) aec.ANSI {
	return aec.NewBuilder(ansi...).ANSI
}

func (a noColor) Apply(s string) string {
	return s
}

func (a noColor) String() string {
	return ""
}
