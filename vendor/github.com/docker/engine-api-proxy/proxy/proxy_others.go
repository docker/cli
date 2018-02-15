// +build !windows

package proxy

import (
	"net"

	user "github.com/dnephin/go-os-user"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
	"github.com/pkg/errors"
)

func newListener(host, groupname string) (net.Listener, error) {
	proto, addr, _, err := client.ParseHost(host)
	if err != nil {
		return nil, err
	}
	switch proto {
	case "unix":
		group, err := user.LookupGroup(groupname)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid group %q", groupname)
		}
		return sockets.NewUnixSocket(addr, group.Gid)
	case "tcp", "http":
		// TODO: support https/tls
		return net.Listen("tcp", addr)
	default:
		return nil, errors.Errorf("unsupported protocol in %s", host)
	}
}
