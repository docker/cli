package schema

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
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

	assert.Check(t, Validate(config, "3.0"))
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
	assert.Check(t, is.ErrorContains(err, ""))
	assert.Check(t, is.Contains(err.Error(), "Additional property helicopters is not allowed"))
}

func TestValidateAllowsXTopLevelFields(t *testing.T) {
	config := dict{
		"version":       "3.4",
		"x-extra-stuff": dict{},
	}

	err := Validate(config, "3.4")
	assert.Check(t, err)
}

func TestValidateSecretConfigNames(t *testing.T) {
	config := dict{
		"version": "3.5",
		"configs": dict{
			"bar": dict{
				"name": "foobar",
			},
		},
		"secrets": dict{
			"baz": dict{
				"name": "foobaz",
			},
		},
	}

	err := Validate(config, "3.5")
	assert.Check(t, err)
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
	assert.Check(t, is.ErrorContains(err, ""))
	assert.Check(t, is.Contains(err.Error(), "unsupported Compose file version: 2.1"))
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

	assert.Check(t, Validate(config, "3.3"))
}

func TestValidateIsolation(t *testing.T) {
	config := dict{
		"version": "3.5",
		"services": dict{
			"foo": dict{
				"image":     "busybox",
				"isolation": "some-isolation-value",
			},
		},
	}
	assert.Check(t, Validate(config, "3.5"))
}
