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
			expectedError: "ssh helper does not accept plain-text password",
		},
		{
			url:           "ssh://foo/bar",
			expectedError: "extra path",
		},
		{
			url:           "ssh://foo?bar",
			expectedError: "extra query",
		},
		{
			url:           "ssh://foo#bar",
			expectedError: "extra fragment",
		},
		{
			url:           "ssh://",
			expectedError: "host is not specified",
		},
		{
			url:           "foo://bar",
			expectedError: "expected scheme ssh",
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
