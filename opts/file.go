package opts

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

const whiteSpaces = " \t"

func parseKeyValueFile(filename string, lookupFn func(string) (string, bool)) ([]string, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return []string{}, err
	}
	defer fh.Close()
	return ParseKeyValueFile(fh, filename, lookupFn)
}

// ParseKeyValueFile parse a file containing key,value pairs separated by equal sign
// Lines starting with `#` are ignored
// If a key is declared without a value (no equal sign), lookupFn is requested to provide value for the given key
// value is returned as-is, without any kind of parsing but removal of leading whitespace
func ParseKeyValueFile(r io.Reader, filename string, lookupFn func(string) (string, bool)) ([]string, error) {
	lines := []string{}
	scanner := bufio.NewScanner(r)
	currentLine := 0
	utf8bom := []byte{0xEF, 0xBB, 0xBF}
	for scanner.Scan() {
		scannedBytes := scanner.Bytes()
		if !utf8.Valid(scannedBytes) {
			return []string{}, fmt.Errorf("env file %s contains invalid utf8 bytes at line %d: %v", filename, currentLine+1, scannedBytes)
		}
		// We trim UTF8 BOM
		if currentLine == 0 {
			scannedBytes = bytes.TrimPrefix(scannedBytes, utf8bom)
		}
		// trim the line from all leading whitespace first
		line := strings.TrimLeftFunc(string(scannedBytes), unicode.IsSpace)
		currentLine++
		// line is not empty, and not starting with '#'
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			variable, value, hasValue := strings.Cut(line, "=")

			// trim the front of a variable, but nothing else
			variable = strings.TrimLeft(variable, whiteSpaces)
			if strings.ContainsAny(variable, whiteSpaces) {
				return []string{}, fmt.Errorf("variable '%s' contains whitespaces", variable)
			}
			if len(variable) == 0 {
				return []string{}, fmt.Errorf("no variable name on line '%s'", line)
			}

			if hasValue {
				// pass the value through, no trimming
				lines = append(lines, variable+"="+value)
			} else {
				var present bool
				if lookupFn != nil {
					value, present = lookupFn(line)
				}
				if present {
					// if only a pass-through variable is given, clean it up.
					lines = append(lines, strings.TrimSpace(variable)+"="+value)
				}
			}
		}
	}
	return lines, scanner.Err()
}
