package template

import (
	"testing"

	"github.com/docker/cli/internal/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaults = map[string]string{
	"FOO": "first",
	"BAR": "",
}

func defaultMapping(name string) (string, bool) {
	val, ok := defaults[name]
	return val, ok
}

func TestEscaped(t *testing.T) {
	result, err := Substitute("$${foo}", defaultMapping)
	assert.NoError(t, err)
	assert.Equal(t, "${foo}", result)
}

func TestSubstituteNoMatch(t *testing.T) {
	result, err := Substitute("foo", defaultMapping)
	require.NoError(t, err)
	require.Equal(t, "foo", result)
}

func TestInvalid(t *testing.T) {
	invalidTemplates := []string{
		"${",
		"$}",
		"${}",
		"${ }",
		"${ foo}",
		"${foo }",
		"${foo!}",
	}

	for _, template := range invalidTemplates {
		_, err := Substitute(template, defaultMapping)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid template")
	}
}

func TestNoValueNoDefault(t *testing.T) {
	for _, template := range []string{"This ${missing} var", "This ${BAR} var"} {
		result, err := Substitute(template, defaultMapping)
		assert.NoError(t, err)
		assert.Equal(t, "This  var", result)
	}
}

func TestValueNoDefault(t *testing.T) {
	for _, template := range []string{"This $FOO var", "This ${FOO} var"} {
		result, err := Substitute(template, defaultMapping)
		assert.NoError(t, err)
		assert.Equal(t, "This first var", result)
	}
}

func TestNoValueWithDefault(t *testing.T) {
	for _, template := range []string{"ok ${missing:-def}", "ok ${missing-def}"} {
		result, err := Substitute(template, defaultMapping)
		assert.NoError(t, err)
		assert.Equal(t, "ok def", result)
	}
}

func TestEmptyValueWithSoftDefault(t *testing.T) {
	result, err := Substitute("ok ${BAR:-def}", defaultMapping)
	assert.NoError(t, err)
	assert.Equal(t, "ok def", result)
}

func TestEmptyValueWithHardDefault(t *testing.T) {
	result, err := Substitute("ok ${BAR-def}", defaultMapping)
	assert.NoError(t, err)
	assert.Equal(t, "ok ", result)
}

func TestNonAlphanumericDefault(t *testing.T) {
	result, err := Substitute("ok ${BAR:-/non:-alphanumeric}", defaultMapping)
	assert.NoError(t, err)
	assert.Equal(t, "ok /non:-alphanumeric", result)
}

func TestMandatoryVariableErrors(t *testing.T) {
	testCases := []struct {
		template      string
		expectedError string
	}{
		{
			template:      "not ok ${UNSET_VAR:?Mandatory Variable Unset}",
			expectedError: "required variable UNSET_VAR is missing a value: Mandatory Variable Unset",
		},
		{
			template:      "not ok ${BAR:?Mandatory Variable Empty}",
			expectedError: "required variable BAR is missing a value: Mandatory Variable Empty",
		},
		{
			template:      "not ok ${UNSET_VAR:?}",
			expectedError: "required variable UNSET_VAR is missing a value",
		},
		{
			template:      "not ok ${UNSET_VAR?Mandatory Variable Unset}",
			expectedError: "required variable UNSET_VAR is missing a value: Mandatory Variable Unset",
		},
		{
			template:      "not ok ${UNSET_VAR?}",
			expectedError: "required variable UNSET_VAR is missing a value",
		},
	}

	for _, tc := range testCases {
		_, err := Substitute(tc.template, defaultMapping)
		assert.Error(t, err)
		assert.IsType(t, &InvalidTemplateError{}, err)
		testutil.ErrorContains(t, err, tc.expectedError)
	}
}

func TestDefaultsForMandatoryVariables(t *testing.T) {
	testCases := []struct {
		template string
		expected string
	}{
		{
			template: "ok ${FOO:?err}",
			expected: "ok first",
		},
		{
			template: "ok ${FOO?err}",
			expected: "ok first",
		},
		{
			template: "ok ${BAR?err}",
			expected: "ok ",
		},
	}

	for _, tc := range testCases {
		result, err := Substitute(tc.template, defaultMapping)
		assert.Nil(t, err)
		assert.Equal(t, tc.expected, result)
	}
}
