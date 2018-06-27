package kubernetes

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestNewStackConverter(t *testing.T) {
	_, err := NewStackConverter("v1alpha1")
	assert.Check(t, is.ErrorContains(err, "stack version v1alpha1 unsupported"))

	_, err = NewStackConverter("v1beta1")
	assert.NilError(t, err)
	_, err = NewStackConverter("v1beta2")
	assert.NilError(t, err)
}
