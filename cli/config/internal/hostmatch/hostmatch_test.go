// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package hostmatch

import (
	"slices"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestIsPattern(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{key: "*.example.com", expected: true},
		{key: "*.dkr.ecr.*.amazonaws.com", expected: true},
		{key: "*-docker.pkg.dev", expected: true},
		{key: "*.example.com:5000", expected: true},
		{key: "foo.*.example.com", expected: true},

		// no wildcard
		{key: "example.com", expected: false},
		{key: "registry.example.com", expected: false},
		{key: "https://index.docker.io/v1/", expected: false},

		// wildcard in the last two labels
		{key: "*", expected: false},
		{key: "*.com", expected: false},
		{key: "*com", expected: false},
		{key: "foo.*.com", expected: false},
		{key: "foo.example.*", expected: false},
		{key: "*.example.com:*", expected: false},
		{key: "*.example.c*m", expected: false},

		// malformed
		{key: "*..example.com", expected: false},
		{key: ".*.example.com", expected: false},
		{key: "*.example.com.", expected: false},
		{key: "https://*.example.com", expected: false},
		{key: "*.example.com/foo", expected: false},
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			assert.Check(t, is.Equal(IsPattern(tc.key), tc.expected))
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
