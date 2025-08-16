package service

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestValidateSingleGenericResource(t *testing.T) {
	incorrect := []string{"foo", "fooo-bar"}
	correct := []string{"foo=bar", "bar=1", "foo=barbar"}

	for _, v := range incorrect {
		_, err := ValidateSingleGenericResource(v)
		assert.Check(t, is.ErrorContains(err, ""))
	}

	for _, v := range correct {
		_, err := ValidateSingleGenericResource(v)
		assert.NilError(t, err)
	}
}
