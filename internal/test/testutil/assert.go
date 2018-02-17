package testutil

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ErrorContains checks that the error is not nil, and contains the expected
// substring.
// TODO: replace with testify if https://github.com/stretchr/testify/pull/486
// is accepted.
func ErrorContains(t require.TestingT, err error, expectedError string) {
	require.Error(t, err)
	assert.Contains(t, err.Error(), expectedError)
}
