package image

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/morikuni/aec"
)

type imageTreeTable struct {
	columns []imgColumn

	headerColor    aec.ANSI
	indexNameColor aec.ANSI
	untaggedColor  aec.ANSI
	normalColor    aec.ANSI
	highlightColor aec.ANSI

	spacing bool
}

type ImageIndexRow struct {
	Names    []string
	Details  rowDetails
	Children []ImageManifestRow

	created int64
}

type ImageManifestRow struct {
	Platform  string
	Available bool
	Highlight bool
	Details   rowDetails
}

// rowDetails is used by both ImageIndexRow and ImageManifestRow
type rowDetails struct {
	ID          string
	DiskUsage   string
	InUse       bool
	ContentSize string
}

func (t *imageTreeTable) printTable(out io.Writer, imgs []ImageIndexRow) {
	t.printHeaders(out)
	for _, img := range imgs {
		t.printIndex(out, img)
		_, _ = fmt.Fprintln(out)
	}
}

func (t *imageTreeTable) printHeaders(out io.Writer) {
	for i, h := range t.columns {
		if i > 0 {
			_, _ = fmt.Fprint(out, strings.Repeat(" ", columnSpacing))
		}

		_, _ = fmt.Fprint(out, h.Print(t.headerColor, strings.ToUpper(h.Title)))
	}

	_, _ = fmt.Fprintln(out)
}

func (t *imageTreeTable) printIndex(out io.Writer, img ImageIndexRow) {
	// print the names for the index
	printNames(out, t.columns, img, t.indexNameColor, t.untaggedColor)
	// print the rest of the columns/details for the header
	printDetails(out, t.columns, t.normalColor, img.Details)

	// print the manifest rows, with their details
	if len(img.Children) > 0 || t.spacing {
		_, _ = fmt.Fprintln(out)
	}

	printChildren(out, t.columns, img, t.normalColor)
}

func printDetails(out io.Writer, headers []imgColumn, defaultColor aec.ANSI, details rowDetails) {
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

func printChildren(out io.Writer, headers []imgColumn, img ImageIndexRow, normalColor aec.ANSI) {
	for idx, sub := range img.Children {
		clr := normalColor
		if !sub.Available {
			clr = normalColor.With(aec.Faint)
		}
		if sub.Highlight {
			clr = normalColor.With(aec.Bold)
		}

		if idx != len(img.Children)-1 {
			_, _ = fmt.Fprint(out, headers[0].Print(clr, "├─ "+sub.Platform))
		} else {
			_, _ = fmt.Fprint(out, headers[0].Print(clr, "└─ "+sub.Platform))
		}

		printDetails(out, headers, clr, sub.Details)
		_, _ = fmt.Fprintln(out)
	}
}

func printNames(out io.Writer, headers []imgColumn, img ImageIndexRow, color, untaggedColor aec.ANSI) {
	if len(img.Names) == 0 {
		_, _ = fmt.Fprint(out, headers[0].Print(untaggedColor, "<untagged>"))
	}

	for nameIdx, name := range img.Names {
		if nameIdx != 0 {
			_, _ = fmt.Fprintln(out)
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

	DetailsValue func(*rowDetails) string
	Color        *aec.ANSI
}

// formatColumnsForOutput resizes the table columns for the provided tty
// size. The first column is made as narrow as possible, and columns are
// removed from the table output if the tty is not wide enough to
// accomodate the entire table.
func formatColumnsForOutput(ttyWidth int, columns []imgColumn, images []ImageIndexRow) []imgColumn {
	nameWidth := ttyWidth
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

	// Try to make the first column as narrow as possible
	widest := widestFirstColumnValue(columns, images)
	if nameWidth > widest {
		nameWidth = widest
	}
	columns[0].Width = nameWidth

	return columns
}

const columnSpacing = 3

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
func widestFirstColumnValue(headers []imgColumn, images []ImageIndexRow) int {
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
