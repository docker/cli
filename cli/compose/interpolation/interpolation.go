package interpolation

import (
	"os"

	"github.com/docker/cli/cli/compose/template"
	"github.com/pkg/errors"
)

// Options supported by Interpolate
type Options struct {
	// SectionName of the configuration section
	SectionName string
	// LookupValue from a key
	LookupValue LookupValue
}

// LookupValue is a function which maps from variable names to values.
// Returns the value as a string and a bool indicating whether
// the value is present, to distinguish between an empty string
// and the absence of a value.
type LookupValue func(key string) (string, bool)

// Interpolate replaces variables in a string with the values from a mapping
func Interpolate(config map[string]interface{}, opts Options) (map[string]interface{}, error) {
	out := map[string]interface{}{}

	if opts.LookupValue == nil {
		opts.LookupValue = os.LookupEnv
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
		interpolatedItem, err := interpolateSectionItem(key, mapItem, opts)
		if err != nil {
			return nil, err
		}
		out[key] = interpolatedItem
	}

	return out, nil
}

func interpolateSectionItem(
	sectionkey string,
	item map[string]interface{},
	opts Options,
) (map[string]interface{}, error) {
	out := map[string]interface{}{}

	for key, value := range item {
		interpolatedValue, err := recursiveInterpolate(value, opts)
		switch err := err.(type) {
		case nil:
		case *template.InvalidTemplateError:
			return nil, errors.Errorf(
				"Invalid interpolation format for %#v option in %s %#v: %#v. You may need to escape any $ with another $.",
				key, opts.SectionName, sectionkey, err.Template,
			)
		default:
			return nil, errors.Wrapf(err, "error while interpolating %s in %s %s", key, opts.SectionName, sectionkey)
		}
		out[key] = interpolatedValue
	}

	return out, nil
}

func recursiveInterpolate(value interface{}, opts Options) (interface{}, error) {
	switch value := value.(type) {

	case string:
		return template.Substitute(value, template.Mapping(opts.LookupValue))

	case map[string]interface{}:
		out := map[string]interface{}{}
		for key, elem := range value {
			interpolatedElem, err := recursiveInterpolate(elem, opts)
			if err != nil {
				return nil, err
			}
			out[key] = interpolatedElem
		}
		return out, nil

	case []interface{}:
		out := make([]interface{}, len(value))
		for i, elem := range value {
			interpolatedElem, err := recursiveInterpolate(elem, opts)
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
