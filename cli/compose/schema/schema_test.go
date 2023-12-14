// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package schema

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
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

	assert.NilError(t, Validate(config, "3.0"))
	assert.NilError(t, Validate(config, "3"))
	assert.NilError(t, Validate(config, ""))
	assert.ErrorContains(t, Validate(config, "1.0"), "unsupported Compose file version: 1.0")
	assert.ErrorContains(t, Validate(config, "12345"), "unsupported Compose file version: 12345")
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
	assert.ErrorContains(t, err, "Additional property helicopters is not allowed")
}

func TestValidateAllowsXTopLevelFields(t *testing.T) {
	config := dict{
		"version":       "3.4",
		"x-extra-stuff": dict{},
	}

	err := Validate(config, "3.4")
	assert.NilError(t, err)
}

func TestValidateAllowsXFields(t *testing.T) {
	config := dict{
		"version": "3.7",
		"services": dict{
			"bar": dict{
				"x-extra-stuff": dict{},
			},
		},
		"volumes": dict{
			"bar": dict{
				"x-extra-stuff": dict{},
			},
		},
		"networks": dict{
			"bar": dict{
				"x-extra-stuff": dict{},
			},
		},
		"configs": dict{
			"bar": dict{
				"x-extra-stuff": dict{},
			},
		},
		"secrets": dict{
			"bar": dict{
				"x-extra-stuff": dict{},
			},
		},
	}
	err := Validate(config, "3.7")
	assert.NilError(t, err)
}

func TestValidateCredentialSpecs(t *testing.T) {
	tests := []struct {
		version     string
		expectedErr string
	}{
		{version: "3.0", expectedErr: "credential_spec"},
		{version: "3.1", expectedErr: "credential_spec"},
		{version: "3.2", expectedErr: "credential_spec"},
		{version: "3.3", expectedErr: "config"},
		{version: "3.4", expectedErr: "config"},
		{version: "3.5", expectedErr: "config"},
		{version: "3.6", expectedErr: "config"},
		{version: "3.7", expectedErr: "config"},
		{version: "3.8"},
		{version: "3.9"},
		{version: "3.10"},
		{version: "3.11"},
		{version: "3"},
		{version: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.version, func(t *testing.T) {
			config := dict{
				"version": "99.99",
				"services": dict{
					"foo": dict{
						"image": "busybox",
						"credential_spec": dict{
							"config": "foobar",
						},
					},
				},
			}
			err := Validate(config, tc.version)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, fmt.Sprintf("Additional property %s is not allowed", tc.expectedErr))
			} else {
				assert.NilError(t, err)
			}
		})
	}
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
	assert.NilError(t, err)
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
	assert.ErrorContains(t, err, "unsupported Compose file version: 2.1")
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

	assert.NilError(t, Validate(config, "3.3"))
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
	assert.NilError(t, Validate(config, "3.5"))
}

func TestValidateRollbackConfig(t *testing.T) {
	config := dict{
		"version": "3.4",
		"services": dict{
			"foo": dict{
				"image": "busybox",
				"deploy": dict{
					"rollback_config": dict{
						"parallelism": 1,
					},
				},
			},
		},
	}

	assert.NilError(t, Validate(config, "3.7"))
}

func TestValidateRollbackConfigWithOrder(t *testing.T) {
	config := dict{
		"version": "3.4",
		"services": dict{
			"foo": dict{
				"image": "busybox",
				"deploy": dict{
					"rollback_config": dict{
						"parallelism": 1,
						"order":       "start-first",
					},
				},
			},
		},
	}

	assert.NilError(t, Validate(config, "3.7"))
}

func TestValidateRollbackConfigWithUpdateConfig(t *testing.T) {
	config := dict{
		"version": "3.4",
		"services": dict{
			"foo": dict{
				"image": "busybox",
				"deploy": dict{
					"update_config": dict{
						"parallelism": 1,
						"order":       "start-first",
					},
					"rollback_config": dict{
						"parallelism": 1,
						"order":       "start-first",
					},
				},
			},
		},
	}

	assert.NilError(t, Validate(config, "3.7"))
}

func TestValidateRollbackConfigWithUpdateConfigFull(t *testing.T) {
	config := dict{
		"version": "3.4",
		"services": dict{
			"foo": dict{
				"image": "busybox",
				"deploy": dict{
					"update_config": dict{
						"parallelism":    1,
						"order":          "start-first",
						"delay":          "10s",
						"failure_action": "pause",
						"monitor":        "10s",
					},
					"rollback_config": dict{
						"parallelism":    1,
						"order":          "start-first",
						"delay":          "10s",
						"failure_action": "pause",
						"monitor":        "10s",
					},
				},
			},
		},
	}

	assert.NilError(t, Validate(config, "3.7"))
}
