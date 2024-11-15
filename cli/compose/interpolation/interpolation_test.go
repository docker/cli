// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package interpolation

import (
	"strconv"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

var defaults = map[string]string{
	"USER":  "jenny",
	"FOO":   "bar",
	"count": "5",
}

func defaultMapping(name string) (string, bool) {
	val, ok := defaults[name]
	return val, ok
}

func TestInterpolate(t *testing.T) {
	services := map[string]any{
		"servicea": map[string]any{
			"image":   "example:${USER}",
			"volumes": []any{"$FOO:/target"},
			"logging": map[string]any{
				"driver": "${FOO}",
				"options": map[string]any{
					"user": "$USER",
				},
			},
		},
	}
	expected := map[string]any{
		"servicea": map[string]any{
			"image":   "example:jenny",
			"volumes": []any{"bar:/target"},
			"logging": map[string]any{
				"driver": "bar",
				"options": map[string]any{
					"user": "jenny",
				},
			},
		},
	}
	result, err := Interpolate(services, Options{LookupValue: defaultMapping})
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, result))
}

func TestInvalidInterpolation(t *testing.T) {
	services := map[string]any{
		"servicea": map[string]any{
			"image": "${",
		},
	}
	_, err := Interpolate(services, Options{LookupValue: defaultMapping})
	assert.Error(t, err, `invalid interpolation format for servicea.image: "${"; you may need to escape any $ with another $`)
}

func TestInterpolateWithDefaults(t *testing.T) {
	t.Setenv("FOO", "BARZ")

	config := map[string]any{
		"networks": map[string]any{
			"foo": "thing_${FOO}",
		},
	}
	expected := map[string]any{
		"networks": map[string]any{
			"foo": "thing_BARZ",
		},
	}
	result, err := Interpolate(config, Options{})
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, result))
}

func TestInterpolateWithCast(t *testing.T) {
	config := map[string]any{
		"foo": map[string]any{
			"replicas": "$count",
		},
	}
	toInt := func(value string) (any, error) {
		return strconv.Atoi(value)
	}
	result, err := Interpolate(config, Options{
		LookupValue:     defaultMapping,
		TypeCastMapping: map[Path]Cast{NewPath(PathMatchAll, "replicas"): toInt},
	})
	assert.NilError(t, err)
	expected := map[string]any{
		"foo": map[string]any{
			"replicas": 5,
		},
	}
	assert.Check(t, is.DeepEqual(expected, result))
}

func TestPathMatches(t *testing.T) {
	testcases := []struct {
		doc      string
		path     Path
		pattern  Path
		expected bool
	}{
		{
			doc:     "pattern too short",
			path:    NewPath("one", "two", "three"),
			pattern: NewPath("one", "two"),
		},
		{
			doc:     "pattern too long",
			path:    NewPath("one", "two"),
			pattern: NewPath("one", "two", "three"),
		},
		{
			doc:     "pattern mismatch",
			path:    NewPath("one", "three", "two"),
			pattern: NewPath("one", "two", "three"),
		},
		{
			doc:     "pattern mismatch with match-all part",
			path:    NewPath("one", "three", "two"),
			pattern: NewPath(PathMatchAll, "two", "three"),
		},
		{
			doc:      "pattern match with match-all part",
			path:     NewPath("one", "two", "three"),
			pattern:  NewPath("one", "*", "three"),
			expected: true,
		},
		{
			doc:      "pattern match",
			path:     NewPath("one", "two", "three"),
			pattern:  NewPath("one", "two", "three"),
			expected: true,
		},
	}
	for _, testcase := range testcases {
		assert.Check(t, is.Equal(testcase.expected, testcase.path.matches(testcase.pattern)))
	}
}
