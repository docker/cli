package ssh

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestParseSSHURL(t *testing.T) {
	testCases := []struct {
		url           string
		expectedArgs  []string
		expectedError string
	}{
		{
			url: "ssh://foo",
			expectedArgs: []string{
				"foo",
			},
		},
		{
			url: "ssh://me@foo:10022",
			expectedArgs: []string{
				"-l", "me",
				"-p", "10022",
				"foo",
			},
		},
		{
			url:           "ssh://me:passw0rd@foo",
			expectedError: "plain-text password is not supported",
		},
		{
			url:           "ssh://foo/bar",
			expectedError: `extra path after the host: "/bar"`,
		},
		{
			url:           "ssh://foo?bar",
			expectedError: `extra query after the host: "bar"`,
		},
		{
			url:           "ssh://foo#bar",
			expectedError: `extra fragment after the host: "bar"`,
		},
		{
			url:           "ssh://",
			expectedError: "no host specified",
		},
		{
			url:           "foo://bar",
			expectedError: `expected scheme ssh, got "foo"`,
		},
	}
	for _, tc := range testCases {
		sp, err := parseSSHURL(tc.url)
		if tc.expectedError == "" {
			assert.NilError(t, err)
			assert.Check(t, is.DeepEqual(tc.expectedArgs, sp.Args()))
		} else {
			assert.ErrorContains(t, err, tc.expectedError)
		}
	}
}
