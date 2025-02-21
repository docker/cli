package manager // import "docker.com/cli/v28/cli-plugins/manager"

import (
	"encoding/json"
	"errors"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestPluginError(t *testing.T) {
	err := NewPluginError("new error")
	assert.Check(t, is.Error(err, "new error"))

	inner := errors.New("testing")
	err = wrapAsPluginError(inner, "wrapping")
	assert.Check(t, is.Error(err, "wrapping: testing"))
	assert.Check(t, is.ErrorIs(err, inner))

	actual, err := json.Marshal(err)
	assert.Check(t, err)
	assert.Check(t, is.Equal(`"wrapping: testing"`, string(actual)))
}
