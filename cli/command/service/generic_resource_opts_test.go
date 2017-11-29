package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSingleGenericResource(t *testing.T) {
	incorrect := []string{"foo", "fooo-bar"}
	correct := []string{"foo=bar", "bar=1", "foo=barbar"}

	for _, v := range incorrect {
		_, err := ValidateSingleGenericResource(v)
		assert.Error(t, err)
	}

	for _, v := range correct {
		_, err := ValidateSingleGenericResource(v)
		assert.NoError(t, err)
	}
}
