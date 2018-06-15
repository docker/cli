package template

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	delimiter    = "\\$"
	substitution = "[_a-z.][_a-z0-9.]*(?::?[-?][^}]*)?"

	patternString = fmt.Sprintf(
		"%s(?i:(?P<escaped>%s)|(?P<named>%s)|{(?P<braced>%s)}|(?P<invalid>))",
		delimiter, delimiter, substitution, substitution,
	)

	pattern = regexp.MustCompile(patternString)
)

// InvalidTemplateError is returned when a variable template is not in a valid
// format
type InvalidTemplateError struct {
	Template string
}

func (e InvalidTemplateError) Error() string {
	return fmt.Sprintf("Invalid template: %#v", e.Template)
}

// Mapping is a user-supplied function which maps from variable names to values.
// Returns the value as a string and a bool indicating whether
// the value is present, to distinguish between an empty string
// and the absence of a value.
type Mapping func(string) (string, bool)

// Substitute variables in the string with their values
func Substitute(template string, mapping Mapping) (string, error) {
	var err error
	result := pattern.ReplaceAllStringFunc(template, func(substring string) string {
		matches := pattern.FindStringSubmatch(substring)
		groups := matchGroups(matches)
		if escaped := groups["escaped"]; escaped != "" {
			return escaped
		}

		substitution := groups["named"]
		if substitution == "" {
			substitution = groups["braced"]
		}

		switch {

		case substitution == "":
			err = &InvalidTemplateError{Template: template}
			return ""

		// Soft default (fall back if unset or empty)
		case strings.Contains(substitution, ":-"):
			name, defaultValue := partition(substitution, ":-")
			value, ok := mapping(name)
			if !ok || value == "" {
				return defaultValue
			}
			return value

		// Hard default (fall back if-and-only-if empty)
		case strings.Contains(substitution, "-"):
			name, defaultValue := partition(substitution, "-")
			value, ok := mapping(name)
			if !ok {
				return defaultValue
			}
			return value

		case strings.Contains(substitution, ":?"):
			name, errorMessage := partition(substitution, ":?")
			value, ok := mapping(name)
			if !ok || value == "" {
				err = &InvalidTemplateError{
					Template: fmt.Sprintf("required variable %s is missing a value: %s", name, errorMessage),
				}
				return ""
			}
			return value

		case strings.Contains(substitution, "?"):
			name, errorMessage := partition(substitution, "?")
			value, ok := mapping(name)
			if !ok {
				err = &InvalidTemplateError{
					Template: fmt.Sprintf("required variable %s is missing a value: %s", name, errorMessage),
				}
				return ""
			}
			return value
		}

		value, _ := mapping(substitution)
		return value
	})

	return result, err
}

func matchGroups(matches []string) map[string]string {
	groups := make(map[string]string)
	for i, name := range pattern.SubexpNames()[1:] {
		groups[name] = matches[i+1]
	}
	return groups
}

// Split the string at the first occurrence of sep, and return the part before the separator,
// and the part after the separator.
//
// If the separator is not found, return the string itself, followed by an empty string.
func partition(s, sep string) (string, string) {
	if strings.Contains(s, sep) {
		parts := strings.SplitN(s, sep, 2)
		return parts[0], parts[1]
	}
	return s, ""
}
