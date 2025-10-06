package swarmopts

import (
	"bytes"
	"os"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestPortOptValidSimpleSyntax(t *testing.T) {
	testCases := []struct {
		value    string
		expected []swarm.PortConfig
	}{
		{
			value: "80",
			expected: []swarm.PortConfig{
				{
					Protocol:    "tcp",
					TargetPort:  80,
					PublishMode: swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "80:8080",
			expected: []swarm.PortConfig{
				{
					Protocol:      "tcp",
					TargetPort:    8080,
					PublishedPort: 80,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "8080:80/tcp",
			expected: []swarm.PortConfig{
				{
					Protocol:      "tcp",
					TargetPort:    80,
					PublishedPort: 8080,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "80:8080/udp",
			expected: []swarm.PortConfig{
				{
					Protocol:      "udp",
					TargetPort:    8080,
					PublishedPort: 80,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "80-81:8080-8081/tcp",
			expected: []swarm.PortConfig{
				{
					Protocol:      "tcp",
					TargetPort:    8080,
					PublishedPort: 80,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
				{
					Protocol:      "tcp",
					TargetPort:    8081,
					PublishedPort: 81,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "80-82:8080-8082/udp",
			expected: []swarm.PortConfig{
				{
					Protocol:      "udp",
					TargetPort:    8080,
					PublishedPort: 80,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
				{
					Protocol:      "udp",
					TargetPort:    8081,
					PublishedPort: 81,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
				{
					Protocol:      "udp",
					TargetPort:    8082,
					PublishedPort: 82,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "80-82:8080/udp",
			expected: []swarm.PortConfig{
				{
					Protocol:      "udp",
					TargetPort:    8080,
					PublishedPort: 80,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
				{
					Protocol:      "udp",
					TargetPort:    8080,
					PublishedPort: 81,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
				{
					Protocol:      "udp",
					TargetPort:    8080,
					PublishedPort: 82,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "80-80:8080/tcp",
			expected: []swarm.PortConfig{
				{
					Protocol:      "tcp",
					TargetPort:    8080,
					PublishedPort: 80,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
			},
		},
	}
	for _, tc := range testCases {
		var port PortOpt
		assert.NilError(t, port.Set(tc.value))
		assert.Check(t, is.Len(port.Value(), len(tc.expected)))
		for _, expectedPortConfig := range tc.expected {
			assertContains(t, port.Value(), expectedPortConfig)
		}
	}
}

func TestPortOptValidComplexSyntax(t *testing.T) {
	testCases := []struct {
		value    string
		expected []swarm.PortConfig
	}{
		{
			value: "target=80",
			expected: []swarm.PortConfig{
				{
					TargetPort:  80,
					Protocol:    "tcp",
					PublishMode: swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "target=80,protocol=tcp",
			expected: []swarm.PortConfig{
				{
					Protocol:    "tcp",
					TargetPort:  80,
					PublishMode: swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "target=80,published=8080,protocol=tcp",
			expected: []swarm.PortConfig{
				{
					Protocol:      "tcp",
					TargetPort:    80,
					PublishedPort: 8080,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "published=80,target=8080,protocol=tcp",
			expected: []swarm.PortConfig{
				{
					Protocol:      "tcp",
					TargetPort:    8080,
					PublishedPort: 80,
					PublishMode:   swarm.PortConfigPublishModeIngress,
				},
			},
		},
		{
			value: "target=80,published=8080,protocol=tcp,mode=host",
			expected: []swarm.PortConfig{
				{
					Protocol:      "tcp",
					TargetPort:    80,
					PublishedPort: 8080,
					PublishMode:   "host",
				},
			},
		},
		{
			value: "target=80,published=8080,mode=host",
			expected: []swarm.PortConfig{
				{
					TargetPort:    80,
					PublishedPort: 8080,
					PublishMode:   "host",
					Protocol:      "tcp",
				},
			},
		},
		{
			value: "target=80,published=8080,mode=ingress",
			expected: []swarm.PortConfig{
				{
					TargetPort:    80,
					PublishedPort: 8080,
					PublishMode:   "ingress",
					Protocol:      "tcp",
				},
			},
		},
	}
	for _, tc := range testCases {
		var port PortOpt
		assert.NilError(t, port.Set(tc.value))
		assert.Check(t, is.Len(port.Value(), len(tc.expected)))
		for _, expectedPortConfig := range tc.expected {
			assertContains(t, port.Value(), expectedPortConfig)
		}
	}
}

func TestPortOptInvalidComplexSyntax(t *testing.T) {
	testCases := []struct {
		value         string
		expectedError string
	}{
		{
			value:         "invalid,target=80",
			expectedError: "invalid field: invalid",
		},
		{
			value:         "invalid=field",
			expectedError: "invalid field key: invalid",
		},
		{
			value:         "protocol=invalid",
			expectedError: "invalid protocol value 'invalid'",
		},
		{
			value:         "target=invalid",
			expectedError: "invalid target port (invalid): value must be an integer: invalid syntax",
		},
		{
			value:         "published=invalid",
			expectedError: "invalid published port (invalid): value must be an integer: invalid syntax",
		},
		{
			value:         "mode=invalid",
			expectedError: "invalid publish mode value (invalid): must be either 'ingress' or 'host'",
		},
		{
			value:         "published=8080,protocol=tcp,mode=ingress",
			expectedError: "missing mandatory field 'target'",
		},
		{
			value:         `target=80,protocol="tcp,mode=ingress"`,
			expectedError: `parse error on line 1, column 20: bare " in non-quoted-field`,
		},
		{
			value:         `target=80,"protocol=tcp,mode=ingress"`,
			expectedError: "invalid protocol value 'tcp,mode=ingress'",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			var port PortOpt
			assert.Error(t, port.Set(tc.value), tc.expectedError)
		})
	}
}

func TestPortOptInvalidSimpleSyntax(t *testing.T) {
	testCases := []struct {
		value         string
		expectedError string
	}{
		{
			value:         "9999999",
			expectedError: "invalid containerPort: 9999999",
		},
		{
			value:         "80/xyz",
			expectedError: "invalid proto: xyz",
		},
		{
			value:         "tcp",
			expectedError: "invalid containerPort: tcp",
		},
		{
			value:         "udp",
			expectedError: "invalid containerPort: udp",
		},
		{
			value:         "",
			expectedError: "invalid proto: ",
			// expectedError: "no port specified: <empty>", // FIXME(thaJeztah): re-enable once https://github.com/docker/go-connections/pull/143 is in a go-connections release.
		},
		{
			value:         "1.1.1.1:80:80",
			expectedError: "hostip is not supported",
		},
	}
	for _, tc := range testCases {
		var port PortOpt
		assert.Error(t, port.Set(tc.value), tc.expectedError)
	}
}

func TestConvertPortToPortConfigWithIP(t *testing.T) {
	testCases := []struct {
		value           string
		expectedWarning string
	}{
		{
			value: "0.0.0.0",
		},
		{
			value: "::",
		},
		{
			value:           "192.168.1.5",
			expectedWarning: `ignoring IP-address (192.168.1.5:2345:80/tcp) service will listen on '0.0.0.0'`,
		},
		{
			value:           "::2",
			expectedWarning: `ignoring IP-address ([::2]:2345:80/tcp) service will listen on '0.0.0.0'`,
		},
	}

	var b bytes.Buffer
	logrus.SetOutput(&b)
	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			_, err := ConvertPortToPortConfig(network.MustParsePort("80/tcp"), map[nat.Port][]nat.PortBinding{
				"80/tcp": {{HostIP: tc.value, HostPort: "2345"}},
			})
			assert.NilError(t, err)
			if tc.expectedWarning == "" {
				assert.Equal(t, b.String(), "")
			} else {
				assert.Assert(t, is.Contains(b.String(), tc.expectedWarning))
			}
		})
	}
	logrus.SetOutput(os.Stderr)
}

func assertContains(t *testing.T, portConfigs []swarm.PortConfig, expected swarm.PortConfig) {
	t.Helper()
	contains := false
	for _, portConfig := range portConfigs {
		if portConfig == expected {
			contains = true
			break
		}
	}
	if !contains {
		t.Errorf("expected %v to contain %v, did not", portConfigs, expected)
	}
}
