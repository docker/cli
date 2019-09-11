package manager

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

func TestPluginError(t *testing.T) {
	err := NewPluginError("new error")
	assert.Error(t, err, "new error")

	inner := fmt.Errorf("testing")
	err = wrapAsPluginError(inner, "wrapping")
	assert.Error(t, err, "wrapping: testing")
	assert.Equal(t, inner, errors.Cause(err))

	actual, err := yaml.Marshal(err)
	assert.NilError(t, err)
	assert.Equal(t, "'wrapping: testing'\n", string(actual))
}
