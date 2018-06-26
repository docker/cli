package template

import (
	"fmt"
	"regexp"
	"strings"
)

var delimiter = "\\$"
var substitution = "[_a-z][_a-z0-9]*(?::?[-?][^}]*)?"

var patternString = fmt.Sprintf(
	"%s(?i:(?P<escaped>%s)|(?P<named>%s)|{(?P<braced>%s)}|(?P<invalid>))",
	delimiter, delimiter, substitution, substitution,
)

var pattern = regexp.MustCompile(patternString)

// DefaultSubstituteFuncs contains the default SubstitueFunc used by the docker cli
var DefaultSubstituteFuncs = []SubstituteFunc{
	softDefault,
	hardDefault,
	requiredNonEmpty,
	required,
}

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

// SubstituteFunc is a user-supplied function that apply substitution.
// Returns the value as a string, a bool indicating if the function could apply
// the substitution and an error.
type SubstituteFunc func(string, Mapping) (string, bool, error)

// SubstituteWith subsitute variables in the string with their values.
// It accepts additional substitute function.
func SubstituteWith(template string, mapping Mapping, pattern *regexp.Regexp, subsFuncs ...SubstituteFunc) (string, error) {
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

		if substitution == "" {
			err = &InvalidTemplateError{Template: template}
			return ""
		}

		for _, f := range subsFuncs {
			var (
				value   string
				applied bool
			)
			value, applied, err = f(substitution, mapping)
			if err != nil {
				return ""
			}
			if !applied {
				continue
			}
			return value
		}

		value, _ := mapping(substitution)
		return value
	})

	return result, err
}

// Substitute variables in the string with their values
func Substitute(template string, mapping Mapping) (string, error) {
	return SubstituteWith(template, mapping, pattern, DefaultSubstituteFuncs...)
}

// Soft default (fall back if unset or empty)
func softDefault(substitution string, mapping Mapping) (string, bool, error) {
	if !strings.Contains(substitution, ":-") {
		return "", false, nil
	}
	name, defaultValue := partition(substitution, ":-")
	value, ok := mapping(name)
	if !ok || value == "" {
		return defaultValue, true, nil
	}
	return value, true, nil
}

// Hard default (fall back if-and-only-if empty)
func hardDefault(substitution string, mapping Mapping) (string, bool, error) {
	if !strings.Contains(substitution, "-") {
		return "", false, nil
	}
	name, defaultValue := partition(substitution, "-")
	value, ok := mapping(name)
	if !ok {
		return defaultValue, true, nil
	}
	return value, true, nil
}

func requiredNonEmpty(substitution string, mapping Mapping) (string, bool, error) {
	if !strings.Contains(substitution, ":?") {
		return "", false, nil
	}
	name, errorMessage := partition(substitution, ":?")
	value, ok := mapping(name)
	if !ok || value == "" {
		return "", true, &InvalidTemplateError{
			Template: fmt.Sprintf("required variable %s is missing a value: %s", name, errorMessage),
		}
	}
	return value, true, nil
}

func required(substitution string, mapping Mapping) (string, bool, error) {
	if !strings.Contains(substitution, "?") {
		return "", false, nil
	}
	name, errorMessage := partition(substitution, "?")
	value, ok := mapping(name)
	if !ok {
		return "", true, &InvalidTemplateError{
			Template: fmt.Sprintf("required variable %s is missing a value: %s", name, errorMessage),
		}
	}
	return value, true, nil
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
