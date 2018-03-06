package testutil

import (
	"github.com/gotestyourself/gotestyourself/assert"
)

type helperT interface {
	Helper()
}

// ErrorContains checks that the error is not nil, and contains the expected
// substring.
// Deprecated: use assert.Assert(t, cmp.ErrorContains(err, expected))
func ErrorContains(t assert.TestingT, err error, expectedError string) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	assert.ErrorContains(t, err, expectedError)
}
