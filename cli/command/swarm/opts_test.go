package swarm

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestNodeAddrOptionSetHostAndPort(t *testing.T) {
	opt := NewNodeAddrOption("old:123")
	addr := "newhost:5555"
	assert.NilError(t, opt.Set(addr))
	assert.Check(t, is.Equal(addr, opt.Value()))
}

func TestNodeAddrOptionSetHostOnly(t *testing.T) {
	opt := NewListenAddrOption()
	assert.NilError(t, opt.Set("newhost"))
	assert.Check(t, is.Equal("newhost:2377", opt.Value()))
}

func TestNodeAddrOptionSetHostOnlyIPv6(t *testing.T) {
	opt := NewListenAddrOption()
	assert.NilError(t, opt.Set("::1"))
	assert.Check(t, is.Equal("[::1]:2377", opt.Value()))
}

func TestNodeAddrOptionSetPortOnly(t *testing.T) {
	opt := NewListenAddrOption()
	assert.NilError(t, opt.Set(":4545"))
	assert.Check(t, is.Equal("0.0.0.0:4545", opt.Value()))
}

func TestNodeAddrOptionSetInvalidFormat(t *testing.T) {
	opt := NewListenAddrOption()
	assert.Error(t, opt.Set("http://localhost:4545"), "Invalid proto, expected tcp: http://localhost:4545")
}

func TestExternalCAOptionErrors(t *testing.T) {
	testCases := []struct {
		externalCA    string
		expectedError string
	}{
		{
			externalCA:    "",
			expectedError: "EOF",
		},
		{
			externalCA:    "anything",
			expectedError: "invalid field 'anything' must be a key=value pair",
		},
		{
			externalCA:    "foo=bar",
			expectedError: "the external-ca option needs a protocol= parameter",
		},
		{
			externalCA:    "protocol=baz",
			expectedError: "unrecognized external CA protocol baz",
		},
		{
			externalCA:    "protocol=cfssl",
			expectedError: "the external-ca option needs a url= parameter",
		},
	}
	for _, tc := range testCases {
		opt := &ExternalCAOption{}
		assert.Error(t, opt.Set(tc.externalCA), tc.expectedError)
	}
}

func TestExternalCAOption(t *testing.T) {
	testCases := []struct {
		externalCA string
		expected   string
	}{
		{
			externalCA: "protocol=cfssl,url=anything",
			expected:   "cfssl: anything",
		},
		{
			externalCA: "protocol=CFSSL,url=anything",
			expected:   "cfssl: anything",
		},
		{
			externalCA: "protocol=Cfssl,url=https://example.com",
			expected:   "cfssl: https://example.com",
		},
		{
			externalCA: "protocol=Cfssl,url=https://example.com,foo=bar",
			expected:   "cfssl: https://example.com",
		},
		{
			externalCA: "protocol=Cfssl,url=https://example.com,foo=bar,foo=baz",
			expected:   "cfssl: https://example.com",
		},
	}
	for _, tc := range testCases {
		opt := &ExternalCAOption{}
		assert.NilError(t, opt.Set(tc.externalCA))
		assert.Check(t, is.Equal(tc.expected, opt.String()))
	}
}

func TestExternalCAOptionMultiple(t *testing.T) {
	opt := &ExternalCAOption{}
	assert.NilError(t, opt.Set("protocol=cfssl,url=https://example.com"))
	assert.NilError(t, opt.Set("protocol=CFSSL,url=anything"))
	assert.Check(t, is.Len(opt.Value(), 2))
	assert.Check(t, is.Equal("cfssl: https://example.com, cfssl: anything", opt.String()))
}
