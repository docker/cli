package interpolation

import (
	"os"

	"strings"

	"github.com/docker/cli/cli/compose/template"
	"github.com/pkg/errors"
)

// Options supported by Interpolate
type Options struct {
	// SectionName of the configuration section
	SectionName string
	// LookupValue from a key
	LookupValue LookupValue
	// TypeCastMapping maps key paths to functions to cast to a type
	TypeCastMapping map[Path]Cast
}

// LookupValue is a function which maps from variable names to values.
// Returns the value as a string and a bool indicating whether
// the value is present, to distinguish between an empty string
// and the absence of a value.
type LookupValue func(key string) (string, bool)

// Cast a value to a new type, or return an error if the value can't be cast
type Cast func(value string) (interface{}, error)

// Interpolate replaces variables in a string with the values from a mapping
func Interpolate(config map[string]interface{}, opts Options) (map[string]interface{}, error) {
	out := map[string]interface{}{}

	if opts.LookupValue == nil {
		opts.LookupValue = os.LookupEnv
	}
	if opts.TypeCastMapping == nil {
		opts.TypeCastMapping = make(map[Path]Cast)
	}

	for key, item := range config {
		if item == nil {
			out[key] = nil
			continue
		}
		mapItem, ok := item.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("Invalid type for %s : %T instead of %T", key, item, out)
		}
		interpolatedItem, err := interpolateSectionItem(NewPath(key), mapItem, opts)
		if err != nil {
			return nil, err
		}
		out[key] = interpolatedItem
	}

	return out, nil
}

func interpolateSectionItem(
	path Path,
	item map[string]interface{},
	opts Options,
) (map[string]interface{}, error) {
	out := map[string]interface{}{}

	for key, value := range item {
		interpolatedValue, err := recursiveInterpolate(value, path.Next(key), opts)
		switch err := err.(type) {
		case nil:
		case *template.InvalidTemplateError:
			return nil, errors.Errorf(
				"Invalid interpolation format for %#v option in %s %#v: %#v. You may need to escape any $ with another $.",
				key, opts.SectionName, path.root(), err.Template,
			)
		default:
			return nil, errors.Wrapf(err, "error while interpolating %s in %s %s", key, opts.SectionName, path.root())
		}
		out[key] = interpolatedValue
	}

	return out, nil
}

func recursiveInterpolate(value interface{}, path Path, opts Options) (interface{}, error) {
	switch value := value.(type) {

	case string:
		newValue, err := template.Substitute(value, template.Mapping(opts.LookupValue))
		if err != nil || newValue == value {
			return value, err
		}
		caster, ok := opts.getCasterForPath(path)
		if !ok {
			return newValue, nil
		}
		return caster(newValue)

	case map[string]interface{}:
		out := map[string]interface{}{}
		for key, elem := range value {
			interpolatedElem, err := recursiveInterpolate(elem, path.Next(key), opts)
			if err != nil {
				return nil, err
			}
			out[key] = interpolatedElem
		}
		return out, nil

	case []interface{}:
		out := make([]interface{}, len(value))
		for i, elem := range value {
			interpolatedElem, err := recursiveInterpolate(elem, path, opts)
			if err != nil {
				return nil, err
			}
			out[i] = interpolatedElem
		}
		return out, nil

	default:
		return value, nil

	}
}

const pathSeparator = "."

// PathMatchAll is a token used as part of a Path to match any key at that level
// in the nested structure
const PathMatchAll = "*"

// Path is a dotted path of keys to a value in a nested mapping structure. A *
// section in a path will match any key in the mapping structure.
type Path string

// NewPath returns a new Path
func NewPath(items ...string) Path {
	return Path(strings.Join(items, pathSeparator))
}

// Next returns a new path by append part to the current path
func (p Path) Next(part string) Path {
	return Path(string(p) + pathSeparator + part)
}

func (p Path) root() string {
	parts := p.parts()
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (p Path) parts() []string {
	return strings.Split(string(p), pathSeparator)
}

func (p Path) matches(pattern Path) bool {
	patternParts := pattern.parts()
	parts := p.parts()

	if len(patternParts) != len(parts) {
		return false
	}
	for index, part := range parts {
		switch patternParts[index] {
		case PathMatchAll, part:
			continue
		default:
			return false
		}
	}
	return true
}

func (o Options) getCasterForPath(path Path) (Cast, bool) {
	for pattern, caster := range o.TypeCastMapping {
		if path.matches(pattern) {
			return caster, true
		}
	}
	return nil, false
}
