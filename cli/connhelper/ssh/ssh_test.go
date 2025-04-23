package ssh

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestParseURL(t *testing.T) {
	testCases := []struct {
		doc           string
		url           string
		remoteCommand []string
		expectedArgs  []string
		expectedError string
		expectedSpec  Spec
	}{
		{
			doc: "bare ssh URL",
			url: "ssh://example.com",
			expectedArgs: []string{
				"--", "example.com",
			},
			expectedSpec: Spec{
				Host: "example.com",
			},
		},
		{
			doc: "bare ssh URL and remote command",
			url: "ssh://example.com",
			remoteCommand: []string{
				"docker", "system", "dial-stdio",
			},
			expectedArgs: []string{
				"--", "example.com",
				"docker", "system", "dial-stdio",
			},
			expectedSpec: Spec{
				Host: "example.com",
			},
		},
		{
			doc: "ssh URL with path and remote command and flag",
			url: "ssh://example.com/var/run/docker.sock",
			remoteCommand: []string{
				"docker", "--host", "unix:///var/run/docker.sock", "system", "dial-stdio",
			},
			expectedArgs: []string{
				"--", "example.com",
				"docker", "--host", "unix:///var/run/docker.sock", "system", "dial-stdio",
			},
			expectedSpec: Spec{
				Host: "example.com",
				Path: "/var/run/docker.sock",
			},
		},
		{
			doc: "ssh URL with username and port",
			url: "ssh://me@example.com:10022",
			expectedArgs: []string{
				"-l", "me",
				"-p", "10022",
				"--", "example.com",
			},
			expectedSpec: Spec{
				User: "me",
				Host: "example.com",
				Port: "10022",
			},
		},
		{
			doc: "ssh URL with username, port, and path",
			url: "ssh://me@example.com:10022/var/run/docker.sock",
			expectedArgs: []string{
				"-l", "me",
				"-p", "10022",
				"--", "example.com",
			},
			expectedSpec: Spec{
				User: "me",
				Host: "example.com",
				Port: "10022",
				Path: "/var/run/docker.sock",
			},
		},
		{
			doc:           "malformed URL",
			url:           "malformed %%url",
			expectedError: `invalid SSH URL: invalid URL escape "%%u"`,
		},
		{
			doc:           "URL missing scheme",
			url:           "no-scheme.example.com",
			expectedError: "invalid SSH URL: no scheme provided",
		},
		{
			doc:           "invalid URL with password",
			url:           "ssh://me:passw0rd@example.com",
			expectedError: "invalid SSH URL: plain-text password is not supported",
		},
		{
			doc:           "invalid URL with query parameter",
			url:           "ssh://example.com?foo=bar&bar=baz",
			expectedError: `invalid SSH URL: query parameters are not allowed: "foo=bar&bar=baz"`,
		},
		{
			doc:           "invalid URL with fragment",
			url:           "ssh://example.com#bar",
			expectedError: `invalid SSH URL: fragments are not allowed: "bar"`,
		},
		{
			doc:           "invalid URL without hostname",
			url:           "ssh://",
			expectedError: "invalid SSH URL: hostname is empty",
		},
		{
			url:           "ssh:///no-hostname",
			expectedError: "invalid SSH URL: hostname is empty",
		},
		{
			doc:           "invalid URL with unsupported scheme",
			url:           "https://example.com",
			expectedError: `invalid SSH URL: incorrect scheme: https`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			sp, err := ParseURL(tc.url)
			if tc.expectedError == "" {
				assert.NilError(t, err)
				actualArgs := sp.Args(tc.remoteCommand...)
				assert.Check(t, is.DeepEqual(actualArgs, tc.expectedArgs))
				assert.Check(t, is.Equal(*sp, tc.expectedSpec))
			} else {
				assert.Check(t, is.Error(err, tc.expectedError))
				assert.Check(t, is.Nil(sp))
			}
		})
	}
}
