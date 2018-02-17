// +build !go1.9,!go.10,!go.11,!go1.12

package skip

import "strings"

func getSourceLinesRange(line int, _ int) (int, int) {
	firstLine := line - maxContextLines
	if firstLine < 0 {
		firstLine = 0
	}
	return firstLine, line
}

func getSource(lines []string, i int) string {
	return strings.Join(lines[len(lines)-i-1:], "\n")
}
