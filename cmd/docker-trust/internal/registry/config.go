// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package registry

import (
	"net"
	"strings"

	"github.com/distribution/reference"
	"github.com/moby/moby/api/types/registry"
)

// IndexName is the name of the index
const IndexName = "docker.io"

func normalizeIndexName(val string) string {
	if val == "index.docker.io" {
		return "docker.io"
	}
	return val
}

// NewIndexInfo creates a new [registry.IndexInfo] or the given
// repository-name, and detects whether the registry is considered
// "secure" (non-localhost).
func NewIndexInfo(reposName reference.Named) *registry.IndexInfo {
	indexName := normalizeIndexName(reference.Domain(reposName))
	if indexName == IndexName {
		return &registry.IndexInfo{
			Name:     IndexName,
			Secure:   true,
			Official: true,
		}
	}

	return &registry.IndexInfo{
		Name:   indexName,
		Secure: !isInsecure(indexName),
	}
}

// isInsecure is used to detect whether a registry domain or IP-address is allowed
// to use an insecure (non-TLS, or self-signed cert) connection according to the
// defaults, which allows for insecure connections with registries running on a
// loopback address ("localhost", "::1/128", "127.0.0.0/8").
//
// It is used in situations where we don't have access to the daemon's configuration,
// for example, when used from the client / CLI.
func isInsecure(hostNameOrIP string) bool {
	// Attempt to strip port if present; this also strips brackets for
	// IPv6 addresses with a port (e.g. "[::1]:5000").
	//
	// This is best-effort; we'll continue using the address as-is if it fails.
	if host, _, err := net.SplitHostPort(hostNameOrIP); err == nil {
		hostNameOrIP = host
	}
	if hostNameOrIP == "127.0.0.1" || hostNameOrIP == "::1" || strings.EqualFold(hostNameOrIP, "localhost") {
		// Fast path; no need to resolve these, assuming nobody overrides
		// "localhost" for anything else than a loopback address (sorry, not sorry).
		return true
	}

	var addresses []net.IP
	if ip := net.ParseIP(hostNameOrIP); ip != nil {
		addresses = append(addresses, ip)
	} else {
		// Try to resolve the host's IP-addresses.
		addrs, _ := net.LookupIP(hostNameOrIP)
		addresses = append(addresses, addrs...)
	}

	for _, addr := range addresses {
		if addr.IsLoopback() {
			return true
		}
	}
	return false
}
