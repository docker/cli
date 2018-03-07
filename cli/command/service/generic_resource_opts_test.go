package service

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
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
