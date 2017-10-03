package interpolation

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/env"
	"github.com/stretchr/testify/assert"
)

var defaults = map[string]string{
	"USER": "jenny",
	"FOO":  "bar",
}

func defaultMapping(name string) (string, bool) {
	val, ok := defaults[name]
	return val, ok
}

func TestInterpolate(t *testing.T) {
	services := map[string]interface{}{
		"servicea": map[string]interface{}{
			"image":   "example:${USER}",
			"volumes": []interface{}{"$FOO:/target"},
			"logging": map[string]interface{}{
				"driver": "${FOO}",
				"options": map[string]interface{}{
					"user": "$USER",
				},
			},
		},
	}
	expected := map[string]interface{}{
		"servicea": map[string]interface{}{
			"image":   "example:jenny",
			"volumes": []interface{}{"bar:/target"},
			"logging": map[string]interface{}{
				"driver": "bar",
				"options": map[string]interface{}{
					"user": "jenny",
				},
			},
		},
	}
	result, err := Interpolate(services, Options{
		SectionName: "service",
		LookupValue: defaultMapping,
	})
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestInvalidInterpolation(t *testing.T) {
	services := map[string]interface{}{
		"servicea": map[string]interface{}{
			"image": "${",
		},
	}
	_, err := Interpolate(services, Options{
		SectionName: "service",
		LookupValue: defaultMapping,
	})
	assert.EqualError(t, err, `Invalid interpolation format for "image" option in service "servicea": "${". You may need to escape any $ with another $.`)
}

func TestInterpolateWithDefaults(t *testing.T) {
	defer env.Patch(t, "FOO", "BARZ")()

	config := map[string]interface{}{
		"networks": map[string]interface{}{
			"foo": "thing_${FOO}",
		},
	}
	expected := map[string]interface{}{
		"networks": map[string]interface{}{
			"foo": "thing_BARZ",
		},
	}
	result, err := Interpolate(config, Options{})
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}
