# Proxy server for Docker Engine API

An HTTP Proxy that listens on one socket, and forwards requests to
a Docker Engine API socket. The proxy can be configured with middleware which
may modify the the request and the response as it passes through the proxy.

