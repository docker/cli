package ssh

import (
	"strings"
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
			doc: "bare ssh URL with trailing slash",
			url: "ssh://example.com/",
			expectedArgs: []string{
				"--", "example.com",
			},
			expectedSpec: Spec{
				Host: "example.com",
				Path: "/",
			},
		},
		{
			doc: "bare ssh URL with trailing slashes",
			url: "ssh://example.com//",
			expectedArgs: []string{
				"--", "example.com",
			},
			expectedSpec: Spec{
				Host: "example.com",
				Path: "//",
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
				`docker system dial-stdio`,
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
				`docker --host unix:///var/run/docker.sock system dial-stdio`,
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
			// This test is only to verify the behavior of ParseURL to
			// pass through the Path as-is. Neither Spec.Args, nor
			// Spec.Command use the Path field directly, and it should
			// likely be deprecated.
			doc: "bad path",
			url: `ssh://example.com/var/run/docker.sock '$(echo hello > /hello.txt)'`,
			remoteCommand: []string{
				"docker", "--host", `unix:///var/run/docker.sock '$(echo hello > /hello.txt)'`, "system", "dial-stdio",
			},
			expectedArgs: []string{
				"--", "example.com",
				`docker --host "unix:///var/run/docker.sock '\$(echo hello > /hello.txt)'" system dial-stdio`,
			},
			expectedSpec: Spec{
				Host: "example.com",
				Path: `/var/run/docker.sock '$(echo hello > /hello.txt)'`,
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
		{
			doc:           "invalid URL with NUL character",
			url:           "ssh://example.com/var/run/\x00docker.sock",
			expectedError: `invalid SSH URL: net/url: invalid control character in URL`,
		},
		{
			doc:           "invalid URL with newline character",
			url:           "ssh://example.com/var/run/docker.sock\n",
			expectedError: `invalid SSH URL: net/url: invalid control character in URL`,
		},
		{
			doc:           "invalid URL with control character",
			url:           "ssh://example.com/var/run/\x1bdocker.sock",
			expectedError: `invalid SSH URL: net/url: invalid control character in URL`,
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

func TestCommand(t *testing.T) {
	testCases := []struct {
		doc           string
		url           string
		sshFlags      []string
		customCmd     []string
		expectedCmd   []string
		expectedError string
	}{
		{
			doc: "bare ssh URL",
			url: "ssh://example.com",
			expectedCmd: []string{
				"--", "example.com",
				"docker system dial-stdio",
			},
		},
		{
			doc: "bare ssh URL with trailing slash",
			url: "ssh://example.com/",
			expectedCmd: []string{
				"--", "example.com",
				"docker system dial-stdio",
			},
		},
		{
			doc:      "bare ssh URL with custom ssh flags",
			url:      "ssh://example.com",
			sshFlags: []string{"-T", "-o", "ConnectTimeout=30", "-oStrictHostKeyChecking=no"},
			expectedCmd: []string{
				"-T",
				"-o", "ConnectTimeout=30",
				"-oStrictHostKeyChecking=no",
				"--", "example.com",
				"docker system dial-stdio",
			},
		},
		{
			doc:      "ssh URL with all options",
			url:      "ssh://me@example.com:10022/var/run/docker.sock",
			sshFlags: []string{"-T", "-o ConnectTimeout=30"},
			expectedCmd: []string{
				"-l", "me",
				"-p", "10022",
				"-T",
				"-o ConnectTimeout=30",
				"--", "example.com",
				"docker '--host=unix:///var/run/docker.sock' system dial-stdio",
			},
		},
		{
			doc:      "bad ssh flags",
			url:      "ssh://example.com",
			sshFlags: []string{"-T", "-o", `ConnectTimeout=30 $(echo hi > /hi.txt)`},
			expectedCmd: []string{
				"-T",
				"-o", `ConnectTimeout=30 $(echo hi > /hi.txt)`,
				"--", "example.com",
				"docker system dial-stdio",
			},
		},
		{
			doc: "bad username",
			url: `ssh://$(shutdown)me@example.com`,
			expectedCmd: []string{
				"-l", `'$(shutdown)me'`,
				"--", "example.com",
				"docker system dial-stdio",
			},
		},
		{
			doc: "bad hostname",
			url: `ssh://$(shutdown)example.com`,
			expectedCmd: []string{
				"--", `'$(shutdown)example.com'`,
				"docker system dial-stdio",
			},
		},
		{
			doc: "bad path",
			url: `ssh://example.com/var/run/docker.sock '$(echo hello > /hello.txt)'`,
			expectedCmd: []string{
				"--", "example.com",
				`docker "--host=unix:///var/run/docker.sock '\$(echo hello > /hello.txt)'" system dial-stdio`,
			},
		},
		{
			doc:           "missing command",
			url:           "ssh://example.com",
			customCmd:     []string{},
			expectedError: "no remote command specified",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			sp, err := ParseURL(tc.url)
			assert.NilError(t, err)

			var commandAndArgs []string
			if tc.customCmd == nil {
				socketPath := sp.Path
				commandAndArgs = []string{"docker", "system", "dial-stdio"}
				if strings.Trim(socketPath, "/") != "" {
					commandAndArgs = []string{"docker", "--host=unix://" + socketPath, "system", "dial-stdio"}
				}
			}

			actualCmd, err := sp.Command(tc.sshFlags, commandAndArgs...)
			if tc.expectedError == "" {
				assert.NilError(t, err)
				assert.Check(t, is.DeepEqual(actualCmd, tc.expectedCmd), "%+#v", actualCmd)
			} else {
				assert.Check(t, is.Error(err, tc.expectedError))
				assert.Check(t, is.Nil(actualCmd))
			}
		})
	}
}
