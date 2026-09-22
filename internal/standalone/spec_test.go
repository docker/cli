//go:build linux

package standalone

import (
	"testing"

	"github.com/moby/moby/api/types/container"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestParseSignal(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{input: "", expected: 15},
		{input: "9", expected: 9},
		{input: "KILL", expected: 9},
		{input: "SIGKILL", expected: 9},
		{input: "sigterm", expected: 15},
		{input: "HUP", expected: 1},
		{input: "NOPE", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			sig, err := parseSignal(tc.input)
			if tc.wantErr {
				assert.Check(t, err != nil)
				return
			}
			assert.NilError(t, err)
			assert.Check(t, is.Equal(sig, tc.expected))
		})
	}
}

func TestStopSignalAndTimeout(t *testing.T) {
	timeout := 42
	m := &containerMeta{Config: &container.Config{}}
	assert.Check(t, is.Equal(stopSignal(m), "SIGTERM"))
	assert.Check(t, is.Equal(stopTimeout(m, nil), 10))
	assert.Check(t, is.Equal(stopTimeout(m, &timeout), 42))

	m.Config.StopSignal = "SIGUSR1"
	m.Config.StopTimeout = &timeout
	assert.Check(t, is.Equal(stopSignal(m), "SIGUSR1"))
	assert.Check(t, is.Equal(stopTimeout(m, nil), 42))
}

func TestNormalizeCaps(t *testing.T) {
	assert.Check(t, is.DeepEqual(normalizeCaps([]string{"net_admin", "CAP_SYS_TIME", "SYS_ADMIN"}),
		[]string{"CAP_NET_ADMIN", "CAP_SYS_TIME", "CAP_SYS_ADMIN"}))
	assert.Check(t, containsAll([]string{"foo", "all"}))
	assert.Check(t, !containsAll([]string{"foo", "bar"}))
}

func TestNetworkModeName(t *testing.T) {
	assert.Check(t, is.Equal(networkModeName(nil), "bridge"))
	assert.Check(t, is.Equal(networkModeName(&container.HostConfig{}), "bridge"))
	assert.Check(t, is.Equal(networkModeName(&container.HostConfig{NetworkMode: "default"}), "bridge"))
	assert.Check(t, is.Equal(networkModeName(&container.HostConfig{NetworkMode: "host"}), "host"))
	assert.Check(t, is.Equal(networkModeName(&container.HostConfig{NetworkMode: "none"}), "none"))
}

func TestShortID(t *testing.T) {
	assert.Check(t, is.Equal(shortID("0123456789abcdef0123"), "0123456789ab"))
	assert.Check(t, is.Equal(shortID("short"), "short"))
	assert.Check(t, is.Equal(shortID(""), ""))
}

func TestValidateName(t *testing.T) {
	assert.NilError(t, validateName("my-container"))
	assert.NilError(t, validateName("a1"))
	assert.NilError(t, validateName("Name_with.dots-and_underscores"))
	assert.Check(t, validateName("a") != nil)         // too short, as in the daemon
	assert.Check(t, validateName("-leading") != nil)  // must start alphanumeric
	assert.Check(t, validateName("has space") != nil) // invalid character
}

func TestResolvConfContent(t *testing.T) {
	e := &engine{cfg: &config{}}
	m := &containerMeta{
		Config: &container.Config{},
		HostConfig: &container.HostConfig{
			DNSSearch:  []string{"example.com"},
			DNSOptions: []string{"ndots:2"},
		},
		Network: networkState{Backend: networkPasta, DNS: []string{pastaDNSForward}},
	}
	out := e.resolvConfContent(m)
	assert.Check(t, is.Contains(out, "nameserver "+pastaDNSForward))
	assert.Check(t, is.Contains(out, "search example.com"))
	assert.Check(t, is.Contains(out, "options ndots:2"))
	assert.Check(t, !is.Contains(out, "nameserver 127.0.0.")().Success(), "loopback nameservers must be filtered")
}

func TestHostsContent(t *testing.T) {
	e := &engine{cfg: &config{}}
	m := &containerMeta{
		Name:   "web",
		Config: &container.Config{Hostname: "abc123"},
		HostConfig: &container.HostConfig{
			ExtraHosts: []string{"extra.example:10.0.0.9", "gw.example:host-gateway"},
		},
		Network: networkState{Backend: networkBridge, IPAddress: "10.88.0.5", Gateway: "10.88.0.1"},
	}
	out := e.hostsContent(m)
	assert.Check(t, is.Contains(out, "127.0.0.1\tlocalhost"))
	assert.Check(t, is.Contains(out, "10.0.0.9\textra.example"))
	assert.Check(t, is.Contains(out, "10.88.0.1\tgw.example"))
	assert.Check(t, is.Contains(out, "10.88.0.5\tabc123 web"))
	assert.Check(t, is.Contains(out, "host.docker.internal"))
}

func TestHostGatewayIP(t *testing.T) {
	assert.Check(t, is.Equal(hostGatewayIP(&containerMeta{Network: networkState{Backend: networkSlirp}}), "10.0.2.2"))
	assert.Check(t, is.Equal(hostGatewayIP(&containerMeta{Network: networkState{Backend: networkHost}}), "127.0.0.1"))
	assert.Check(t, is.Equal(hostGatewayIP(&containerMeta{Network: networkState{Backend: networkBridge, Gateway: "10.88.0.1"}}), "10.88.0.1"))
	assert.Check(t, is.Equal(hostGatewayIP(&containerMeta{Network: networkState{Backend: networkNone}}), ""))
}
