// Package sockets provides helper functions to create and configure Unix or TCP sockets.
package sockets

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"
)

const defaultTimeout = 10 * time.Second

// ErrProtocolNotAvailable is returned when a given transport protocol is not provided by the operating system.
var ErrProtocolNotAvailable = errors.New("protocol not available")

// ConfigureTransport configures the specified Transport according to the
// specified proto and addr.
// If the proto is unix (using a unix socket to communicate) or npipe the
// compression is disabled.
func ConfigureTransport(tr *http.Transport, proto, addr string) error {
	switch proto {
	case "unix":
		return configureUnixTransport(tr, proto, addr)
	case "npipe":
		return configureNpipeTransport(tr, proto, addr)
	default:
		tr.Proxy = TCPProxyFromEnvironment
		tr.Proxy = http.ProxyFromEnvironment
		dialer, err := DialerFromEnvironment(&net.Dialer{
			Timeout: defaultTimeout,
		})
		if err != nil {
			return err
		}
		tr.Dial = dialer.Dial //nolint: staticcheck // SA1019: tr.Dial is deprecated: Use DialContext instead
	}
	return nil
}

// TCPProxyFromEnvironment wraps http.ProxyFromEnvironment, to preserve the
// pre-go1.16 behavior for URLs using the 'tcp://' scheme. For other schemes,
// golang's standard behavior is preserved (and depends on the Go version used).
//
// Prior to go1.16, `https://` schemes would use HTTPS_PROXY, and any other
// scheme would use HTTP_PROXY. However, https://github.com/golang/net/commit/7b1cca2348c07eb09fef635269c8e01611260f9f
// (per a request in golang/go#40909) changed this behavior to only use
// HTTP_PROXY for `http://` schemes, no longer using a proxy for any other
// scheme.
//
// Docker uses the `tcp://` scheme as a default for API connections, to indicate
// that the API is not "purely" HTTP. Various parts in the code also *require*
// this scheme to be used. While we could change the default and allow http(s)
// schemes to be used, doing so will take time, taking into account that there
// are many installs in existence that have tcp:// configured as DOCKER_HOST.
//
// This function detects if the `tcp://` scheme is used; if it is, it creates
// a shallow copy of req, containing just the URL, and overrides the scheme with
// 'http', which should be sufficient to perform proxy detection.
// For other (non-'tcp://') schemes, http.ProxyFromEnvironment is called without
// altering the request.
func TCPProxyFromEnvironment(req *http.Request) (*url.URL, error) {
	if req.URL.Scheme != "tcp" {
		return http.ProxyFromEnvironment(req)
	}
	u := req.URL
	if u.Scheme == "tcp" {
		u.Scheme = "http"
	}
	return http.ProxyFromEnvironment(&http.Request{URL: u})
}
