// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package hostmatch

import (
	"slices"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		key         string
		expectedErr string
	}{
		{key: "*.example.com"},
		{key: "*.dkr.ecr.*.amazonaws.com"},
		{key: "*-docker.pkg.dev"},
		{key: "*.example.com:5000"},
		{key: "foo.*.example.com"},

		// no wildcard; not validated
		{key: "example.com"},
		{key: "registry.example.com"},
		{key: "https://index.docker.io/v1/"},
		{key: "localhost"},

		// wildcard in the last two labels
		{key: "*", expectedErr: "wildcards are not allowed in the last two labels"},
		{key: "*.com", expectedErr: "wildcards are not allowed in the last two labels"},
		{key: "*com", expectedErr: "wildcards are not allowed in the last two labels"},
		{key: "foo.*.com", expectedErr: "wildcards are not allowed in the last two labels"},
		{key: "foo.example.*", expectedErr: "wildcards are not allowed in the last two labels"},
		{key: "*.example.com:*", expectedErr: "wildcards are not allowed in the last two labels"},
		{key: "*.example.c*m", expectedErr: "wildcards are not allowed in the last two labels"},

		// malformed
		{key: "*..example.com", expectedErr: "contains an empty label"},
		{key: ".*.example.com", expectedErr: "contains an empty label"},
		{key: "*.example.com.", expectedErr: "contains an empty label"},
		{key: "https://*.example.com", expectedErr: "without scheme or path"},
		{key: "*.example.com/foo", expectedErr: "without scheme or path"},
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			err := Validate(tc.key)
			if tc.expectedErr == "" {
				assert.NilError(t, err)
				assert.Check(t, is.Equal(IsPattern(tc.key), strings.Contains(tc.key, "*")))
			} else {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
				assert.Check(t, !IsPattern(tc.key))
			}
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern  string
		host     string
		expected bool
	}{
		{pattern: "*.example.com", host: "foo.example.com", expected: true},
		{pattern: "*.example.com", host: "example.com", expected: false},
		{pattern: "*.example.com", host: "foo.bar.example.com", expected: false},
		{pattern: "*.example.com", host: "foo.example.org", expected: false},
		{pattern: "*.example.com", host: "foo.example.com:5000", expected: false},
		{pattern: "*.example.com", host: ".example.com", expected: false},
		{pattern: "*.example.com", host: "https://foo.example.com", expected: false},
		{pattern: "*.example.com", host: "foo.example.com/v2/", expected: false},
		{pattern: "*.example.com:5000", host: "foo.example.com:5000", expected: true},
		{pattern: "*.example.com:5000", host: "foo.example.com", expected: false},
		{pattern: "abc.*.def.example.com", host: "abc.sdf.def.example.com", expected: true},
		{pattern: "abc.*.def.example.com", host: "abc.sdf.sdf.def.example.com", expected: false},
		{pattern: "*.dkr.ecr.*.amazonaws.com", host: "123456789012.dkr.ecr.us-east-1.amazonaws.com", expected: true},
		{pattern: "*.dkr.ecr.*.amazonaws.com", host: "123456789012.dkr.ecr.amazonaws.com", expected: false},
		{pattern: "*-docker.pkg.dev", host: "us-docker.pkg.dev", expected: true},
		{pattern: "*-docker.pkg.dev", host: "docker.pkg.dev", expected: false},
		{pattern: "*-docker.pkg.dev", host: "-docker.pkg.dev", expected: true},
		{pattern: "a*b*c.example.com", host: "abc.example.com", expected: true},
		{pattern: "a*b*c.example.com", host: "axxbyyc.example.com", expected: true},
		{pattern: "a*b*c.example.com", host: "axxcyyb.example.com", expected: false},
		{pattern: "ab*ba.example.com", host: "aba.example.com", expected: false},

		// invalid patterns never match
		{pattern: "*.com", host: "example.com", expected: false},
		{pattern: "foo.*.com", host: "foo.example.com", expected: false},
		{pattern: "foo.example.com", host: "foo.example.com", expected: false},
	}
	for _, tc := range tests {
		t.Run(tc.pattern+"="+tc.host, func(t *testing.T) {
			assert.Check(t, is.Equal(Match(tc.pattern, tc.host), tc.expected))
		})
	}
}

func TestBest(t *testing.T) {
	keys := []string{
		"example.com",
		"*.com",
		"*.example.com",
		"*.docker.example.com",
		"*.dock*.example.com",
		"*.*.example.com",
		"*.docker.exa*.com", // invalid: wildcard in the last two labels
		"b*.foo.example.com",
		"*a.foo.example.com",
	}
	tests := []struct {
		host     string
		expected string
	}{
		{host: "foo.docker.example.com", expected: "*.docker.example.com"},
		{host: "foo.dockyard.example.com", expected: "*.dock*.example.com"},
		{host: "foo.other.example.com", expected: "*.*.example.com"},
		{host: "foo.docker.examine.com", expected: ""},
		{host: "foo.example.com", expected: "*.example.com"},
		{host: "ba.foo.example.com", expected: "*a.foo.example.com"},
		{host: "example.com", expected: ""},
		{host: "foo.example.org", expected: ""},
	}
	for _, tc := range tests {
		t.Run(tc.host, func(t *testing.T) {
			actual, ok := Best(slices.Values(keys), tc.host)
			assert.Check(t, is.Equal(ok, tc.expected != ""))
			assert.Check(t, is.Equal(actual, tc.expected))
		})
	}
}
