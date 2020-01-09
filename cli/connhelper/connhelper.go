// Package connhelper provides helpers for connecting to a remote daemon host with custom logic.
package connhelper

import (
	"context"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/connhelper/commandconn"
	"github.com/docker/cli/cli/connhelper/ssh"
	"github.com/pkg/errors"
)

// ConnectionHelper allows to connect to a remote host with custom stream provider binary.
type ConnectionHelper struct {
	Dialer func(ctx context.Context, network, addr string) (net.Conn, error)
	Host   string // dummy URL used for HTTP requests. e.g. "http://docker"
}

// GetConnectionHelper returns Docker-specific connection helper for the given URL.
// GetConnectionHelper returns nil without error when no helper is registered for the scheme.
//
// ssh://<user>@<host> URL requires Docker 18.09 or later on the remote host.
func GetConnectionHelper(daemonURL string) (*ConnectionHelper, error) {
	u, err := url.Parse(daemonURL)
	if err != nil {
		return nil, err
	}
	switch scheme := u.Scheme; scheme {
	case "ssh":
		sp, err := ssh.ParseURL(daemonURL)
		if err != nil {
			return nil, errors.Wrap(err, "ssh host connection is not valid")
		}
		return &ConnectionHelper{
			Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return commandconn.New(ctx, "ssh", append(multiplexingArgs(), append(sp.Args(), []string{"--", "docker", "system", "dial-stdio"}...)...)...)
			},
			Host: "http://docker",
		}, nil
	}
	// Future version may support plugins via ~/.docker/config.json. e.g. "dind"
	// See docker/cli#889 for the previous discussion.
	return nil, err
}

// GetCommandConnectionHelper returns Docker-specific connection helper constructed from an arbitrary command.
func GetCommandConnectionHelper(cmd string, flags ...string) (*ConnectionHelper, error) {
	return &ConnectionHelper{
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return commandconn.New(ctx, cmd, flags...)
		},
		Host: "http://docker",
	}, nil
}

func multiplexingArgs() []string {
	if v := os.Getenv("DOCKER_SSH_NO_MUX"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil && b {
			return nil
		}
	}
	if err := os.MkdirAll(config.Dir(), 0700); err != nil {
		return nil
	}
	args := []string{"-o", "ControlMaster=auto", "-o", "ControlPath=" + config.Dir() + "/%r@%h:%p"}
	if v := os.Getenv("DOCKER_SSH_MUX_PERSIST"); v != "" {
		args = append(args, "-o", "ControlPersist="+v)
	}
	return args
}
