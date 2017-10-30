package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type dict map[string]interface{}

func TestValidate(t *testing.T) {
	config := dict{
		"version": "3.0",
		"services": dict{
			"foo": dict{
				"image": "busybox",
			},
		},
	}

	assert.NoError(t, Validate(config, "3.0"))
}

func TestValidateUndefinedTopLevelOption(t *testing.T) {
	config := dict{
		"version": "3.0",
		"helicopters": dict{
			"foo": dict{
				"image": "busybox",
			},
		},
	}

	err := Validate(config, "3.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Additional property helicopters is not allowed")
}

func TestValidateAllowsXTopLevelFields(t *testing.T) {
	config := dict{
		"version":       "3.4",
		"x-extra-stuff": dict{},
	}

	err := Validate(config, "3.4")
	assert.NoError(t, err)
}

func TestValidateAllowsXBelowServices(t *testing.T) {
	config := dict{
		"version": "3.4",
		"services": dict{
			"foo": dict{
				"image":   "busybox",
				"x-extra": dict{},
			},
		},
		"x-extra-stuff": dict{},
	}

	err := Validate(config, "3.4")
	assert.NoError(t, err)
}

func TestValidateInvalidVersion(t *testing.T) {
	config := dict{
		"version": "2.1",
		"services": dict{
			"foo": dict{
				"image": "busybox",
			},
		},
	}

	err := Validate(config, "2.1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported Compose file version: 2.1")
}

type array []interface{}

func TestValidatePlacement(t *testing.T) {
	config := dict{
		"version": "3.3",
		"services": dict{
			"foo": dict{
				"image": "busybox",
				"deploy": dict{
					"placement": dict{
						"preferences": array{
							dict{
								"spread": "node.labels.az",
							},
						},
					},
				},
			},
		},
	}

	assert.NoError(t, Validate(config, "3.3"))
}
