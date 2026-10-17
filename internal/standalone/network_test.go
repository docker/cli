//go:build linux

package standalone

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestResolvePortMappings(t *testing.T) {
	t.Run("explicit binding", func(t *testing.T) {
		hc := &container.HostConfig{PortBindings: network.PortMap{
			network.MustParsePort("80/tcp"): []network.PortBinding{{HostPort: "8080"}},
			network.MustParsePort("53/udp"): []network.PortBinding{{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "5353"}},
		}}
		mappings, portMap, err := resolvePortMappings(&container.Config{}, hc)
		assert.NilError(t, err)
		assert.Check(t, is.Len(mappings, 2))
		assert.Check(t, is.Len(portMap, 2))

		byProto := map[string]portMapping{}
		for _, m := range mappings {
			byProto[m.proto] = m
		}
		assert.Check(t, is.Equal(byProto["tcp"].hostPort, uint16(8080)))
		assert.Check(t, is.Equal(byProto["tcp"].containerPort, uint16(80)))
		assert.Check(t, is.Equal(byProto["udp"].hostPort, uint16(5353)))
		assert.Check(t, is.Equal(byProto["udp"].hostIP.String(), "127.0.0.1"))

		// An unspecified host address is reported as 0.0.0.0, as the daemon does.
		bindings := portMap[network.MustParsePort("80/tcp")]
		assert.Check(t, is.Len(bindings, 1))
		assert.Check(t, is.Equal(bindings[0].HostIP.String(), "0.0.0.0"))
	})

	t.Run("publish all", func(t *testing.T) {
		cfg := &container.Config{ExposedPorts: network.PortSet{network.MustParsePort("80/tcp"): {}}}
		hc := &container.HostConfig{PublishAllPorts: true}
		mappings, _, err := resolvePortMappings(cfg, hc)
		assert.NilError(t, err)
		assert.Check(t, is.Len(mappings, 1))
		// A host port is allocated automatically.
		assert.Check(t, mappings[0].hostPort != 0)
	})

	t.Run("port range picks the first port", func(t *testing.T) {
		hc := &container.HostConfig{PortBindings: network.PortMap{
			network.MustParsePort("80/tcp"): []network.PortBinding{{HostPort: "9000-9010"}},
		}}
		mappings, _, err := resolvePortMappings(&container.Config{}, hc)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(mappings[0].hostPort, uint16(9000)))
	})

	t.Run("invalid host port", func(t *testing.T) {
		hc := &container.HostConfig{PortBindings: network.PortMap{
			network.MustParsePort("80/tcp"): []network.PortBinding{{HostPort: "nope"}},
		}}
		_, _, err := resolvePortMappings(&container.Config{}, hc)
		assert.Check(t, err != nil)
	})
}

func TestDedupePastaNone(t *testing.T) {
	// "none" defaults are dropped for protocols that have explicit mappings,
	// because pasta honours the first occurrence of an option.
	args := []string{"-t", "8080:80", "-t", "none", "-u", "none"}
	got := dedupePastaNone(args, []portMapping{{proto: "tcp"}})
	assert.Check(t, is.DeepEqual(got, []string{"-t", "8080:80", "-u", "none"}))

	got = dedupePastaNone(args, []portMapping{{proto: "tcp"}, {proto: "udp"}})
	assert.Check(t, is.DeepEqual(got, []string{"-t", "8080:80"}))

	got = dedupePastaNone([]string{"-t", "none", "-u", "none"}, nil)
	assert.Check(t, is.DeepEqual(got, []string{"-t", "none", "-u", "none"}))
}

func TestDefaultNetworkBackend(t *testing.T) {
	// Rootless never uses the CNI bridge.
	backend := defaultNetworkBackend(&config{rootless: true})
	assert.Check(t, backend != networkBridge)
}

func TestNetworkSummary(t *testing.T) {
	e := &engine{cfg: &config{network: networkPasta}}
	sum := e.networkSummary(networkBridge)
	assert.Check(t, is.Equal(sum.Name, networkBridge))
	assert.Check(t, is.Equal(sum.Driver, networkPasta))
	assert.Check(t, is.Equal(sum.Scope, "local"))

	host := e.networkSummary(networkHost)
	assert.Check(t, is.Equal(host.Driver, networkHost))
}

func TestRemoveNetnsMissing(t *testing.T) {
	assert.NilError(t, removeNetns(""))
	assert.NilError(t, removeNetns(filepath.Join(t.TempDir(), "nosuch")))
}

func TestParseResolvConf(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "resolv.conf")
	content := `# comment
nameserver 192.168.1.1
nameserver 2001:db8::1
search example.com corp.example.com
options ndots:2 timeout:1
`
	assert.NilError(t, os.WriteFile(path, []byte(content), 0o600))

	ns, search, opts := parseResolvConf(path)
	assert.Check(t, is.DeepEqual(ns, []string{"192.168.1.1", "2001:db8::1"}))
	assert.Check(t, is.DeepEqual(search, []string{"example.com", "corp.example.com"}))
	assert.Check(t, is.DeepEqual(opts, []string{"ndots:2", "timeout:1"}))

	ns, _, _ = parseResolvConf(filepath.Join(dir, "missing"))
	assert.Check(t, is.Len(ns, 0))
}

func TestIsLoopbackAddr(t *testing.T) {
	assert.Check(t, isLoopbackAddr("127.0.0.1"))
	assert.Check(t, isLoopbackAddr("127.0.0.53"))
	assert.Check(t, isLoopbackAddr("::1"))
	assert.Check(t, !isLoopbackAddr("192.168.1.1"))
}

func TestParseIntDefault(t *testing.T) {
	assert.Check(t, is.Equal(parseIntDefault("42", 7), 42))
	assert.Check(t, is.Equal(parseIntDefault("", 7), 7))
	assert.Check(t, is.Equal(parseIntDefault("abc", 7), 7))
}
