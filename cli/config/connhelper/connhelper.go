// Package connhelper provides connection helpers.
// ConnectionHelper allows to connect to a remote host with custom stream provider binary.
//
// convention:
//   * filename MUST be `docker-connection-%s`
//   * called with args: {"connect", url}
//   * stderr can be used for logging purpose
package connhelper

import (
	"context"
	"io"
	"net"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ConnectionHelper allows to connect to a remote host with custom stream provider binary.
type ConnectionHelper struct {
	Dialer    func(ctx context.Context, network, addr string) (net.Conn, error)
	DummyHost string // dummy URL used for HTTP requests. e.g. "http://docker"
}

// GetConnectionHelper returns nil without error when no helper is registered for the scheme.
// host is like "ssh://me@server01:/var/run/docker.sock".
// cfg is like {"ssh": "ssh"}.
// prefix is like "docker-connection-".
func GetConnectionHelper(host string, cfg map[string]string, prefix string) (*ConnectionHelper, error) {
	path, err := lookupConnectionHelperPath(host, cfg, prefix)
	if path == "" || err != nil {
		return nil, err
	}
	dialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return newHelperConn(ctx, path, host, network, addr)
	}
	return &ConnectionHelper{
		Dialer:    dialer,
		DummyHost: "tcp://docker",
	}, nil
}

// lookupConnectionHelperPath returns an empty string without error when no helper is registered for the scheme.
func lookupConnectionHelperPath(host string, cfg map[string]string, prefix string) (string, error) {
	u, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		return "", nil // unregistered
	}
	helperName := cfg[u.Scheme]
	if helperName == "" {
		return "", nil // unregistered
	}
	if strings.Contains(helperName, string(filepath.Separator)) {
		return "", errors.Errorf("helper name (e.g. \"ssh\") should not contain path separator: %s", helperName)
	}
	return exec.LookPath(prefix + helperName)
}

func newHelperConn(ctx context.Context, helper string, host, dialNetwork, dialAddr string) (net.Conn, error) {
	var (
		c   helperConn
		err error
	)
	c.cmd = exec.CommandContext(ctx, helper, "connect", host)
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	c.cmd.Stderr = &logrusDebugWriter{
		prefix: "helper: ",
	}
	c.localAddr = dummyAddr{network: dialNetwork, s: "localhost"}
	c.remoteAddr = dummyAddr{network: dialNetwork, s: dialAddr}
	return &c, c.cmd.Start()
}

// helperConn implements net.Conn
type helperConn struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c *helperConn) Read(p []byte) (int, error) {
	return c.stdout.Read(p)
}

func (c *helperConn) Write(p []byte) (int, error) {
	return c.stdin.Write(p)
}

func (c *helperConn) Close() error {
	if err := c.stdin.Close(); err != nil {
		logrus.Warnf("error while closing stdin: %v", err)
	}
	if err := c.stdout.Close(); err != nil {
		logrus.Warnf("error while closing stdout: %v", err)
	}
	if err := c.cmd.Process.Kill(); err != nil {
		return err
	}
	_, err := c.cmd.Process.Wait()
	return err
}

func (c *helperConn) LocalAddr() net.Addr {
	return c.localAddr
}
func (c *helperConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}
func (c *helperConn) SetDeadline(t time.Time) error {
	logrus.Debugf("unimplemented call: SetDeadline(%v)", t)
	return nil
}
func (c *helperConn) SetReadDeadline(t time.Time) error {
	logrus.Debugf("unimplemented call: SetReadDeadline(%v)", t)
	return nil
}
func (c *helperConn) SetWriteDeadline(t time.Time) error {
	logrus.Debugf("unimplemented call: SetWriteDeadline(%v)", t)
	return nil
}

type dummyAddr struct {
	network string
	s       string
}

func (d dummyAddr) Network() string {
	return d.network
}

func (d dummyAddr) String() string {
	return d.s
}

type logrusDebugWriter struct {
	prefix string
}

func (w *logrusDebugWriter) Write(p []byte) (int, error) {
	logrus.Debugf("%s%s", w.prefix, string(p))
	return len(p), nil
}
