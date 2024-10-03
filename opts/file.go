package opts

import (
	"bufio"
	"bytes"
	"fmt"
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

	lines := []string{}
	scanner := bufio.NewScanner(fh)
	utf8bom := []byte{0xEF, 0xBB, 0xBF}
	for currentLine := 1; scanner.Scan(); currentLine++ {
		scannedBytes := scanner.Bytes()
		if !utf8.Valid(scannedBytes) {
			return []string{}, fmt.Errorf("env file %s contains invalid utf8 bytes at line %d: %v", filename, currentLine, scannedBytes)
		}
		// We trim UTF8 BOM
		if currentLine == 1 {
			scannedBytes = bytes.TrimPrefix(scannedBytes, utf8bom)
		}
		// trim the line from all leading whitespace first. trailing whitespace
		// is part of the value, and is kept unmodified.
		line := strings.TrimLeftFunc(string(scannedBytes), unicode.IsSpace)

		if len(line) == 0 || line[0] == '#' {
			// skip empty lines and comments (lines starting with '#')
			continue
		}

		key, _, hasValue := strings.Cut(line, "=")
		if len(key) == 0 {
			return []string{}, fmt.Errorf("no variable name on line '%s'", line)
		}

		// leading whitespace was already removed from the line, but
		// variables are not allowed to contain whitespace or have
		// trailing whitespace.
		if strings.ContainsAny(key, whiteSpaces) {
			return []string{}, fmt.Errorf("variable '%s' contains whitespaces", key)
		}

		if hasValue {
			// key/value pair is valid and has a value; add the line as-is.
			lines = append(lines, line)
			continue
		}

		if lookupFn != nil {
			// No value given; try to look up the value. The value may be
			// empty but if no value is found, the key is omitted.
			if value, found := lookupFn(line); found {
				lines = append(lines, key+"="+value)
			}
		}
	}
	return lines, scanner.Err()
}
