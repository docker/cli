package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/docker/engine-api-proxy/types"
)

// ResponseRewriter is an interface that can be implemented by Middleware to
// modifying the response body
type ResponseRewriter interface {
	RewriteBody(*http.Response, io.ReadCloser) (int, io.ReadCloser, error)
}

// NopRewriter implements the ResponseRewriter interface but takes no action
type NopRewriter struct{}

// RewriteBody does nothing
func (n NopRewriter) RewriteBody(resp *http.Response, body io.ReadCloser) (int, io.ReadCloser) {
	return -1, body
}

// BackendDialer is an interface which provides a connection to a backend
type BackendDialer interface {
	Dial() (types.ReadWriteCloseCloseWriter, error)
}

// WriteFlusher extends the io.Writer interface with http.Flusher
type WriteFlusher interface {
	io.Writer
	http.Flusher
}

type writeFlusher struct {
	writer WriteFlusher
}

func (t *writeFlusher) Write(buf []byte) (int, error) {
	count, err := t.writer.Write(buf)
	t.writer.Flush()
	return count, err
}

func IsRawStreamUpgrade(resp *http.Response) bool {
	return (resp.StatusCode == 101 && resp.Header.Get("Upgrade") == "tcp") ||
		resp.Header.Get("Content-Type") == "application/vnd.docker.raw-stream"
}

func writeProxyError(writer http.ResponseWriter) {
	http.Error(writer, "Bad response from Docker Engine", 502)
}

type passthru struct {
	backendDialer BackendDialer
}

func newPassthru(backendDialer BackendDialer) *passthru {
	return &passthru{backendDialer: backendDialer}
}

func (p *passthru) Passthru(writer http.ResponseWriter, req *http.Request, rewriter ResponseRewriter, cancellable bool) error {
	log.Debugf("proxy >> %s %s\n", req.Method, AnonymizeURL(req.URL))

	// Connect to underlying service
	var underlying types.ReadWriteCloseCloseWriter
	underlying, err := p.backendDialer.Dial()
	if err != nil {
		return err
	}
	defer underlying.Close()

	if cancellable {
		if notifier, ok := writer.(http.CloseNotifier); ok {
			notify := notifier.CloseNotify()
			finished := make(chan struct{})
			defer close(finished)
			go func() {
				select {
				case <-notify:
					log.Debug("Cancel connection...")
					underlying.CloseWrite()
				case <-finished:
				}
			}()
		}
	}

	return p.doHandleHTTP(underlying, writer, req, rewriter)
}

func (p *passthru) doHandleHTTP(underlying types.ReadWriteCloseCloseWriter, writer http.ResponseWriter, req *http.Request, rewriter ResponseRewriter) error {
	underlying = &withLogging{label: "underlying", underlying: underlying}

	// Forward request to underlying
	requestErrors := make(chan error)
	go func() {
		requestErrors <- req.Write(underlying)
	}()

	// Read response
	resp, err := http.ReadResponse(bufio.NewReader(underlying), req)
	if err != nil {
		log.Warnf("error reading response from Docker: %s", err)
		writeProxyError(writer)
		return nil // ??
	}

	// Forward response to client
	isRaw := IsRawStreamUpgrade(resp)
	copyHeaders(writer.Header(), resp.Header)
	if isRaw {
		// Stop Go adding a chunked encoding header
		writer.Header().Set("Transfer-Encoding", "identity")
		writer.WriteHeader(resp.StatusCode)
		writer.(http.Flusher).Flush()

		// Attach console - hijack connection
		err := upgradeToRaw(writer, underlying)
		if err != nil {
			return err
		}
	} else {
		// Regular HTTP response
		newContentLength, body, err := rewriter.RewriteBody(resp, resp.Body)
		switch {
		case err != nil:
			writeProxyError(writer)
			return nil
		case newContentLength >= 0:
			writer.Header().Set(http.CanonicalHeaderKey("content-length"), fmt.Sprintf("%d", newContentLength))
		}
		writer.WriteHeader(resp.StatusCode)
		writer.(http.Flusher).Flush()

		_, err = io.Copy(&writeFlusher{writer: writer.(WriteFlusher)}, body)
		if err != nil {
			log.Warnf("error copying response body from Docker: %s", err)
		}
		err = resp.Body.Close()
		if err != nil {
			log.Warnf("error closing response body from Docker: %s", err)
		}
	}

	// Wait for request thread to finish if it's still going
	err = <-requestErrors
	if err != nil {
		log.Warnf("error forwarding client's request to Docker: %s", err)
	}
	log.Debugf("proxy << %s %s\n", req.Method, AnonymizeURL(req.URL))

	return nil
}

func upgradeToRaw(writer http.ResponseWriter, underlying io.ReadWriteCloser) error {
	log.Debug("Upgrading to raw stream")
	hj, ok := writer.(http.Hijacker)
	if !ok {
		panic("BUG: webserver doesn't support hijacking")
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return err
	}

	defer conn.Close()
	bufrw.Flush()
	done := make(chan bool)

	// Stream underlying -> conn
	go func() {
		var err error
		n := bufrw.Reader.Buffered()
		if n > 0 {
			buf := make([]byte, n)
			n, err := bufrw.Read(buf)
			if err != nil {
				panic(err)
			}
			_, err = conn.Write(buf[0:n])
		}
		if err != nil {
			log.Warnf("Error draining buffer: %s", err)
		} else {
			_, err := io.Copy(conn, underlying)
			if err != nil {
				log.Warnf("Error forwarding raw stream from container: %s", err)
			}
			connCloseWriter, ok := conn.(types.CloseWriter)
			if ok {
				err = connCloseWriter.CloseWrite()
				if err != nil {
					log.Warnf("Error closing raw stream from container: %s", err)
				}
			} else {
				log.Errorln("client connection is not a CloseWriter")
			}
		}
		done <- true
	}()

	// Stream underlying <- conn
	_, err = io.Copy(underlying, conn)
	if err != nil {
		log.Warnf("Error forwarding raw stream to container: %s", err)
	}

	underlyingCloseWriter, ok := underlying.(types.CloseWriter)
	if ok {
		err = underlyingCloseWriter.CloseWrite()
		if err != nil {
			log.Warnf("Error closing raw stream to container: %s", err)
		}
	} else {
		log.Errorln("underlying connection is not a CloseWriter")
	}
	<-done

	return nil
}

// Make dst have the same contents as src
func copyHeaders(dst, src http.Header) {
	// http://stackoverflow.com/questions/13812121/how-to-clear-a-map-in-go
	for k := range dst {
		delete(dst, k)
	}

	for k, v := range src {
		dst[k] = v
	}
}

// AnonymizeURL searches for `username` or `password` in the query string. If any
// of those is found, the whole query string is redacted.
// We could be nore precise but since the urls are only printed for debug purpose,
// it's ok.
func AnonymizeURL(u *url.URL) string {
	url := u.String()

	if strings.Contains(u.RawQuery, "username") || strings.Contains(u.RawQuery, "password") {
		return url[0:strings.Index(url, "?")] + "?[REDACTED]"
	}

	return url
}
