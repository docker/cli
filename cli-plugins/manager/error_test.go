package manager

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"gotest.tools/v3/assert"
)

func TestPluginError(t *testing.T) {
	err := NewPluginError("new error")
	assert.Error(t, err, "new error")

	inner := fmt.Errorf("testing")
	err = wrapAsPluginError(inner, "wrapping")
	assert.Error(t, err, "wrapping: testing")
	assert.Assert(t, errors.Is(err, inner))

	actual, err := yaml.Marshal(err)
	assert.NilError(t, err)
	assert.Equal(t, "'wrapping: testing'\n", string(actual))
}
