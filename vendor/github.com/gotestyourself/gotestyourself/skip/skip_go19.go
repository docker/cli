// +build go1.9

package skip

import "strings"

func getSourceLinesRange(line int, lines int) (int, int) {
	lastLine := line + maxContextLines
	if lastLine > lines {
		lastLine = lines
	}
	return line - 1, lastLine
}

func getSource(lines []string, i int) string {
	return strings.Join(lines[:i], "\n")
}
