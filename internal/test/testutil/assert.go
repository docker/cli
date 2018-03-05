package testutil

import (
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

// ErrorContains checks that the error is not nil, and contains the expected
// substring.
// TODO: replace with testify if https://github.com/stretchr/testify/pull/486
// is accepted.
func ErrorContains(t assert.TestingT, err error, expectedError string) {
	assert.Assert(t, is.ErrorContains(err, ""))
	assert.Check(t, is.Contains(err.Error(), expectedError))
}
