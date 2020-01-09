package swarm

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
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
		externalCAs    []string
		expected       []*swarm.ExternalCA
		expectedString string
	}{
		{
			externalCAs:    []string{""},
			expected:       nil,
			expectedString: "",
		},
		{
			externalCAs: []string{"protocol=cfssl,url=anything"},
			expected: []*swarm.ExternalCA{
				{
					Protocol: swarm.ExternalCAProtocolCFSSL,
					URL:      "anything",
					Options:  make(map[string]string),
				},
			},
			expectedString: "cfssl: anything",
		},
		{
			externalCAs: []string{"protocol=CFSSL,url=anything"},
			expected: []*swarm.ExternalCA{
				{
					Protocol: swarm.ExternalCAProtocolCFSSL,
					URL:      "anything",
					Options:  make(map[string]string),
				},
			},
			expectedString: "cfssl: anything",
		},
		{
			externalCAs: []string{"protocol=Cfssl,url=https://example.com"},
			expected: []*swarm.ExternalCA{
				{
					Protocol: swarm.ExternalCAProtocolCFSSL,
					URL:      "https://example.com",
					Options:  make(map[string]string),
				},
			},
			expectedString: "cfssl: https://example.com",
		},
		{
			externalCAs: []string{"protocol=Cfssl,url=https://example.com,foo=bar"},
			expected: []*swarm.ExternalCA{
				{
					Protocol: swarm.ExternalCAProtocolCFSSL,
					URL:      "https://example.com",
					Options: map[string]string{
						"foo": "bar",
					},
				},
			},
			expectedString: "cfssl: https://example.com",
		},
		{
			externalCAs: []string{"protocol=Cfssl,url=https://example.com,foo=bar,foo=baz"},
			expected: []*swarm.ExternalCA{
				{
					Protocol: swarm.ExternalCAProtocolCFSSL,
					URL:      "https://example.com",
					Options: map[string]string{
						"foo": "baz",
					},
				},
			},
			expectedString: "cfssl: https://example.com",
		},
		{
			externalCAs: []string{"", "protocol=Cfssl,url=https://example.com"},
			expected: []*swarm.ExternalCA{
				{
					Protocol: swarm.ExternalCAProtocolCFSSL,
					URL:      "https://example.com",
					Options:  make(map[string]string),
				},
			},
			expectedString: "cfssl: https://example.com",
		},
		{
			externalCAs: []string{
				"protocol=Cfssl,url=https://example.com",
				"protocol=Cfssl,url=https://example2.com",
			},
			expected: []*swarm.ExternalCA{
				{
					Protocol: swarm.ExternalCAProtocolCFSSL,
					URL:      "https://example.com",
					Options:  make(map[string]string),
				},
				{
					Protocol: swarm.ExternalCAProtocolCFSSL,
					URL:      "https://example2.com",
					Options:  make(map[string]string),
				},
			},
			expectedString: "cfssl: https://example.com, cfssl: https://example2.com",
		},
	}
	for _, tc := range testCases {
		opt := &ExternalCAOption{}
		for _, extCA := range tc.externalCAs {
			assert.NilError(t, opt.Set(extCA))
		}
		assert.Check(t, is.DeepEqual(tc.expected, opt.Value()))
		assert.Check(t, is.Equal(tc.expectedString, opt.String()))
	}
}

func TestExternalCAOptionMultiple(t *testing.T) {
	opt := &ExternalCAOption{}
	assert.NilError(t, opt.Set("protocol=cfssl,url=https://example.com"))
	assert.NilError(t, opt.Set("protocol=CFSSL,url=anything"))
	assert.Check(t, is.Len(opt.Value(), 2))
	assert.Check(t, is.Equal("cfssl: https://example.com, cfssl: anything", opt.String()))
}
