package proxy

import (
	log "github.com/Sirupsen/logrus"

	"github.com/docker/engine-api-proxy/types"
)

// check that withLogging is a types.ReadWriteCloseCloseWriter
var _ types.ReadWriteCloseCloseWriter = &withLogging{}

// A ReadWriteCloseCloseWriter that decorates an underlying ReadWriteWriteCloser and
// logs all reads and writes.
type withLogging struct {
	label      string
	underlying types.ReadWriteCloseCloseWriter
}

func (t *withLogging) Read(buf []byte) (int, error) {
	count, err := t.underlying.Read(buf)
	log.Debugf("proxy %s -> %s\n", t.label, buf[:count])
	if err != nil {
		log.Debugf("proxy %s -> %s\n", t.label, err)
	}
	return count, err
}

func (t *withLogging) Write(buf []byte) (int, error) {
	log.Debugf("proxy %s <- %s\n", t.label, buf)
	return t.underlying.Write(buf)
}

func (t *withLogging) Close() error {
	log.Debugf("proxy %s <- EOF\n", t.label)
	return t.underlying.Close()
}

func (t *withLogging) CloseWrite() error {
	return t.underlying.CloseWrite()
}
