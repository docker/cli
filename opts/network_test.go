package opts

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNetworkOptLegacySyntax(t *testing.T) {
	testCases := []struct {
		value    string
		expected []NetworkAttachmentOpts
	}{
		{
			value: "docknet1",
			expected: []NetworkAttachmentOpts{
				{
					Target: "docknet1",
				},
			},
		},
	}
	for _, tc := range testCases {
		var network NetworkOpt
		assert.NilError(t, network.Set(tc.value))
		assert.Check(t, is.DeepEqual(tc.expected, network.Value()))
	}
}

func TestNetworkOptAdvancedSyntax(t *testing.T) {
	testCases := []struct {
		value    string
		expected []NetworkAttachmentOpts
	}{
		{
			value: "name=docknet1,alias=web,driver-opt=field1=value1",
			expected: []NetworkAttachmentOpts{
				{
					Target:  "docknet1",
					Aliases: []string{"web"},
					DriverOpts: map[string]string{
						"field1": "value1",
					},
				},
			},
		},
		{
			value: "name=docknet1,alias=web1,alias=web2,driver-opt=field1=value1,driver-opt=field2=value2",
			expected: []NetworkAttachmentOpts{
				{
					Target:  "docknet1",
					Aliases: []string{"web1", "web2"},
					DriverOpts: map[string]string{
						"field1": "value1",
						"field2": "value2",
					},
				},
			},
		},
		{
			value: "name=docknet1,ip=172.20.88.22,ip6=2001:db8::8822",
			expected: []NetworkAttachmentOpts{
				{
					Target:      "docknet1",
					Aliases:     []string{},
					IPv4Address: "172.20.88.22",
					IPv6Address: "2001:db8::8822",
				},
			},
		},
		{
			value: "name=docknet1",
			expected: []NetworkAttachmentOpts{
				{
					Target:  "docknet1",
					Aliases: []string{},
				},
			},
		},
		{
			value: "name=docknet1,mac-address=52:0f:f3:dc:50:10",
			expected: []NetworkAttachmentOpts{
				{
					Target:     "docknet1",
					Aliases:    []string{},
					MacAddress: "52:0f:f3:dc:50:10",
				},
			},
		},
		{
			value: "name=docknet1,link-local-ip=169.254.169.254,link-local-ip=169.254.10.10",
			expected: []NetworkAttachmentOpts{
				{
					Target:       "docknet1",
					Aliases:      []string{},
					LinkLocalIPs: []string{"169.254.169.254", "169.254.10.10"},
				},
			},
		},
		{
			value: "name=docknet1,\"driver-opt=com.docker.network.endpoint.sysctls=net.ipv6.conf.IFNAME.accept_ra=2,net.ipv6.conf.IFNAME.forwarding=1\"",
			expected: []NetworkAttachmentOpts{
				{
					Target:  "docknet1",
					Aliases: []string{},
					DriverOpts: map[string]string{
						// The CLI converts IFNAME to ifname - it probably shouldn't, but the API
						// allows ifname to cater for this.
						"com.docker.network.endpoint.sysctls": "net.ipv6.conf.ifname.accept_ra=2,net.ipv6.conf.ifname.forwarding=1",
					},
				},
			},
		},
		{
			value: "name=docknet1,gw-priority=10",
			expected: []NetworkAttachmentOpts{
				{
					Target:     "docknet1",
					Aliases:    []string{},
					GwPriority: 10,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			var network NetworkOpt
			assert.NilError(t, network.Set(tc.value))
			assert.Check(t, is.DeepEqual(tc.expected, network.Value()))
		})
	}
}

func TestNetworkOptAdvancedSyntaxInvalid(t *testing.T) {
	testCases := []struct {
		value         string
		expectedError string
	}{
		{
			value:         "invalidField=docknet1",
			expectedError: "invalid field",
		},
		{
			value:         "network=docknet1,invalid=web",
			expectedError: "invalid field",
		},
		{
			value:         "driver-opt=field1=value1,driver-opt=field2=value2",
			expectedError: "network name/id is not specified",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			var network NetworkOpt
			assert.ErrorContains(t, network.Set(tc.value), tc.expectedError)
		})
	}
}

func TestNetworkOptStringNetOptString(t *testing.T) {
	networkOpt := &NetworkOpt{}
	result := networkOpt.String()
	assert.Check(t, is.Equal("", result))
	if result != "" {
		t.Errorf("Expected an empty string, got %s", result)
	}
}

func TestNetworkOptTypeNetOptType(t *testing.T) {
	networkOpt := &NetworkOpt{}
	result := networkOpt.Type()
	assert.Check(t, is.Equal("network", result))
}
