/* The Proxy package proxies HTTP and streaming requests to the Engine API.

The proxy listens for requests, runs any transformations that are configured
for the url and then passes the request along to the backend.
*/
package proxy

import (
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	"github.com/docker/engine-api-proxy/fakes"
	"github.com/docker/engine-api-proxy/types"
	"github.com/docker/go-connections/sockets"
	"github.com/pkg/errors"
)

// Options accepted by NewProxy for creating a proxy
type Options struct {
	Listen      string
	Backend     string
	SocketGroup string
	Routes      []MiddlewareRoute
}

// Proxy server which accepts requests and forwards them to a backend
type Proxy struct {
	listener net.Listener
	handler  http.Handler
}

// GetListener returns the proxy's listener.
// This is to be used when doing "in-process" proxying.
func (p *Proxy) GetListener() net.Listener {
	return p.listener
}

// NewInMemoryProxy creates an in-memory proxy for "in-process" use
func NewInMemoryProxy(opts Options) (*Proxy, error) {
	listener := fakes.NewListener()
	handler, err := newDefaultHandler(opts.Backend, opts.Routes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create handler")
	}
	return &Proxy{listener: listener, handler: handler}, nil
}

func NewProxy(opts Options) (*Proxy, error) {
	listener, err := newListener(opts.Listen, opts.SocketGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create listener")
	}
	handler, err := newDefaultHandler(opts.Backend, opts.Routes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create handler")
	}
	return &Proxy{listener: listener, handler: handler}, nil
}

func (p *Proxy) Start() error {
	log.Debugf("Proxy listening on %s", p.listener.Addr())
	return http.Serve(p.listener, p.handler)
}

func newDefaultHandler(addr string, routes []MiddlewareRoute) (http.Handler, error) {
	dialer, err := newBackendDialer(addr)
	if err != nil {
		return nil, err
	}
	return newHandlerFromMiddleware(routes, dialer)
}

func newBackendDialer(host string) (*backendDialer, error) {
	proto, addr, _, err := client.ParseHost(host)
	if err != nil {
		return nil, err
	}
	return &backendDialer{proto: proto, addr: addr}, nil
}

// TODO: move to new module
type backendDialer struct {
	proto string
	addr  string
}

// TODO: configurable timeout
func (s *backendDialer) Dial() (types.ReadWriteCloseCloseWriter, error) {
	// support Windows named-pipes
	if s.proto == "npipe" {
		conn, err := sockets.DialPipe(s.addr, 32*time.Second)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to connect to backend npipe socket")
		}
		resultConn, ok := conn.(types.ReadWriteCloseCloseWriter)
		if !ok {
			return nil, errors.Wrapf(err, "backend net.Conn does not implement the ReadWriteCloseCloseWriter interface")
		}
		return resultConn, nil
	}

	conn, err := net.DialTimeout(s.proto, s.addr, 32*time.Second)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to backend socket")
	}

	// when the backend dialer is connected to the actual Docker daemon
	// using a TCP connection, be sure to send keep-alive messages so
	// the connection doesn't timeout when no bytes are sent or received
	// for long periods of time.
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	resultConn, ok := conn.(types.ReadWriteCloseCloseWriter)
	if !ok {
		return nil, errors.Wrapf(err, "backend net.Conn does not implement the ReadWriteCloseCloseWriter interface")
	}
	return resultConn, nil
}
