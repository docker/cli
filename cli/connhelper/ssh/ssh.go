// Package ssh provides the connection helper for ssh:// URL.
// Requires Docker 18.09 or later on the remote host.
package ssh

import (
	"net/url"

	"github.com/pkg/errors"
)

// New returns cmd and its args
func New(daemonURL string) (string, []string, error) {
	sp, err := parseSSHURL(daemonURL)
	if err != nil {
		return "", nil, errors.Wrap(err, "SSH host connection is not valid")
	}
	return "ssh", append(sp.Args(), []string{"--", "docker", "system", "dial-stdio"}...), nil
}

func parseSSHURL(daemonURL string) (*sshSpec, error) {
	u, err := url.Parse(daemonURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ssh" {
		return nil, errors.Errorf("expected scheme ssh, got %q", u.Scheme)
	}

	var sp sshSpec

	if u.User != nil {
		sp.user = u.User.Username()
		if _, ok := u.User.Password(); ok {
			return nil, errors.New("plain-text password is not supported")
		}
	}
	sp.host = u.Hostname()
	if sp.host == "" {
		return nil, errors.Errorf("no host specified")
	}
	sp.port = u.Port()
	if u.Path != "" {
		return nil, errors.Errorf("extra path after the host: %q", u.Path)
	}
	if u.RawQuery != "" {
		return nil, errors.Errorf("extra query after the host: %q", u.RawQuery)
	}
	if u.Fragment != "" {
		return nil, errors.Errorf("extra fragment after the host: %q", u.Fragment)
	}
	return &sp, err
}

type sshSpec struct {
	user string
	host string
	port string
}

func (sp *sshSpec) Args() []string {
	var args []string
	if sp.user != "" {
		args = append(args, "-l", sp.user)
	}
	if sp.port != "" {
		args = append(args, "-p", sp.port)
	}
	args = append(args, sp.host)
	return args
}
