package formatter

import (
	"unicode/utf8"

	"golang.org/x/text/width"
)

// charWidth returns the number of horizontal positions a character occupies,
// and is used to account for wide characters when displaying strings.
//
// In a broad sense, wide characters include East Asian Wide, East Asian Full-width,
// (when not in East Asian context) see http://unicode.org/reports/tr11/.
func charWidth(r rune) int {
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	default:
		return 1
	}
}

// Ellipsis truncates a string to fit within maxDisplayWidth, and appends ellipsis (…).
// For maxDisplayWidth of 1 and lower, no ellipsis is appended.
// For maxDisplayWidth of 1, first char of string will return even if its width > 1.
func Ellipsis(s string, maxDisplayWidth int) string {
	if maxDisplayWidth <= 0 {
		return ""
	}
	rs := []rune(s)
	if maxDisplayWidth == 1 {
		return string(rs[0])
	}

	byteLen := len(s)
	if byteLen == utf8.RuneCountInString(s) {
		if byteLen <= maxDisplayWidth {
			return s
		}
		return string(rs[:maxDisplayWidth-1]) + "…"
	}

	var (
		display      = make([]int, 0, len(rs))
		displayWidth int
	)
	for _, r := range rs {
		cw := charWidth(r)
		displayWidth += cw
		display = append(display, displayWidth)
	}
	if displayWidth <= maxDisplayWidth {
		return s
	}
	for i := range display {
		if display[i] <= maxDisplayWidth-1 && display[i+1] > maxDisplayWidth-1 {
			return string(rs[:i+1]) + "…"
		}
	}
	return s
}
